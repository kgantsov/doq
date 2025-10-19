package storage

import (
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

func (s *BadgerStore) DeleteQueue(queueName string) error {
	err := s.db.Update(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = true
		it := txn.NewIterator(opts)
		defer it.Close()

		prefix := s.getMessagesPrefix(queueName)
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			return txn.Delete(item.Key())
		}

		err := txn.Delete(s.GetQueueKey(queueName))
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

		if ack {
			// in case of autoAck, we need to remove the message from the queue
			return txn.Delete(s.GetMessagesKey(queueName, id))
		}

		return nil
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

func (s *BadgerStore) PersistSnapshot(
	queueConfig *entity.QueueConfig, sink raft.SnapshotSink, txn *badger.Txn,
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
