package queue

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/kgantsov/doq/pkg/config"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
)

var ErrEmptyQueue = fmt.Errorf("queue is empty")
var ErrMessageNotFound = fmt.Errorf("message not found")

type QueueInfo struct {
	Name    string
	Type    string
	Stats   *QueueStats
	Ready   int64
	Unacked int64
	Total   int64
}

type QueueConfig struct {
	Name string
	Type string
}

func (qc *QueueConfig) ToBytes() ([]byte, error) {
	return json.Marshal(qc)
}

func QueueConfigFromBytes(data []byte) (*QueueConfig, error) {
	var qc QueueConfig
	err := json.Unmarshal(data, &qc)
	return &qc, err
}

type Message struct {
	Group    string
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
	config            *QueueConfig
	cfg               *config.Config
	PrometheusMetrics *PrometheusMetrics

	stats *queueStats

	pq Queue
	db *badger.DB

	mu sync.Mutex

	ackQueueMonitoringChan chan struct{}

	ackQueue   Queue
	ackQueueMu sync.Mutex
}

func NewBadgerPriorityQueue(db *badger.DB, cfg *config.Config, metrics *PrometheusMetrics) *BadgerPriorityQueue {

	bpq := &BadgerPriorityQueue{
		db:                db,
		cfg:               cfg,
		ackQueue:          NewDelayedPriorityQueue(false),
		PrometheusMetrics: metrics,
		stats:             NewQueueStats(10),
	}

	return bpq
}
func (bpq *BadgerPriorityQueue) Init(queueType, queueName string) error {
	var queue Queue
	if queueType == "fair" {
		queue = NewFairPriorityQueue()
	} else {
		queue = NewDelayedPriorityQueue(true)
	}
	bpq.config = &QueueConfig{Name: queueName, Type: queueType}
	bpq.pq = queue

	bpq.StartAckQueueMonitoring()

	go bpq.stats.Start()

	return nil
}

func (bpq *BadgerPriorityQueue) GetStats() *QueueInfo {
	return &QueueInfo{
		Name:    bpq.config.Name,
		Type:    bpq.config.Type,
		Stats:   bpq.stats.GetRPS(),
		Ready:   int64(bpq.pq.Len()),
		Unacked: int64(bpq.ackQueue.Len()),
		Total:   int64(bpq.pq.Len() + bpq.ackQueue.Len()),
	}
}

func (bpq *BadgerPriorityQueue) monitorAckQueue() {
	ticker := time.NewTicker(
		time.Duration(bpq.cfg.Queue.AcknowledgementCheckInterval) * time.Second,
	)

	for {
		select {
		case <-ticker.C:
			for {
				bpq.ackQueueMu.Lock()

				item := bpq.ackQueue.Dequeue()
				if item == nil {
					bpq.ackQueueMu.Unlock()
					break
				}

				message, err := bpq.GetByID(item.ID)
				if err != nil {
					log.Error().Err(err).Msgf("Failed to get message by ID: %d", item.ID)
					bpq.ackQueueMu.Unlock()
					continue
				}

				queueItem := &Item{
					ID:       message.ID,
					Priority: message.Priority,
				}

				bpq.pq.Enqueue(message.Group, queueItem)
				log.Debug().Msgf("Re-enqueued message: %d", message.ID)

				bpq.ackQueueMu.Unlock()
			}
		case <-bpq.ackQueueMonitoringChan:
			return
		}
	}
}

func (bpq *BadgerPriorityQueue) StartAckQueueMonitoring() {
	bpq.ackQueueMonitoringChan = make(chan struct{})
	go bpq.monitorAckQueue()
}

func (bpq *BadgerPriorityQueue) StopAckQueueMonitoring() {
	close(bpq.ackQueueMonitoringChan)
}

func (bpq *BadgerPriorityQueue) getMessagesPrefix() []byte {
	return []byte(fmt.Sprintf("messages:%s:", bpq.config.Name))
}

func (bpq *BadgerPriorityQueue) GetQueueKey(queueName string) []byte {
	return []byte(fmt.Sprintf("queues:%s:", queueName))
}

func (bpq *BadgerPriorityQueue) GetMessagesKey(id uint64) []byte {
	return addPrefix(bpq.getMessagesPrefix(), uint64ToBytes(id))
}

func (bpq *BadgerPriorityQueue) Create(queueType, queueName string) error {
	bpq.mu.Lock()
	defer bpq.mu.Unlock()

	bpq.config = &QueueConfig{Name: queueName, Type: queueType}

	err := bpq.db.Update(func(txn *badger.Txn) error {
		data, err := bpq.config.ToBytes()
		if err != nil {
			return err
		}
		return txn.Set(bpq.GetQueueKey(queueName), data)
	})
	if err != nil {
		return err
	}

	return bpq.Init(queueType, queueName)
}

func (bpq *BadgerPriorityQueue) Delete() error {
	bpq.mu.Lock()
	defer bpq.mu.Unlock()

	if bpq.config == nil {
		return ErrQueueNotFound
	}

	err := bpq.db.Update(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = true
		it := txn.NewIterator(opts)
		defer it.Close()

		prefix := bpq.getMessagesPrefix()
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			return txn.Delete(item.Key())
		}

		err := txn.Delete(bpq.GetQueueKey(bpq.config.Name))
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	bpq.StopAckQueueMonitoring()

	bpq.stats.Stop()

	return nil
}

