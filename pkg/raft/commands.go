package raft

import (
	"encoding/json"
	"time"

	"github.com/kgantsov/doq/pkg/queue"
)

func (n *Node) Enqueue(queueName string, priority int64, content string) (*queue.Message, error) {
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

func (n *Node) Dequeue(QueueName string, ack bool) (*queue.Message, error) {
	cmd := Command{
		Op:        "dequeue",
		QueueName: QueueName,
		Ack:       ack,
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

func (n *Node) Ack(QueueName string, id uint64) error {
	cmd := Command{
		Op:        "ack",
		QueueName: QueueName,
		ID:        id,
	}
	data, err := json.Marshal(cmd)
	if err != nil {
		return err
	}

	f := n.Raft.Apply(data, 5*time.Second)
	if f.Error() != nil {
		return f.Error()
	}

	r := f.Response().(*FSMResponse)
	if r.error != nil {
		return r.error
	}

	return nil
}

// func (n *Node) GetByID(id uint64) (*queue.Message, error) {

// }

func (n *Node) UpdatePriority(queueName string, id uint64, priority int64) error {
	cmd := Command{
		ID:        id,
		Op:        "updatePriority",
		QueueName: queueName,
		Priority:  priority,
	}
	data, err := json.Marshal(cmd)
	if err != nil {
		return err
	}

	f := n.Raft.Apply(data, 5*time.Second)
	if f.Error() != nil {
		return f.Error()
	}

	r := f.Response().(*FSMResponse)
	if r.error != nil {
		return r.error
	}

	return nil
}
