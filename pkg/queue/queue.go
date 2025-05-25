package queue

import (
	"sync"
	"time"

	"github.com/hashicorp/raft"
	"github.com/kgantsov/doq/pkg/config"
	"github.com/kgantsov/doq/pkg/entity"
	"github.com/kgantsov/doq/pkg/errors"
	"github.com/kgantsov/doq/pkg/metrics"
	"github.com/kgantsov/doq/pkg/queue/memory"
	"github.com/kgantsov/doq/pkg/storage"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
)

type QueueInfo struct {
	Name    string
	Type    string
	Stats   *metrics.Stats
	Ready   int64
	Unacked int64
	Total   int64
}

type Queue struct {
	config *entity.QueueConfig
	cfg    *config.Config

	PrometheusMetrics *metrics.PrometheusMetrics
	stats             *metrics.QueueStats

	queue memory.MemoryQueue
	store storage.Store

	mu sync.Mutex

	ackQueueMonitoringChan chan struct{}
	ackQueue               memory.MemoryQueue
}

func NewQueue(
	store storage.Store, cfg *config.Config, promMetrics *metrics.PrometheusMetrics,
) *Queue {

	bpq := &Queue{
		store:             store,
		cfg:               cfg,
		ackQueue:          memory.NewDelayedMemoryQueue(false),
		PrometheusMetrics: promMetrics,
		stats:             metrics.NewQueueStats(cfg.Queue.QueueStats.WindowSide),
	}

	return bpq
}

func (q *Queue) Init(queueType, queueName string) error {
	var queue memory.MemoryQueue
	if queueType == "fair" {
		queue = memory.NewFairMemoryQueue()
		// queue = memory.NewFairAckMemoryQueue()
	} else {
		queue = memory.NewDelayedMemoryQueue(true)
	}
	q.config = &entity.QueueConfig{Name: queueName, Type: queueType}
	q.queue = queue

	q.StartAckQueueMonitoring()

	go q.stats.Start()

	return nil
}

func (q *Queue) GetStats() *QueueInfo {
	return &QueueInfo{
		Name:    q.config.Name,
		Type:    q.config.Type,
		Stats:   q.stats.GetRPS(),
		Ready:   int64(q.queue.Len()),
		Unacked: int64(q.ackQueue.Len()),
		Total:   int64(q.queue.Len() + q.ackQueue.Len()),
	}
}

func (q *Queue) updatePrometheusQueueSizes() {
	if q.cfg.Prometheus.Enabled {
		readyMessages := float64(q.queue.Len())
		unackedMessages := float64(q.ackQueue.Len())

		q.PrometheusMetrics.Messages.With(
			prometheus.Labels{"queue_name": q.config.Name},
		).Set(readyMessages + unackedMessages)
		q.PrometheusMetrics.UnackedMessages.With(
			prometheus.Labels{"queue_name": q.config.Name},
		).Set(unackedMessages)
		q.PrometheusMetrics.ReadyMessages.With(
			prometheus.Labels{"queue_name": q.config.Name},
		).Set(readyMessages)
	}
}

func (q *Queue) monitorAckQueue() {
	ticker := time.NewTicker(
		time.Duration(q.cfg.Queue.AcknowledgementCheckInterval) * time.Second,
	)

	for {
		select {
		case <-ticker.C:
			for {
				item := q.ackQueue.Dequeue(false)
				if item == nil {
					break
				}

				message, err := q.Get(item.ID)
				if err != nil {
					log.Error().Err(err).Msgf("Failed to get message by ID: %d", item.ID)
					continue
				}

				queueItem := &memory.Item{
					ID:       message.ID,
					Priority: message.Priority,
					Group:    message.Group,
				}

				q.queue.Enqueue(message.Group, queueItem)
				log.Debug().Msgf("Re-enqueued message: %d", message.ID)
			}
		case <-q.ackQueueMonitoringChan:
			return
		}
	}
}

func (q *Queue) StartAckQueueMonitoring() {
	q.ackQueueMonitoringChan = make(chan struct{})
	go q.monitorAckQueue()
}

func (q *Queue) StopAckQueueMonitoring() {
	close(q.ackQueueMonitoringChan)
}

func (q *Queue) Create(queueType, queueName string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.config = &entity.QueueConfig{Name: queueName, Type: queueType}

	err := q.store.CreateQueue(queueType, queueName)
	if err != nil {
		return err
	}

	return q.Init(queueType, queueName)
}

func (q *Queue) DeleteQueue() error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.config == nil {
		return errors.ErrQueueNotFound
	}

	err := q.store.DeleteQueue(q.config.Name)
	if err != nil {
		return err
	}

	q.StopAckQueueMonitoring()

	q.stats.Stop()

	q.updatePrometheusQueueSizes()

	return nil
}

func (q *Queue) Load(queueName string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	qc, err := q.store.LoadQueue(queueName)
	if err != nil {
		return err
	}

	err = q.Init(qc.Type, qc.Name)
	if err != nil {
		return err
	}

	q.updatePrometheusQueueSizes()

	return nil
}

