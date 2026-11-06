package storage

import (
	"bytes"
	"fmt"

	"github.com/dgraph-io/badger/v4"
	"github.com/hashicorp/raft"
	"github.com/kgantsov/doq/pkg/entity"
	"github.com/kgantsov/doq/pkg/errors"
	pb "github.com/kgantsov/doq/pkg/proto"
)

type BadgerStore struct {
	db *badger.DB
}

func NewBadgerStore(db *badger.DB) *BadgerStore {
	return &BadgerStore{db: db}
}

func (bpq *BadgerStore) ListQueues() ([]*entity.QueueConfig, error) {
	var queues []*entity.QueueConfig

	err := bpq.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()

		var qc *entity.QueueConfig
		var err error

		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			key := item.Key()

			if bytes.HasPrefix(key, []byte("queues:")) {
				err = item.Value(func(val []byte) error {
					qc, err = entity.QueueConfigFromBytes(val)
					return err
				})
				queues = append(queues, qc)
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return queues, nil
}

func (bpq *BadgerStore) GetMessages(queueName string, limit int, lastID uint64) ([]*entity.Message, error) {
	messages := make([]*entity.Message, 0, limit)

	err := bpq.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()

		var count int
		var err error

		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			key := item.Key()

			if bytes.HasPrefix(key, bpq.getMessagesPrefix(queueName)) {
				if count >= limit {
					break
				}
				err = item.Value(func(val []byte) error {
					msg, err := entity.MessageFromBytes(val)
					if err != nil {
						return err
					}
					if msg.ID > lastID {
						messages = append(messages, msg)
						count++
					}
					return nil
				})
				if err != nil {
					return err
				}
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return messages, nil
}

func (bpq *BadgerStore) getMessagesPrefix(queueName string) []byte {
	return []byte(fmt.Sprintf("messages:%s:", queueName))
}

func (bpq *BadgerStore) GetQueueKey(queueName string) []byte {
	return []byte(fmt.Sprintf("queues:%s:", queueName))
}

func (bpq *BadgerStore) GetMessagesKey(queueName string, id uint64) []byte {
	return addPrefix(bpq.getMessagesPrefix(queueName), uint64ToBytes(id))
}

func (s *BadgerStore) CreateQueue(queueType, queueName string, settings entity.QueueSettings) error {
	err := s.db.Update(func(txn *badger.Txn) error {
		config := &entity.QueueConfig{Name: queueName, Type: queueType, Settings: settings}

		data, err := config.ToBytes()
		if err != nil {
			return err
		}
		return txn.Set(s.GetQueueKey(queueName), data)
	})
	if err != nil {
		return err
	}
	return nil
}

func (s *BadgerStore) UpdateQueue(queueType, queueName string, settings entity.QueueSettings) error {
	err := s.db.Update(func(txn *badger.Txn) error {
		config := &entity.QueueConfig{Name: queueName, Type: queueType, Settings: settings}

		data, err := config.ToBytes()
		if err != nil {
			return err
		}
		return txn.Set(s.GetQueueKey(queueName), data)
	})
	if err != nil {
		return err
	}
	return nil
}

// DeleteQueue removes a queue's config and all of its messages. Messages are
// deleted in batches, so a queue of any size fits within Badger's transaction
// limits.
func (s *BadgerStore) DeleteQueue(queueName string) error {
	var keys [][]byte
	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()

		prefix := s.getMessagesPrefix(queueName)
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			keys = append(keys, it.Item().KeyCopy(nil))
		}
		return nil
	})
	if err != nil {
		return err
	}

	wb := s.db.NewWriteBatch()
	defer wb.Cancel()
	for _, key := range keys {
		if err := wb.Delete(key); err != nil {
			return err
		}
	}
	if err := wb.Delete(s.GetQueueKey(queueName)); err != nil {
		return err
	}
	return wb.Flush()
}

// DeleteAll removes every queue and message. Raft's log and stable store
// share this database under their own prefixes and are left alone.
func (s *BadgerStore) DeleteAll() error {
	return s.db.DropPrefix([]byte("queues:"), []byte("messages:"))
}

func (s *BadgerStore) Enqueue(
	queueName string,
	id uint64,
	group string,
	priority int64,
	content string,
	metadata map[string]string,
) (*entity.Message, error) {
	msg := &entity.Message{
		ID:       id,
		Group:    group,
		Priority: priority,
		Content:  content,
		Metadata: metadata,
	}

	err := s.db.Update(func(txn *badger.Txn) error {
		data, err := msg.ToBytes()
		if err != nil {
			return err
		}
		return txn.Set(s.GetMessagesKey(queueName, msg.ID), data)
	})
	if err != nil {
		return msg, err
	}

	return msg, nil
}

func (s *BadgerStore) Dequeue(queueName string, id uint64, ack bool) (*entity.Message, error) {
	var msg *entity.Message

	err := s.db.Update(func(txn *badger.Txn) error {
		item, err := txn.Get(s.GetMessagesKey(queueName, id))
		if err != nil {
			return err
		}

		item.Value(func(val []byte) error {
			msg, err = entity.MessageFromBytes(val)
			return err
		})
		if err != nil {
			return err
		}

		// Handing the message out is a delivery, so this is the one place the
		// counter moves. DeliveryLimit is then checked on the failure paths
		// (Nack, AckTimeout) against a count that is already truthful: 1 while
		// the first delivery is in flight.
		msg.DeliveryCount++

		if ack {
			// in case of autoAck, we need to remove the message from the queue
			return txn.Delete(s.GetMessagesKey(queueName, id))
		}

		data, err := msg.ToBytes()
		if err != nil {
			return err
		}

		return txn.Set(s.GetMessagesKey(queueName, id), data)
	})

	if err != nil {
		return msg, err
	}

	return msg, nil
}

