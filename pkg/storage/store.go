package storage

import (
	"io"

	"github.com/kgantsov/doq/pkg/queue"
)

type Store interface {
	CreateQueue(queueType, queueName string) error
	DeleteQueue(queueName string) error
	Enqueue(
		queueName string,
		group string,
		priority int64,
		content string,
		metadata map[string]string,
	) (*queue.Message, error)
	Dequeue(queueName string, ack bool) (*queue.Message, error)
	Get(queueName string, id uint64) (*queue.Message, error)
	Delete(queueName string, id uint64) (*queue.Message, error)
	Ack(queueName string, id uint64) error
	Nack(queueName string, id uint64, priority int64, metadata map[string]string) error
	UpdatePriority(queueName string, id uint64, priority int64) (*queue.Message, error)
	Backup(w io.Writer, since uint64) (uint64, error)
	Restore(r io.Reader, maxPendingWrites int) error
}
