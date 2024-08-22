package queue

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/rs/zerolog/log"
)

var ErrEmptyQueue = fmt.Errorf("queue is empty")
var ErrMessageNotFound = fmt.Errorf("message not found")

type Message struct {
	ID       uint64
	Priority int64
	Content  string
}

func (m *Message) ToBytes() ([]byte, error) {
	return json.Marshal(m)
}

func (m *Message) UpdatePriority(newPriority int64) {
	m.Priority = newPriority
}

func MessageFromBytes(data []byte) (*Message, error) {
	var msg Message
	err := json.Unmarshal(data, &msg)
	return &msg, err
}

type BadgerPriorityQueue struct {
	queueName string

	pq        Queue
	db        *badger.DB
	sequesnce *badger.Sequence

	mu sync.Mutex

	ackQueue   Queue
	ackQueueMu sync.Mutex
}

func NewBadgerPriorityQueue(db *badger.DB, queueName string) *BadgerPriorityQueue {
	bpq := &BadgerPriorityQueue{
		pq:        NewDelayedPriorityQueue(true),
		db:        db,
		queueName: queueName,
		ackQueue:  NewDelayedPriorityQueue(false),
	}

	seq, err := bpq.db.GetSequence(bpq.getQueueSequenceKey(), 1)

	if err != nil {
		log.Warn().Err(err).Msgf("Failed to get sequence: %s", err)
	}

	defer seq.Release()

	bpq.sequesnce = seq

	return bpq
}

func (bpq *BadgerPriorityQueue) getQueuePrefix() []byte {
	return addPrefix([]byte("queues:"), []byte(bpq.queueName))
}

func (bpq *BadgerPriorityQueue) getQueueSequenceKey() []byte {
	return addPrefix([]byte("sequences:"), []byte(bpq.queueName))
}

func (bpq *BadgerPriorityQueue) GetKey(id uint64) []byte {
	return addPrefix(bpq.getQueuePrefix(), uint64ToBytes(id))
}

func (bpq *BadgerPriorityQueue) GetNextID() (uint64, error) {
	num, err := bpq.sequesnce.Next()

	if err != nil {
		return 0, err
	}

	return num, nil
}

func (bpq *BadgerPriorityQueue) Enqueue(priority int64, content string) (*Message, error) {
	bpq.mu.Lock()
	defer bpq.mu.Unlock()

	nextID, err := bpq.GetNextID()
	if err != nil {
		return &Message{}, err
	}

	msg := &Message{
		ID:       nextID,
		Priority: priority,
		Content:  content,
	}

	queueItem := &Item{
		ID:       nextID,
		Priority: priority,
	}

	err = bpq.db.Update(func(txn *badger.Txn) error {
		data, err := msg.ToBytes()
		if err != nil {
			return err
		}
		return txn.Set(bpq.GetKey(msg.ID), data)
	})
	if err != nil {
		return msg, err
	}

	bpq.pq.Enqueue("default", queueItem)
	return msg, nil
}

func (bpq *BadgerPriorityQueue) Dequeue(ack bool) (*Message, error) {
	bpq.mu.Lock()
	defer bpq.mu.Unlock()

	if bpq.pq.Len() == 0 {
		return nil, ErrEmptyQueue
	}

	queueItem := bpq.pq.Dequeue()
	if queueItem == nil {
		return nil, ErrEmptyQueue
	}

	var msg *Message

	err := bpq.db.Update(func(txn *badger.Txn) error {
		item, err := txn.Get(bpq.GetKey(queueItem.ID))
		if err != nil {
			return err
		}

		item.Value(func(val []byte) error {
			msg, err = MessageFromBytes(val)
			return err
		})

		if ack {
			// in case of autoAck, we need to remove the message from the queue
			return txn.Delete(bpq.GetKey(queueItem.ID))
		} else {
			// in case of manual ack, we need to keep the message in the queue
			// so we can ack it later and add it to the ackQueue
			bpq.ackQueueMu.Lock()
			defer bpq.ackQueueMu.Unlock()

			bpq.ackQueue.Enqueue(
				"default",
				&Item{
					ID:       queueItem.ID,
					Priority: time.Now().UTC().Add(time.Duration(5) * time.Minute).Unix(),
				},
			)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return msg, nil
}

func (bpq *BadgerPriorityQueue) GetByID(id uint64) (*Message, error) {
	bpq.mu.Lock()
	defer bpq.mu.Unlock()

	var msg *Message

	queueItem := bpq.pq.GetByID("default", id)

	err := bpq.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(bpq.GetKey(queueItem.ID))
		if err != nil {
			return err
		}

		return item.Value(func(val []byte) error {
			msg, err = MessageFromBytes(val)
			return err
		})
	})
	if err != nil {
		return nil, err
	}

	return msg, nil
}

func (bpq *BadgerPriorityQueue) UpdatePriority(id uint64, newPriority int64) error {
	bpq.mu.Lock()
	defer bpq.mu.Unlock()

	queueItem := bpq.pq.GetByID("default", id)

	// Update BadgerDB
	err := bpq.db.Update(func(txn *badger.Txn) error {
		var msg *Message
		item, err := txn.Get(bpq.GetKey(id))
		if err != nil {
			return err
		}

		err = item.Value(func(val []byte) error {
			msg, err = MessageFromBytes(val)
			return err
		})

		if err != nil {
			return err
		}

		msg.UpdatePriority(newPriority)

		data, err := msg.ToBytes()
		if err != nil {
			return err
		}
		return txn.Set(bpq.GetKey(queueItem.ID), data)
	})
	if err != nil {
		return err
	}

	// Update in-memory heap
	queueItem.UpdatePriority(newPriority)
	bpq.pq.UpdatePriority("default", id, newPriority)
	return nil
}

func (bpq *BadgerPriorityQueue) Ack(id uint64) error {
	bpq.ackQueueMu.Lock()
	defer bpq.ackQueueMu.Unlock()

	queueItem := bpq.ackQueue.GetByID("default", id)

	if queueItem == nil {
		return ErrMessageNotFound
	}

	err := bpq.db.Update(func(txn *badger.Txn) error {
		err := txn.Delete(bpq.GetKey(queueItem.ID))
		if err != nil {
			return err
		}
		bpq.ackQueue.DeleteByID("default", queueItem.ID)

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func (bpq *BadgerPriorityQueue) Len() int {
	bpq.mu.Lock()
	defer bpq.mu.Unlock()

	return int(bpq.pq.Len())
}
