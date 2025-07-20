package queue

import (
	"sort"
	"sync"

	"github.com/hashicorp/raft"
	"github.com/kgantsov/doq/pkg/config"
	"github.com/kgantsov/doq/pkg/entity"
	"github.com/kgantsov/doq/pkg/errors"
	"github.com/kgantsov/doq/pkg/metrics"
	"github.com/kgantsov/doq/pkg/storage"
)

type QueueManager struct {
	store             storage.Store
	config            *config.Config
	PrometheusMetrics *metrics.PrometheusMetrics

	queues map[string]*Queue
	mu     sync.Mutex
}

func NewQueueManager(store storage.Store, cfg *config.Config, metrics *metrics.PrometheusMetrics) *QueueManager {
	return &QueueManager{
		store:             store,
		config:            cfg,
		queues:            make(map[string]*Queue),
		PrometheusMetrics: metrics,
	}
}

func (qm *QueueManager) PersistSnapshot(sink raft.SnapshotSink) error {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	for _, queue := range qm.queues {
		err := queue.PersistSnapshot(sink)
		if err != nil {
			return err
		}
	}

	return nil
}

func (qm *QueueManager) CreateQueue(queueType, queueName string, settings entity.QueueSettings) (*Queue, error) {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	q, ok := qm.queues[queueName]
	if ok {
		return q, nil
	}

	q = NewQueue(qm.store, qm.config, qm.PrometheusMetrics)
	err := q.Create(queueType, queueName, settings)

	if err != nil {
		return nil, err
	}

	qm.queues[queueName] = q

	return q, nil
}

func (qm *QueueManager) DeleteQueue(queueName string) error {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	q, ok := qm.queues[queueName]
	if !ok {
		q = NewQueue(qm.store, qm.config, qm.PrometheusMetrics)
		q.Load(queueName)
	}

	err := q.DeleteQueue()
	if err != nil {
		return err
	}

	delete(qm.queues, queueName)

	return nil
}

func (qm *QueueManager) GetQueue(queueName string) (*Queue, error) {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	q, ok := qm.queues[queueName]
	if !ok {
		q = NewQueue(qm.store, qm.config, qm.PrometheusMetrics)
		err := q.Load(queueName)
		if err != nil {
			return nil, errors.ErrQueueNotFound
		}

		qm.queues[queueName] = q

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

	sort.Slice(queues, func(i, j int) bool {
		return queues[i].Name < queues[j].Name
	})

	return queues
}
