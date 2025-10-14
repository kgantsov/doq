package mocks

import (
	"io"

	"github.com/kgantsov/doq/pkg/entity"
	"github.com/kgantsov/doq/pkg/errors"
	"github.com/kgantsov/doq/pkg/metrics"
	"github.com/kgantsov/doq/pkg/queue"
	"github.com/kgantsov/doq/pkg/queue/memory"
	"github.com/prometheus/client_golang/prometheus"
)

type mockNode struct {
	nextID   uint64
	isLeader bool
	leader   string
	messages map[uint64]*entity.Message
	acks     map[uint64]*entity.Message
	queues   map[string]*memory.DelayedQueue
}

func NewMockNode(leader string, isLeader bool) *mockNode {
	return &mockNode{
		messages: make(map[uint64]*entity.Message),
		acks:     make(map[uint64]*entity.Message),
		queues:   make(map[string]*memory.DelayedQueue),
		leader:   leader,
		isLeader: isLeader,
	}
}

func (n *mockNode) Join(nodeID, addr string) error {
	return nil
}

func (n *mockNode) Leave(nodeID string) error {
	return nil
}

func (n *mockNode) GetServers() ([]*entity.Server, error) {
	return nil, nil
}

func (n *mockNode) Backup(w io.Writer, since uint64) (uint64, error) {

	return 0, nil
}
func (n *mockNode) Restore(r io.Reader, maxPendingWrites int) error {

	return nil
}

func (n *mockNode) IsLeader() bool {
	return n.isLeader
}

func (n *mockNode) GenerateID() uint64 {
	n.nextID++
	return n.nextID
}

func (n *mockNode) GetQueues() []*queue.QueueInfo {
	queues := make([]*queue.QueueInfo, 0, len(n.queues))
	for name, q := range n.queues {
		queues = append(queues, &queue.QueueInfo{
			Name:    name,
			Type:    "delayed",
			Ready:   int64(q.Len()),
			Unacked: 0,
			Total:   int64(q.Len()),
			Stats: &metrics.Stats{
				EnqueueRPS: 3.1,
				DequeueRPS: 2.5,
				AckRPS:     1.2,
				NackRPS:    2.3,
			},
		})
	}

	return queues
}

func (n *mockNode) GetQueueInfo(queueName string) (*queue.QueueInfo, error) {
	q, ok := n.queues[queueName]
	if !ok {
		return nil, errors.ErrQueueNotFound
	}

	return &queue.QueueInfo{
		Name:    queueName,
		Type:    "delayed",
		Ready:   int64(q.Len()),
		Unacked: 0,
		Total:   int64(q.Len()),
		Stats: &metrics.Stats{
			EnqueueRPS: 3.1,
			DequeueRPS: 2.5,
			AckRPS:     1.2,
			NackRPS:    2.3,
		},
	}, nil
}

func (n *mockNode) PrometheusRegistry() prometheus.Registerer {
	return nil
}

func (n *mockNode) CreateQueue(queueType, queueName string, settings entity.QueueSettings) error {
	n.queues[queueName] = memory.NewDelayedQueue(true)
	return nil
}

func (n *mockNode) UpdateQueue(queueName string, settings entity.QueueSettings) error {
	_, ok := n.queues[queueName]
	if !ok {
		return errors.ErrQueueNotFound
	}

	return nil
}

func (n *mockNode) DeleteQueue(queueName string) error {
	_, ok := n.queues[queueName]
	if !ok {
		return errors.ErrQueueNotFound
	}

	delete(n.queues, queueName)
	return nil
}

func (n *mockNode) Enqueue(
	queueName string, id uint64, group string, priority int64, content string, metadata map[string]string,
) (*entity.Message, error) {
	q, ok := n.queues[queueName]
	if !ok {
		return &entity.Message{}, errors.ErrQueueNotFound
	}

	n.nextID++
	if id == 0 {
		id = n.nextID
	}
	message := &entity.Message{
		ID: id, Group: group, Priority: priority, Content: content, Metadata: metadata,
	}
	n.messages[message.ID] = message
	q.Enqueue(group, &memory.Item{ID: message.ID, Priority: message.Priority})
	return message, nil
}

func (n *mockNode) Dequeue(QueueName string, ack bool) (*entity.Message, error) {
	q, ok := n.queues[QueueName]
	if !ok {
		return &entity.Message{}, errors.ErrQueueNotFound
	}

	if q.Len() == 0 {
		return nil, errors.ErrEmptyQueue
	}

	item := q.Dequeue(false)

	message := n.messages[item.ID]

	if ack {
		delete(n.messages, item.ID)
	} else {
		n.acks[item.ID] = message
	}

	return message, nil
}

func (n *mockNode) Ack(QueueName string, id uint64) error {
	_, ok := n.queues[QueueName]
	if !ok {
		return errors.ErrQueueNotFound
	}

	if _, ok := n.acks[id]; !ok {
		return errors.ErrMessageNotFound
	}
	delete(n.acks, id)
	delete(n.messages, id)
	return nil
}

func (n *mockNode) Nack(QueueName string, id uint64, priority int64, metadata map[string]string) error {
	q, ok := n.queues[QueueName]
	if !ok {
		return errors.ErrQueueNotFound
	}

	message, ok := n.acks[id]
	if !ok {
		return errors.ErrMessageNotFound
	}

	if priority != 0 {
		message.Priority = priority
	}

	message.Metadata = metadata
	n.messages[message.ID] = message

	q.Enqueue(message.Group, &memory.Item{ID: message.ID, Priority: message.Priority})

	delete(n.acks, id)
	return nil
}

func (n *mockNode) Get(QueueName string, id uint64) (*entity.Message, error) {
	for _, m := range n.messages {
		if m.ID == id {
			return m, nil
		}
	}
	return nil, errors.ErrMessageNotFound
}

func (n *mockNode) Delete(QueueName string, id uint64) error {
	q, ok := n.queues[QueueName]
	if !ok {
		return errors.ErrQueueNotFound
	}

	msg, ok := n.messages[id]
	if !ok {
		return errors.ErrMessageNotFound
	}

	q.Delete(msg.Group, id)

	delete(n.acks, id)
	delete(n.messages, id)

	return nil
}

func (n *mockNode) UpdatePriority(queueName string, id uint64, priority int64) error {
	q, ok := n.queues[queueName]
	if !ok {
		return errors.ErrQueueNotFound
	}

	message, ok := n.messages[id]
	if !ok {
		return errors.ErrMessageNotFound
	}

	message.Priority = priority
	q.UpdatePriority(message.Group, id, priority)
	return nil
}
