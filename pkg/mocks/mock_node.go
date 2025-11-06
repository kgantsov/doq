package mocks

import (
	"io"
	"sync"

	"github.com/kgantsov/doq/pkg/entity"
	"github.com/kgantsov/doq/pkg/queue"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/mock"
)

type mockNode struct {
	mock.Mock
	NextIDFunc func() uint64
	mu         sync.Mutex
}

func NewMockNode() *mockNode {
	return &mockNode{
		mu: sync.Mutex{},
	}
}

func (n *mockNode) Join(nodeID, addr string) error {
	args := n.Called(nodeID, addr)
	return args.Error(0)
}

func (n *mockNode) Leave(nodeID string) error {
	args := n.Called(nodeID)
	return args.Error(0)
}

func (n *mockNode) GetServers() ([]*entity.Server, error) {
	args := n.Called()
	return args.Get(0).([]*entity.Server), args.Error(1)
}

func (n *mockNode) Backup(w io.Writer, since uint64) (uint64, error) {
	args := n.Called(w, since)
	return args.Get(0).(uint64), args.Error(1)
}

func (n *mockNode) Restore(r io.Reader, maxPendingWrites int) error {
	args := n.Called(r, maxPendingWrites)
	return args.Error(0)
}

func (n *mockNode) IsLeader() bool {
	args := n.Called()
	return args.Bool(0)
}

func (n *mockNode) GenerateID() uint64 {
	if n.NextIDFunc != nil {
		return n.NextIDFunc()
	}
	args := n.Called()
	return args.Get(0).(uint64)
}

func (n *mockNode) GetQueues() []*queue.QueueInfo {
	args := n.Called()
	return args.Get(0).([]*queue.QueueInfo)
}

func (n *mockNode) GetQueueInfo(queueName string) (*queue.QueueInfo, error) {
	args := n.Called(queueName)
	return args.Get(0).(*queue.QueueInfo), args.Error(1)
}

func (n *mockNode) PrometheusRegistry() prometheus.Registerer {
	args := n.Called()
	return args.Get(0).(prometheus.Registerer)
}

func (n *mockNode) CreateQueue(queueType, queueName string, settings entity.QueueSettings) error {
	args := n.Called(queueType, queueName, settings)
	return args.Error(0)
}

func (n *mockNode) UpdateQueue(queueName string, settings entity.QueueSettings) error {
	args := n.Called(queueName, settings)
	return args.Error(0)
}

func (n *mockNode) DeleteQueue(queueName string) error {
	args := n.Called(queueName)
	return args.Error(0)
}

func (n *mockNode) Enqueue(
	queueName string, id uint64, group string, priority int64, content string, metadata map[string]string,
) (*entity.Message, error) {
	args := n.Called(queueName, id, group, priority, content, metadata)
	return args.Get(0).(*entity.Message), args.Error(1)
}

func (n *mockNode) Dequeue(QueueName string, ack bool) (*entity.Message, error) {
	args := n.Called(QueueName, ack)
	return args.Get(0).(*entity.Message), args.Error(1)
}

func (n *mockNode) Ack(QueueName string, id uint64) error {
	args := n.Called(QueueName, id)
	return args.Error(0)
}

func (n *mockNode) Nack(QueueName string, id uint64, priority int64, metadata map[string]string) error {
	args := n.Called(QueueName, id, priority, metadata)
	return args.Error(0)
}

func (n *mockNode) Touch(QueueName string, id uint64) error {
	args := n.Called(QueueName, id)
	return args.Error(0)
}

func (n *mockNode) Get(QueueName string, id uint64) (*entity.Message, error) {
	args := n.Called(QueueName, id)
	return args.Get(0).(*entity.Message), args.Error(1)
}

func (n *mockNode) Delete(QueueName string, id uint64) error {
	args := n.Called(QueueName, id)
	return args.Error(0)
}

func (n *mockNode) UpdatePriority(queueName string, id uint64, priority int64) error {
	args := n.Called(queueName, id, priority)
	return args.Error(0)
}