func (bpq *BadgerPriorityQueue) Load(queueName string, loadMessages bool) error {
	bpq.mu.Lock()
	defer bpq.mu.Unlock()

	var qc *QueueConfig

	err := bpq.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(bpq.GetQueueKey(queueName))
		if err != nil {
			return err
		}

		err = item.Value(func(val []byte) error {
			qc, err = QueueConfigFromBytes(val)
			return err
		})

		if err != nil {
			return err
		}

		bpq.Init(qc.Type, qc.Name)

		if loadMessages {
			it := txn.NewIterator(badger.DefaultIteratorOptions)
			defer it.Close()

			prefix := bpq.getMessagesPrefix()
			for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
				item := it.Item()
				err := item.Value(func(val []byte) error {
					msg, err := MessageFromBytes(val)
					if err != nil {
						return err
					}
					bpq.pq.Enqueue(msg.Group, &Item{ID: msg.ID, Priority: msg.Priority})
					return nil
				})
				if err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func (bpq *BadgerPriorityQueue) Enqueue(id uint64, group string, priority int64, content string) (*Message, error) {
	bpq.mu.Lock()
	defer bpq.mu.Unlock()

	if bpq.config.Type != "fair" {
		group = "default"
	}

	msg := &Message{
		ID:       id,
		Group:    group,
		Priority: priority,
		Content:  content,
	}

	queueItem := &Item{
		ID:       msg.ID,
		Priority: msg.Priority,
	}

	err := bpq.db.Update(func(txn *badger.Txn) error {
		data, err := msg.ToBytes()
		if err != nil {
			return err
		}
		return txn.Set(bpq.GetMessagesKey(msg.ID), data)
	})
	if err != nil {
		return msg, err
	}

	bpq.pq.Enqueue(msg.Group, queueItem)

	bpq.stats.IncrementEnqueue()

	if bpq.cfg.Prometheus.Enabled {
		bpq.PrometheusMetrics.enqueueTotal.With(prometheus.Labels{"queue_name": bpq.config.Name}).Inc()
		bpq.PrometheusMetrics.messages.With(prometheus.Labels{"queue_name": bpq.config.Name}).Set(float64(bpq.pq.Len()))
	}

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
		item, err := txn.Get(bpq.GetMessagesKey(queueItem.ID))
		if err != nil {
			return err
		}

		item.Value(func(val []byte) error {
			msg, err = MessageFromBytes(val)
			return err
		})

		if ack {
			// in case of autoAck, we need to remove the message from the queue
			return txn.Delete(bpq.GetMessagesKey(queueItem.ID))
		} else {
			// in case of manual ack, we need to keep the message in the queue
			// so we can ack it later and add it to the ackQueue
			bpq.ackQueueMu.Lock()
			bpq.ackQueue.Enqueue(
				"default",
				&Item{
					ID: queueItem.ID,
					Priority: time.Now().UTC().Add(
						time.Duration(bpq.cfg.Queue.AcknowledgementTimeout) * time.Second,
					).Unix(),
				},
			)
			bpq.ackQueueMu.Unlock()
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	bpq.stats.IncrementDequeue()
	if bpq.cfg.Prometheus.Enabled {
		bpq.PrometheusMetrics.dequeueTotal.With(prometheus.Labels{"queue_name": bpq.config.Name}).Inc()
		bpq.PrometheusMetrics.messages.With(prometheus.Labels{"queue_name": bpq.config.Name}).Set(float64(bpq.pq.Len()))
	}
	return msg, nil
}

func (bpq *BadgerPriorityQueue) GetByID(id uint64) (*Message, error) {
	bpq.mu.Lock()
	defer bpq.mu.Unlock()

	var msg *Message

	err := bpq.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(bpq.GetMessagesKey(id))
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

	group := "default"

	// Update BadgerDB
	err := bpq.db.Update(func(txn *badger.Txn) error {
		var msg *Message
		item, err := txn.Get(bpq.GetMessagesKey(id))
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

		group = msg.Group
		queueItem := bpq.pq.GetByID(group, id)

		msg.UpdatePriority(newPriority)

		data, err := msg.ToBytes()
		if err != nil {
			return err
		}
		err = txn.Set(bpq.GetMessagesKey(queueItem.ID), data)

		if err != nil {
			return err
		}

		queueItem.UpdatePriority(newPriority)

		return nil
	})
	if err != nil {
		return err
	}

	// Update in-memory heap
	bpq.pq.UpdatePriority(group, id, newPriority)
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
		err := txn.Delete(bpq.GetMessagesKey(queueItem.ID))
		if err != nil {
			return err
		}
		bpq.ackQueue.DeleteByID("default", queueItem.ID)

		return nil
	})
	if err != nil {
		return err
	}

	bpq.stats.IncrementAck()
	if bpq.cfg.Prometheus.Enabled {
		bpq.PrometheusMetrics.ackTotal.With(prometheus.Labels{"queue_name": bpq.config.Name}).Inc()
	}
	return nil
}

func (bpq *BadgerPriorityQueue) Nack(id uint64) error {
	bpq.ackQueueMu.Lock()
	defer bpq.ackQueueMu.Unlock()

	item := bpq.ackQueue.GetByID("default", id)

	if item == nil {
		return ErrMessageNotFound
	}

	message, err := bpq.GetByID(item.ID)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to get message by ID: %d", item.ID)
		return err
	}

	queueItem := &Item{
		ID:       message.ID,
		Priority: message.Priority,
	}

	bpq.pq.Enqueue(message.Group, queueItem)
	bpq.ackQueue.DeleteByID("default", item.ID)

	bpq.stats.IncrementNack()
	if bpq.cfg.Prometheus.Enabled {
		bpq.PrometheusMetrics.nackTotal.With(prometheus.Labels{"queue_name": bpq.config.Name}).Inc()
	}

	return nil
}

func (bpq *BadgerPriorityQueue) Len() int {
	bpq.mu.Lock()
	defer bpq.mu.Unlock()

	return int(bpq.pq.Len())
}
