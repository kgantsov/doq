package storage

import (
	"github.com/hashicorp/raft"
	"github.com/kgantsov/doq/pkg/entity"
)

type Store interface {
	LoadQueue(queueName string) (*entity.QueueConfig, error)
	CreateQueue(queueType, queueName string, settings entity.QueueSettings) error
	DeleteQueue(queueName string) error
	Enqueue(
		queueName string,
		id uint64,
		group string,
		priority int64,
		content string,
		metadata map[string]string,
	) (*entity.Message, error)
	Dequeue(queueName string, id uint64, ack bool) (*entity.Message, error)
	Get(queueName string, id uint64) (*entity.Message, error)
	Delete(queueName string, id uint64) (*entity.Message, error)
	Ack(queueName string, id uint64) error
	UpdatePriority(queueName string, id uint64, priority int64) (*entity.Message, error)
	UpdateMessage(
		queueName string,
		id uint64,
		priority int64,
		content string,
		metadata map[string]string,
	) error
	PersistSnapshot(queueType, queueName string, sink raft.SnapshotSink) error
}
