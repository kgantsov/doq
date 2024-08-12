package raft

import (
	"encoding/json"
	"time"

	"github.com/kgantsov/doq/pkg/queue"
)

func (n *Node) Enqueue(queueName string, priority int, content string) (*queue.Message, error) {
	cmd := Command{
		Op:        "enqueue",
		QueueName: queueName,
		Priority:  priority,
		Content:   content,
	}
	data, err := json.Marshal(cmd)
	if err != nil {
		return nil, err
	}

	f := n.Raft.Apply(data, 5*time.Second)
	if f.Error() != nil {
		return nil, f.Error()
	}

	r := f.Response().(*FSMResponse)
	if r.error != nil {
		return nil, r.error
	}

	return &queue.Message{
		ID:       r.ID,
		Priority: r.Priority,
		Content:  r.Content,
	}, nil
}

func (n *Node) Dequeue(QueueName string) (*queue.Message, error) {
	cmd := Command{
		Op:        "dequeue",
		QueueName: QueueName,
	}
	data, err := json.Marshal(cmd)
	if err != nil {
		return nil, err
	}

	f := n.Raft.Apply(data, 5*time.Second)
	if f.Error() != nil {
		return nil, f.Error()
	}

	r := f.Response().(*FSMResponse)
	if r.error != nil {
		return nil, r.error
	}

	return &queue.Message{
		ID:       r.ID,
		Priority: r.Priority,
		Content:  r.Content,
	}, nil
}

// func (n *Node) GetByID(id uint64) (*queue.Message, error) {

// }

// func (n *Node) UpdatePriority(id uint64, newPriority int) error {

// }
