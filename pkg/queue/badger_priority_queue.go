package queue

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/hashicorp/raft"
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
	Metadata map[string]string

	QueueName string `json:"QueueName,omitempty"`
	QueueType string `json:"QueueType,omitempty"`
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

	ackQueue Queue
}

func NewBadgerPriorityQueue(db *badger.DB, cfg *config.Config, metrics *PrometheusMetrics) *BadgerPriorityQueue {

	bpq := &BadgerPriorityQueue{
		db:                db,
		cfg:               cfg,
		ackQueue:          NewDelayedPriorityQueue(false),
		PrometheusMetrics: metrics,
		stats:             NewQueueStats(cfg.Queue.QueueStats.WindowSide),
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

func (bpq *BadgerPriorityQueue) updatePrometheusQueueSizes() {
	if bpq.cfg.Prometheus.Enabled {
		readyMessages := float64(bpq.pq.Len())
		unackedMessages := float64(bpq.ackQueue.Len())

		bpq.PrometheusMetrics.messages.With(
			prometheus.Labels{"queue_name": bpq.config.Name},
		).Set(readyMessages + unackedMessages)
		bpq.PrometheusMetrics.unackedMessages.With(
			prometheus.Labels{"queue_name": bpq.config.Name},
		).Set(unackedMessages)
		bpq.PrometheusMetrics.readyMessages.With(
			prometheus.Labels{"queue_name": bpq.config.Name},
		).Set(readyMessages)
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
				item := bpq.ackQueue.Dequeue()
				if item == nil {
					break
				}

				message, err := bpq.Get(item.ID)
				if err != nil {
					log.Error().Err(err).Msgf("Failed to get message by ID: %d", item.ID)
					continue
				}

				queueItem := &Item{
					ID:       message.ID,
					Priority: message.Priority,
					Group:    message.Group,
				}

				bpq.pq.Enqueue(message.Group, queueItem)
				log.Debug().Msgf("Re-enqueued message: %d", message.ID)
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

func (bpq *BadgerPriorityQueue) DeleteQueue() error {
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

	bpq.updatePrometheusQueueSizes()

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
					bpq.pq.Enqueue(
						msg.Group, &Item{ID: msg.ID, Priority: msg.Priority, Group: msg.Group},
					)
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

	bpq.updatePrometheusQueueSizes()

	return nil
}

func (bpq *BadgerPriorityQueue) Enqueue(
	id uint64,
	group string,
	priority int64,
	content string,
	metadata map[string]string,
) (*Message, error) {
	if bpq.config.Type != "fair" {
		group = "default"
	}

	msg := &Message{
		ID:       id,
		Group:    group,
		Priority: priority,
		Content:  content,
		Metadata: metadata,
	}

	queueItem := &Item{
		ID:       msg.ID,
		Priority: msg.Priority,
		Group:    msg.Group,
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

		bpq.updatePrometheusQueueSizes()
	}

	return msg, nil
}

func (bpq *BadgerPriorityQueue) Dequeue(ack bool) (*Message, error) {
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
			bpq.ackQueue.Enqueue(
				"default",
				&Item{
					ID: queueItem.ID,
					Priority: time.Now().UTC().Add(
						time.Duration(bpq.cfg.Queue.AcknowledgementTimeout) * time.Second,
					).Unix(),
					Group: msg.Group,
				},
			)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	bpq.stats.IncrementDequeue()
	if bpq.cfg.Prometheus.Enabled {
		bpq.PrometheusMetrics.dequeueTotal.With(prometheus.Labels{"queue_name": bpq.config.Name}).Inc()

		bpq.updatePrometheusQueueSizes()
	}

	return msg, nil
}

func (bpq *BadgerPriorityQueue) Get(id uint64) (*Message, error) {
	var msg *Message

	err := bpq.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(bpq.GetMessagesKey(id))
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return ErrMessageNotFound
			}
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

func (bpq *BadgerPriorityQueue) Delete(id uint64) error {
	group := "default"

	err := bpq.db.Update(func(txn *badger.Txn) error {
		var msg *Message

		item, err := txn.Get(bpq.GetMessagesKey(id))
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return ErrMessageNotFound
			}
			return err
		}

		err = item.Value(func(val []byte) error {
			msg, err = MessageFromBytes(val)
			if err != nil {
				return err
			}

			return nil
		})

		if err != nil {
			return err
		}

		group = msg.Group
		bpq.pq.Delete(group, id)
		bpq.ackQueue.Delete(group, id)

		err = txn.Delete(bpq.GetMessagesKey(msg.ID))
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return ErrMessageNotFound
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func (bpq *BadgerPriorityQueue) UpdatePriority(id uint64, newPriority int64) error {
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
		queueItem := bpq.pq.Get(group, id)

		if queueItem == nil {
			queueItem = bpq.ackQueue.Get(group, id)
			if queueItem == nil {
				return ErrMessageNotFound
			}
		}

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
	queueItem := bpq.ackQueue.Get("default", id)

	if queueItem == nil {
		return ErrMessageNotFound
	}

	err := bpq.db.Update(func(txn *badger.Txn) error {
		err := txn.Delete(bpq.GetMessagesKey(queueItem.ID))
		if err != nil {
			return err
		}
		bpq.ackQueue.Delete("default", queueItem.ID)

		return nil
	})
	if err != nil {
		return err
	}

	bpq.stats.IncrementAck()
	if bpq.cfg.Prometheus.Enabled {
		bpq.PrometheusMetrics.ackTotal.With(prometheus.Labels{"queue_name": bpq.config.Name}).Inc()

		bpq.updatePrometheusQueueSizes()
	}
	return nil
}

func (bpq *BadgerPriorityQueue) Nack(id uint64, priority int64, metadata map[string]string) error {
	item := bpq.ackQueue.Get("default", id)

	if item == nil {
		return ErrMessageNotFound
	}

	message, err := bpq.Get(item.ID)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to get message by ID: %d", item.ID)
		return err
	}

	if priority != 0 {
		message.Priority = priority
	}

	queueItem := &Item{
		ID:       message.ID,
		Priority: message.Priority,
		Group:    message.Group,
	}

	bpq.pq.Enqueue(message.Group, queueItem)
	bpq.ackQueue.Delete("default", item.ID)

	if metadata != nil {
		err = bpq.updateMessage(item.ID, priority, "", metadata)

		if err != nil {
			log.Error().Err(err).Msgf("Failed to update message: %d", item.ID)
			return err
		}
	}

	bpq.stats.IncrementNack()
	if bpq.cfg.Prometheus.Enabled {
		bpq.PrometheusMetrics.nackTotal.With(prometheus.Labels{"queue_name": bpq.config.Name}).Inc()

		bpq.updatePrometheusQueueSizes()
	}

	return nil
}

func (bpq *BadgerPriorityQueue) updateMessage(
	id uint64,
	priority int64,
	content string,
	metadata map[string]string,
) error {
	return bpq.db.Update(func(txn *badger.Txn) error {
		item, err := txn.Get(bpq.GetMessagesKey(id))
		if err != nil {
			return err
		}

		var msg *Message
		err = item.Value(func(val []byte) error {
			msg, err = MessageFromBytes(val)
			return err
		})
		if err != nil {
			return err
		}

		if priority != 0 {
			msg.Priority = priority
		}

		if content != "" {
			msg.Content = content
		}

		if metadata != nil {
			msg.Metadata = metadata
		}

		data, err := msg.ToBytes()
		if err != nil {
			return err
		}

		return txn.Set(bpq.GetMessagesKey(id), data)
	})
}

func (bpq *BadgerPriorityQueue) Len() int {
	return int(bpq.pq.Len())
}

func (bpq *BadgerPriorityQueue) PersistSnapshot(sink raft.SnapshotSink) error {
	err := bpq.db.View(func(txn *badger.Txn) error {
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

				msg.QueueType = bpq.config.Type
				msg.QueueName = bpq.config.Name

				data, err := json.Marshal(msg)
				if err != nil {
					log.Error().Err(err).Msgf("Failed to marshal queue %s item", bpq.config.Name)
					return err
				}

				_, err = sink.Write(data)
				if err != nil {
					log.Error().Err(err).Msgf(
						"Failed to write a queue %s item to snapshot sink", bpq.config.Name,
					)
					return err
				}

				_, err = sink.Write([]byte("\n"))
				if err != nil {
					log.Error().Err(err).Msgf(
						"Failed to write a newline to snapshot sink for queue %s", bpq.config.Name,
					)
				}

				return nil
			})
			if err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}
