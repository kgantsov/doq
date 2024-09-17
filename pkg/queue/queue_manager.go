package queue

import (
	"fmt"
	"sync"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/kgantsov/doq/pkg/config"
	"github.com/rs/zerolog/log"
)

var ErrQueueNotFound = fmt.Errorf("queue not found")

type QueueManager struct {
	db     *badger.DB
	config *config.Config

	queues map[string]*BadgerPriorityQueue
	mu     sync.Mutex
}

func NewQueueManager(db *badger.DB, cfg *config.Config) *QueueManager {
	return &QueueManager{
		db:     db,
		config: cfg,
		queues: make(map[string]*BadgerPriorityQueue),
	}
}

func (qm *QueueManager) Create(queueType, queueName string) (*BadgerPriorityQueue, error) {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	q, ok := qm.queues[queueName]
	if ok {
		return q, nil
	}

	q = NewBadgerPriorityQueue(qm.db, qm.config)
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
		q = NewBadgerPriorityQueue(qm.db, qm.config)
		q.Load(queueName, false)
	}

	err := q.Delete()
	if err != nil {
		return err
	}

	delete(qm.queues, queueName)

	return nil
}

func (qm *QueueManager) GetQueue(queueName string) (*BadgerPriorityQueue, error) {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	q, ok := qm.queues[queueName]
	if !ok {
		q = NewBadgerPriorityQueue(qm.db, qm.config)
		err := q.Load(queueName, true)
		if err != nil {
			return nil, ErrQueueNotFound
		}
		return q, nil
	}
	return q, nil
}

func (qm *QueueManager) RunValueLogGC() {

	ticker := time.NewTicker(time.Duration(qm.config.Storage.GCInterval) * time.Second)
	defer ticker.Stop()

	log.Debug().Msg("Started running value GC")

	for range ticker.C {
		log.Debug().Msg("Running value GC")
	again:
		err := qm.db.RunValueLogGC(qm.config.Storage.GCDiscardRatio)
		if err == nil {
			goto again
		}
	}
}
