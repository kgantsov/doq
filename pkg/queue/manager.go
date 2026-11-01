package queue

import (
	"sort"
	"sync"

	"github.com/dgraph-io/badger/v4"
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

	ackTimeoutHandler AckTimeoutHandler
}

// NewQueueManager creates a queue manager. ackTimeoutHandler is called for every
// unacked message whose ack deadline passes (see AckTimeoutHandler); under Raft
// it must replicate an AckTimeout command. With a nil handler, ack deadlines
// never expire.
func NewQueueManager(
	store storage.Store,
	cfg *config.Config,
	metrics *metrics.PrometheusMetrics,
	ackTimeoutHandler AckTimeoutHandler,
) *QueueManager {
	return &QueueManager{
		store:             store,
		config:            cfg,
		queues:            make(map[string]*Queue),
		PrometheusMetrics: metrics,
		ackTimeoutHandler: ackTimeoutHandler,
	}
}

func (qm *QueueManager) newQueue() *Queue {
	q := NewQueue(qm.store, qm.config, qm.PrometheusMetrics)
	q.ackTimeoutHandler = qm.ackTimeoutHandler
	return q
}

// CaptureSnapshotState returns every queue's in-memory snapshot state, keyed by
// queue name. See Queue.CaptureSnapshotState.
func (qm *QueueManager) CaptureSnapshotState() map[string]*SnapshotState {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	states := make(map[string]*SnapshotState, len(qm.queues))
	for name, queue := range qm.queues {
		states[name] = queue.CaptureSnapshotState()
	}
	return states
}

func (qm *QueueManager) PersistSnapshot(
	sink raft.SnapshotSink, txn *badger.Txn, states map[string]*SnapshotState,
) error {
	qm.mu.Lock()
	queues := qm.queues
	qm.mu.Unlock()

	for name, queue := range queues {
		err := queue.PersistSnapshot(sink, txn, states[name])
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

	q = qm.newQueue()
	err := q.CreateQueue(queueType, queueName, settings)

	if err != nil {
		return nil, err
	}

	qm.queues[queueName] = q

	return q, nil
}

func (qm *QueueManager) UpdateQueue(queueName string, settings entity.QueueSettings) error {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	q, ok := qm.queues[queueName]
	if !ok {
		q = qm.newQueue()
		q.Load(queueName)
	}

	err := q.UpdateQueue(settings)
	if err != nil {
		return err
	}

	qm.queues[queueName] = q

	return nil
}

func (qm *QueueManager) DeleteQueue(queueName string) error {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	q, ok := qm.queues[queueName]
	if !ok {
		q = qm.newQueue()
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
		q = qm.newQueue()
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
