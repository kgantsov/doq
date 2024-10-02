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
	db                *badger.DB
	config            *config.Config
	PrometheusMetrics *PrometheusMetrics

	queues map[string]*BadgerPriorityQueue
	mu     sync.Mutex
}

func NewQueueManager(db *badger.DB, cfg *config.Config, metrics *PrometheusMetrics) *QueueManager {
	return &QueueManager{
		db:                db,
		config:            cfg,
		queues:            make(map[string]*BadgerPriorityQueue),
		PrometheusMetrics: metrics,
	}
}

func (qm *QueueManager) LoadQueues() {
	prefix := []byte("queues:")
	err := qm.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10
		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			key := item.Key()
			queueName := string(key[len(prefix) : len(key)-1])

			log.Debug().Msgf("Loading queue: %s", queueName)

			q := NewBadgerPriorityQueue(qm.db, qm.config, qm.PrometheusMetrics)
			err := q.Load(queueName, true)
			if err != nil {
				log.Error().Err(err).Str("queue", queueName).Msg("Failed to load queue")
			}
			qm.queues[queueName] = q
		}

		return nil
	})

	if err != nil {
		log.Error().Err(err).Msg("Failed to load queues")
	}
}

func (qm *QueueManager) Create(queueType, queueName string) (*BadgerPriorityQueue, error) {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	q, ok := qm.queues[queueName]
	if ok {
		return q, nil
	}

	q = NewBadgerPriorityQueue(qm.db, qm.config, qm.PrometheusMetrics)
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
		q = NewBadgerPriorityQueue(qm.db, qm.config, qm.PrometheusMetrics)
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
		q = NewBadgerPriorityQueue(qm.db, qm.config, qm.PrometheusMetrics)
		err := q.Load(queueName, true)
		if err != nil {
			return nil, ErrQueueNotFound
		}
		return q, nil
	}
	return q, nil
}

func (qm *QueueManager) GetQueues() []*QueueInfo {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	queues := make([]*QueueInfo, 0, len(qm.queues))
	for _, queue := range qm.queues {
		queues = append(queues, queue.GetStats())
	}

	return queues
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