func (s *BadgerStore) Get(queueName string, id uint64) (*entity.Message, error) {
	var msg *entity.Message

	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(s.GetMessagesKey(queueName, id))
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return errors.ErrMessageNotFound
			}
			return err
		}

		return item.Value(func(val []byte) error {
			msg, err = entity.MessageFromBytes(val)
			return err
		})
	})
	if err != nil {
		return nil, err
	}

	return msg, nil
}

func (s *BadgerStore) Delete(queueName string, id uint64) (*entity.Message, error) {
	var msg *entity.Message

	err := s.db.Update(func(txn *badger.Txn) error {
		item, err := txn.Get(s.GetMessagesKey(queueName, id))
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return errors.ErrMessageNotFound
			}
			return err
		}

		err = item.Value(func(val []byte) error {
			msg, err = entity.MessageFromBytes(val)
			if err != nil {
				return err
			}

			return nil
		})

		if err != nil {
			return err
		}

		err = txn.Delete(s.GetMessagesKey(queueName, msg.ID))
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return errors.ErrMessageNotFound
			}
		}

		return nil
	})
	if err != nil {
		return msg, err
	}

	return msg, nil
}

func (s *BadgerStore) Ack(queueName string, id uint64) error {
	err := s.db.Update(func(txn *badger.Txn) error {
		err := txn.Delete(s.GetMessagesKey(queueName, id))
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *BadgerStore) UpdatePriority(queueName string, id uint64, priority int64) (*entity.Message, error) {
	var msg *entity.Message

	err := s.db.Update(func(txn *badger.Txn) error {
		item, err := txn.Get(s.GetMessagesKey(queueName, id))
		if err != nil {
			return err
		}

		err = item.Value(func(val []byte) error {
			msg, err = entity.MessageFromBytes(val)
			return err
		})

		if err != nil {
			return err
		}

		msg.UpdatePriority(priority)

		data, err := msg.ToBytes()
		if err != nil {
			return err
		}
		err = txn.Set(s.GetMessagesKey(queueName, id), data)

		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return msg, err
	}

	return msg, nil
}

func (s *BadgerStore) UpdateMessage(
	queueName string,
	id uint64,
	priority int64,
	content string,
	metadata map[string]string,
) error {
	return s.db.Update(func(txn *badger.Txn) error {
		item, err := txn.Get(s.GetMessagesKey(queueName, id))
		if err != nil {
			return err
		}

		var msg *entity.Message
		err = item.Value(func(val []byte) error {
			msg, err = entity.MessageFromBytes(val)
			return err
		})
		if err != nil {
			return err
		}

		if priority != 0 {
			msg.Priority = priority
		}

		if content != "" {
			msg.Content = content
		}

		if metadata != nil {
			msg.Metadata = metadata
		}

		data, err := msg.ToBytes()
		if err != nil {
			return err
		}

		return txn.Set(s.GetMessagesKey(queueName, id), data)
	})
}

// RestoreMessage writes a message exactly as it was persisted, including the
// fields the normal Enqueue path does not accept. Snapshot restore uses it so
// counters such as DeliveryCount survive a restore instead of resetting.
func (s *BadgerStore) RestoreMessage(queueName string, msg *entity.Message) error {
	return s.db.Update(func(txn *badger.Txn) error {
		data, err := msg.ToBytes()
		if err != nil {
			return err
		}

		return txn.Set(s.GetMessagesKey(queueName, msg.ID), data)
	})
}

// PersistSnapshot writes the queue's config, then its unacked markers, then its
// messages. Restore relies on that order: it must know which messages are in
// flight before it reads them.
func (s *BadgerStore) PersistSnapshot(
	queueConfig *entity.QueueConfig,
	unacked []*pb.UnackedMessage,
	sink raft.SnapshotSink,
	txn *badger.Txn,
) error {
	it := txn.NewIterator(badger.DefaultIteratorOptions)
	defer it.Close()

	snapshotItem := &pb.SnapshotItem{
		Item: &pb.SnapshotItem_Queue{
			Queue: queueConfig.ToProto(),
		},
	}

	if err := WriteSnapshotItem(sink, snapshotItem); err != nil {
		return fmt.Errorf("failed to write message snapshot item: %v", err)
	}

	for _, u := range unacked {
		snapshotItem := &pb.SnapshotItem{
			Item: &pb.SnapshotItem_UnackedMessage{UnackedMessage: u},
		}
		if err := WriteSnapshotItem(sink, snapshotItem); err != nil {
			return fmt.Errorf("failed to write unacked message snapshot item: %v", err)
		}
	}

	prefix := s.getMessagesPrefix(queueConfig.Name)
	for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
		item := it.Item()

		err := item.Value(func(val []byte) error {
			msg, err := entity.MessageProtoFromBytes(val)
			if err != nil {
				return err
			}

			snapshotItem := &pb.SnapshotItem{
				Item: &pb.SnapshotItem_Message{Message: msg},
			}
			if err := WriteSnapshotItem(sink, snapshotItem); err != nil {
				return fmt.Errorf("failed to write message snapshot item: %v", err)
			}

			return nil
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *BadgerStore) LoadQueue(queueName string) (*entity.QueueConfig, error) {
	var qc *entity.QueueConfig

	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(s.GetQueueKey(queueName))
		if err != nil {
			return err
		}

		err = item.Value(func(val []byte) error {
			qc, err = entity.QueueConfigFromBytes(val)
			return err
		})

		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return qc, err
	}

	return qc, nil
}
