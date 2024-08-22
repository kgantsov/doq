package queue

import (
	"sync"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/rs/zerolog/log"
)

type QueueManager struct {
	db *badger.DB

	queues map[string]*BadgerPriorityQueue
	mu     sync.Mutex
}

func NewQueueManager(db *badger.DB) *QueueManager {
	return &QueueManager{db: db, queues: make(map[string]*BadgerPriorityQueue)}
}

func (qm *QueueManager) DeclareQueue(queue_name string) (*BadgerPriorityQueue, error) {
	q := NewBadgerPriorityQueue(qm.db, queue_name)
	return q, nil
}

func (qm *QueueManager) GetQueue(queue_name string) (*BadgerPriorityQueue, error) {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	q, ok := qm.queues[queue_name]
	if !ok {
		q, err := qm.DeclareQueue(queue_name)
		if err != nil {
			return nil, err
		}

		qm.queues[queue_name] = q

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
