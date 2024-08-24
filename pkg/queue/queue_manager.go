package queue

import (
	"fmt"
	"sync"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/rs/zerolog/log"
)

var ErrQueueNotFound = fmt.Errorf("queue not found")

type QueueManager struct {
	db *badger.DB

	queues map[string]*BadgerPriorityQueue
	mu     sync.Mutex
}

func NewQueueManager(db *badger.DB) *QueueManager {
	return &QueueManager{db: db, queues: make(map[string]*BadgerPriorityQueue)}
}

func (qm *QueueManager) Create(queueType, queueName string) (*BadgerPriorityQueue, error) {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	q, ok := qm.queues[queueName]
	if ok {
		return q, nil
	}

	q = NewBadgerPriorityQueue(qm.db)
	err := q.Create(queueType, queueName)

	if err != nil {
		return nil, err
	}

	qm.queues[queueName] = q

	return q, nil
}

func (qm *QueueManager) Delete(queueName string) error {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	q, ok := qm.queues[queueName]
	if !ok {
		q = NewBadgerPriorityQueue(qm.db)
		q.Load(queueName, false)
	}

	return q.Delete()
}

func (qm *QueueManager) GetQueue(queueName string) (*BadgerPriorityQueue, error) {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	q, ok := qm.queues[queueName]
	if !ok {
		q = NewBadgerPriorityQueue(qm.db)
		err := q.Load(queueName, true)
		if err != nil {
			return nil, ErrQueueNotFound
		}
		return q, nil
	}
	return q, nil
}

func (qm *QueueManager) RunValueLogGC() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	log.Debug().Msg("Started running value GC")

	for range ticker.C {
		log.Debug().Msg("Running value GC")
	again:
		err := qm.db.RunValueLogGC(0.7)
		if err == nil {
			goto again
		}
	}
}