func (q *Queue) Enqueue(
	id uint64,
	group string,
	priority int64,
	content string,
	metadata map[string]string,
) (*entity.Message, error) {
	if q.config.Type != "fair" {
		group = "default"
	}

	queueItem := &memory.Item{
		ID:       id,
		Group:    group,
		Priority: priority,
	}

	msg, err := q.store.Enqueue(
		q.config.Name, queueItem.ID, queueItem.Group, queueItem.Priority, content, metadata,
	)
	if err != nil {
		return msg, err
	}

	q.queue.Enqueue(msg.Group, queueItem)

	q.stats.IncrementEnqueue()

	if q.cfg.Prometheus.Enabled {
		q.PrometheusMetrics.EnqueueTotal.With(prometheus.Labels{"queue_name": q.config.Name}).Inc()

		q.updatePrometheusQueueSizes()
	}

	return msg, nil
}

func (q *Queue) Dequeue(ack bool) (*entity.Message, error) {
	if q.queue.Len() == 0 {
		return nil, errors.ErrEmptyQueue
	}

	queueItem := q.queue.Dequeue(ack)
	if queueItem == nil {
		return nil, errors.ErrEmptyQueue
	}

	msg, err := q.store.Dequeue(q.config.Name, queueItem.ID, ack)

	if err != nil {
		return nil, err
	}

	if !ack {
		q.ackQueue.Enqueue(
			"default",
			&memory.Item{
				ID: queueItem.ID,
				Priority: time.Now().UTC().Add(
					time.Duration(q.cfg.Queue.AcknowledgementTimeout) * time.Second,
				).Unix(),
				Group: msg.Group,
			},
		)
	}

	if q.config.Type == "fair" && ack {
		q.queue.UpdateWeights(msg.Group, msg.ID)
	}

	q.stats.IncrementDequeue()
	if q.cfg.Prometheus.Enabled {
		q.PrometheusMetrics.DequeueTotal.With(prometheus.Labels{"queue_name": q.config.Name}).Inc()

		q.updatePrometheusQueueSizes()
	}

	return msg, nil
}

func (q *Queue) Get(id uint64) (*entity.Message, error) {
	return q.store.Get(q.config.Name, id)
}

func (q *Queue) Delete(id uint64) error {
	msg, err := q.store.Delete(q.config.Name, id)
	if err != nil {
		return err
	}

	group := msg.Group
	q.queue.Delete(group, id)
	q.ackQueue.Delete(group, id)

	return nil
}

func (q *Queue) UpdatePriority(id uint64, newPriority int64) error {
	msg, err := q.store.Get(q.config.Name, id)
	if err != nil {
		return err
	}

	group := msg.Group
	queueItem := q.queue.Get(group, id)

	if queueItem == nil {
		queueItem = q.ackQueue.Get(group, id)
		if queueItem == nil {
			return errors.ErrMessageNotFound
		}
	}

	msg, err = q.store.UpdatePriority(q.config.Name, id, newPriority)
	if err != nil {
		return err
	}

	queueItem.UpdatePriority(newPriority)

	// Update in-memory heap
	q.queue.UpdatePriority(group, id, newPriority)
	return nil
}

func (q *Queue) Ack(id uint64) error {
	queueItem := q.ackQueue.Get("default", id)

	if queueItem == nil {
		return errors.ErrMessageNotFound
	}

	if q.config.Type == "fair" {
		msg, err := q.Get(queueItem.ID)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to get message by ID: %d", queueItem.ID)
			return err
		}

		q.queue.UpdateWeights(msg.Group, msg.ID)
	}

	err := q.store.Ack(q.config.Name, queueItem.ID)
	if err != nil {
		return err
	}

	q.ackQueue.Delete("default", queueItem.ID)

	q.stats.IncrementAck()
	if q.cfg.Prometheus.Enabled {
		q.PrometheusMetrics.AckTotal.With(prometheus.Labels{"queue_name": q.config.Name}).Inc()

		q.updatePrometheusQueueSizes()
	}
	return nil
}

func (q *Queue) Nack(id uint64, priority int64, metadata map[string]string) error {
	item := q.ackQueue.Get("default", id)

	if item == nil {
		return errors.ErrMessageNotFound
	}

	message, err := q.Get(item.ID)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to get message by ID: %d", item.ID)
		return err
	}

	if priority != 0 {
		message.Priority = priority
	}

	queueItem := &memory.Item{
		ID:       message.ID,
		Priority: message.Priority,
		Group:    message.Group,
	}

	q.queue.Enqueue(message.Group, queueItem)
	q.ackQueue.Delete("default", item.ID)

	if q.config.Type == "fair" {
		q.queue.UpdateWeights(message.Group, message.ID)
	}

	if metadata != nil {
		err = q.store.UpdateMessage(q.config.Name, item.ID, priority, "", metadata)

		if err != nil {
			log.Error().Err(err).Msgf("Failed to update message: %d", item.ID)
			return err
		}
	}

	q.stats.IncrementNack()
	if q.cfg.Prometheus.Enabled {
		q.PrometheusMetrics.NackTotal.With(prometheus.Labels{"queue_name": q.config.Name}).Inc()

		q.updatePrometheusQueueSizes()
	}

	return nil
}

func (q *Queue) Len() int {
	return int(q.queue.Len())
}

func (q *Queue) PersistSnapshot(sink raft.SnapshotSink) error {
	return q.store.PersistSnapshot(q.config.Type, q.config.Name, sink)
}
