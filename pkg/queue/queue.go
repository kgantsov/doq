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
func (bpq *Queue) Init(queueType, queueName string) error {
	var queue memory.MemoryQueue
	if queueType == "fair" {
		queue = memory.NewFairMemoryQueue()
	} else {
		queue = memory.NewDelayedMemoryQueue(true)
	}
	bpq.config = &entity.QueueConfig{Name: queueName, Type: queueType}
	bpq.queue = queue

	bpq.StartAckQueueMonitoring()

	go bpq.stats.Start()

	return nil
}

func (bpq *Queue) GetStats() *QueueInfo {
	return &QueueInfo{
		Name:    bpq.config.Name,
		Type:    bpq.config.Type,
		Stats:   bpq.stats.GetRPS(),
		Ready:   int64(bpq.queue.Len()),
		Unacked: int64(bpq.ackQueue.Len()),
		Total:   int64(bpq.queue.Len() + bpq.ackQueue.Len()),
	}
}

func (bpq *Queue) updatePrometheusQueueSizes() {
	if bpq.cfg.Prometheus.Enabled {
		readyMessages := float64(bpq.queue.Len())
		unackedMessages := float64(bpq.ackQueue.Len())

		bpq.PrometheusMetrics.Messages.With(
			prometheus.Labels{"queue_name": bpq.config.Name},
		).Set(readyMessages + unackedMessages)
		bpq.PrometheusMetrics.UnackedMessages.With(
			prometheus.Labels{"queue_name": bpq.config.Name},
		).Set(unackedMessages)
		bpq.PrometheusMetrics.ReadyMessages.With(
			prometheus.Labels{"queue_name": bpq.config.Name},
		).Set(readyMessages)
	}
}

func (bpq *Queue) monitorAckQueue() {
	ticker := time.NewTicker(
		time.Duration(bpq.cfg.Queue.AcknowledgementCheckInterval) * time.Second,
	)

	for {
		select {
		case <-ticker.C:
			for {
				item := bpq.ackQueue.Dequeue()
				if item == nil {
					break
				}

				message, err := bpq.Get(item.ID)
				if err != nil {
					log.Error().Err(err).Msgf("Failed to get message by ID: %d", item.ID)
					continue
				}

				queueItem := &memory.Item{
					ID:       message.ID,
					Priority: message.Priority,
					Group:    message.Group,
				}

				bpq.queue.Enqueue(message.Group, queueItem)
				log.Debug().Msgf("Re-enqueued message: %d", message.ID)
			}
		case <-bpq.ackQueueMonitoringChan:
			return
		}
	}
}

func (bpq *Queue) StartAckQueueMonitoring() {
	bpq.ackQueueMonitoringChan = make(chan struct{})
	go bpq.monitorAckQueue()
}

func (bpq *Queue) StopAckQueueMonitoring() {
	close(bpq.ackQueueMonitoringChan)
}

func (bpq *Queue) Create(queueType, queueName string) error {
	bpq.mu.Lock()
	defer bpq.mu.Unlock()

	bpq.config = &entity.QueueConfig{Name: queueName, Type: queueType}

	err := bpq.store.CreateQueue(queueType, queueName)
	if err != nil {
		return err
	}

	return bpq.Init(queueType, queueName)
}

func (bpq *Queue) DeleteQueue() error {
	bpq.mu.Lock()
	defer bpq.mu.Unlock()

	if bpq.config == nil {
		return errors.ErrQueueNotFound
	}

	err := bpq.store.DeleteQueue(bpq.config.Name)
	if err != nil {
		return err
	}

	bpq.StopAckQueueMonitoring()

	bpq.stats.Stop()

	bpq.updatePrometheusQueueSizes()

	return nil
}

func (bpq *Queue) Load(queueName string) error {
	bpq.mu.Lock()
	defer bpq.mu.Unlock()

	qc, err := bpq.store.LoadQueue(queueName)
	if err != nil {
		return err
	}

	err = bpq.Init(qc.Type, qc.Name)
	if err != nil {
		return err
	}

	bpq.updatePrometheusQueueSizes()

	return nil
}

func (bpq *Queue) Enqueue(
	id uint64,
	group string,
	priority int64,
	content string,
	metadata map[string]string,
) (*entity.Message, error) {
	if bpq.config.Type != "fair" {
		group = "default"
	}

	queueItem := &memory.Item{
		ID:       id,
		Group:    group,
		Priority: priority,
	}

	msg, err := bpq.store.Enqueue(
		bpq.config.Name, queueItem.ID, queueItem.Group, queueItem.Priority, content, metadata,
	)
	if err != nil {
		return msg, err
	}

	bpq.queue.Enqueue(msg.Group, queueItem)

	bpq.stats.IncrementEnqueue()

	if bpq.cfg.Prometheus.Enabled {
		bpq.PrometheusMetrics.EnqueueTotal.With(prometheus.Labels{"queue_name": bpq.config.Name}).Inc()

		bpq.updatePrometheusQueueSizes()
	}

	return msg, nil
}

