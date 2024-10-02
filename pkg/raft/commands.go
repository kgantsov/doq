package raft

import (
	"encoding/json"
	"time"

	"github.com/kgantsov/doq/pkg/queue"
)

func (n *Node) Enqueue(
	queueName string, group string, priority int64, content string,
) (*queue.Message, error) {
	cmd := Command{
		Op: "enqueue",
		Payload: EnqueuePayload{
			ID:        uint64(n.idGenerator.Generate().Int64()),
			QueueName: queueName,
			Group:     group,
			Priority:  priority,
			Content:   content,
		},
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
		Op: "dequeue",
		Payload: DequeuePayload{
			QueueName: QueueName,
			Ack:       ack,
		},
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
		Op: "ack",
		Payload: AckPayload{
			QueueName: QueueName,
			ID:        id,
		},
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

func (n *Node) GetByID(id uint64) (*queue.Message, error) {
	return nil, nil
}

func (n *Node) UpdatePriority(queueName string, id uint64, priority int64) error {
	cmd := Command{
		Op: "updatePriority",
		Payload: UpdatePriorityPayload{
			ID:        id,
			QueueName: queueName,
			Priority:  priority,
		},
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

func (n *Node) GetQueueInfo(queueName string) (*queue.QueueInfo, error) {
	q, err := n.QueueManager.GetQueue(queueName)
	if err != nil {
		return nil, err
	}

	return q.GetStats(), nil
}

func (n *Node) CreateQueue(queueType, queueName string) error {
	cmd := Command{
		Op: "createQueue",
		Payload: CreateQueuePayload{
			QueueType: queueType,
			QueueName: queueName,
		},
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
func (n *Node) DeleteQueue(queueName string) error {
	cmd := Command{
		Op: "deleteQueue",
		Payload: DeleteQueuePayload{
			QueueName: queueName,
		},
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
