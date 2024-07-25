package queue

import (
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/rs/zerolog/log"
)

type QueueManager struct {
	db *badger.DB
}

func NewQueueManager(db *badger.DB) *QueueManager {
	return &QueueManager{db: db}
}

func (qm *QueueManager) DeclareQueue(queue_name string) (Queue, error) {
	q := NewBadgerPriorityQueue(qm.db, queue_name)
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