func (bpq *Queue) Dequeue(ack bool) (*entity.Message, error) {
	if bpq.queue.Len() == 0 {
		return nil, errors.ErrEmptyQueue
	}

	queueItem := bpq.queue.Dequeue()
	if queueItem == nil {
		return nil, errors.ErrEmptyQueue
	}

	msg, err := bpq.store.Dequeue(bpq.config.Name, queueItem.ID, ack)

	if err != nil {
		return nil, err
	}

	if !ack {
		bpq.ackQueue.Enqueue(
			"default",
			&memory.Item{
				ID: queueItem.ID,
				Priority: time.Now().UTC().Add(
					time.Duration(bpq.cfg.Queue.AcknowledgementTimeout) * time.Second,
				).Unix(),
				Group: msg.Group,
			},
		)
	}

	bpq.stats.IncrementDequeue()
	if bpq.cfg.Prometheus.Enabled {
		bpq.PrometheusMetrics.DequeueTotal.With(prometheus.Labels{"queue_name": bpq.config.Name}).Inc()

		bpq.updatePrometheusQueueSizes()
	}

	return msg, nil
}

func (bpq *Queue) Get(id uint64) (*entity.Message, error) {
	return bpq.store.Get(bpq.config.Name, id)
}

func (bpq *Queue) Delete(id uint64) error {
	msg, err := bpq.store.Delete(bpq.config.Name, id)
	if err != nil {
		return err
	}

	group := msg.Group
	bpq.queue.Delete(group, id)
	bpq.ackQueue.Delete(group, id)

	return nil
}

func (bpq *Queue) UpdatePriority(id uint64, newPriority int64) error {
	msg, err := bpq.store.Get(bpq.config.Name, id)
	if err != nil {
		return err
	}

	group := msg.Group
	queueItem := bpq.queue.Get(group, id)

	if queueItem == nil {
		queueItem = bpq.ackQueue.Get(group, id)
		if queueItem == nil {
			return errors.ErrMessageNotFound
		}
	}

	msg, err = bpq.store.UpdatePriority(bpq.config.Name, id, newPriority)
	if err != nil {
		return err
	}

	queueItem.UpdatePriority(newPriority)

	// Update in-memory heap
	bpq.queue.UpdatePriority(group, id, newPriority)
	return nil
}

func (bpq *Queue) Ack(id uint64) error {
	queueItem := bpq.ackQueue.Get("default", id)

	if queueItem == nil {
		return errors.ErrMessageNotFound
	}

	err := bpq.store.Ack(bpq.config.Name, queueItem.ID)
	if err != nil {
		return err
	}

	bpq.ackQueue.Delete("default", queueItem.ID)

	bpq.stats.IncrementAck()
	if bpq.cfg.Prometheus.Enabled {
		bpq.PrometheusMetrics.AckTotal.With(prometheus.Labels{"queue_name": bpq.config.Name}).Inc()

		bpq.updatePrometheusQueueSizes()
	}
	return nil
}

func (bpq *Queue) Nack(id uint64, priority int64, metadata map[string]string) error {
	item := bpq.ackQueue.Get("default", id)

	if item == nil {
		return errors.ErrMessageNotFound
	}

	message, err := bpq.Get(item.ID)
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

	bpq.queue.Enqueue(message.Group, queueItem)
	bpq.ackQueue.Delete("default", item.ID)

	if metadata != nil {
		err = bpq.store.UpdateMessage(bpq.config.Name, item.ID, priority, "", metadata)

		if err != nil {
			log.Error().Err(err).Msgf("Failed to update message: %d", item.ID)
			return err
		}
	}

	bpq.stats.IncrementNack()
	if bpq.cfg.Prometheus.Enabled {
		bpq.PrometheusMetrics.NackTotal.With(prometheus.Labels{"queue_name": bpq.config.Name}).Inc()

		bpq.updatePrometheusQueueSizes()
	}

	return nil
}

func (bpq *Queue) Len() int {
	return int(bpq.queue.Len())
}

func (bpq *Queue) PersistSnapshot(sink raft.SnapshotSink) error {
	return bpq.store.PersistSnapshot(bpq.config.Type, bpq.config.Name, sink)
}
