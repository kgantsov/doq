package raft

import (
	"encoding/json"
	"time"

	"github.com/kgantsov/doq/pkg/entity"
	"github.com/kgantsov/doq/pkg/queue"
)

func (n *Node) Enqueue(
	queueName string, id uint64, group string, priority int64, content string, metadata map[string]string,
) (*entity.Message, error) {
	if id == 0 {
		id = uint64(n.idGenerator.Generate().Int64())
	}
	cmd := Command{
		Op: "enqueue",
		Payload: EnqueuePayload{
			ID:        id,
			QueueName: queueName,
			Group:     group,
			Priority:  priority,
			Content:   content,
			Metadata:  metadata,
		},
	}
	data, err := json.Marshal(cmd)
	if err != nil {
		return nil, err
	}

	f := n.Raft.Apply(data, time.Duration(n.cfg.Raft.ApplyTimeout)*time.Second)
	if f.Error() != nil {
		return nil, f.Error()
	}

	r := f.Response().(*FSMResponse)
	if r.error != nil {
		return nil, r.error
	}

	return &entity.Message{
		ID:       r.ID,
		Group:    group,
		Priority: r.Priority,
		Content:  r.Content,
		Metadata: r.Metadata,
	}, nil
}

func (n *Node) Dequeue(QueueName string, ack bool) (*entity.Message, error) {
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

	f := n.Raft.Apply(data, time.Duration(n.cfg.Raft.ApplyTimeout)*time.Second)
	if f.Error() != nil {
		return nil, f.Error()
	}

	r := f.Response().(*FSMResponse)
	if r.error != nil {
		return nil, r.error
	}

	return &entity.Message{
		ID:       r.ID,
		Group:    r.Group,
		Priority: r.Priority,
		Content:  r.Content,
		Metadata: r.Metadata,
	}, nil
}

func (n *Node) Get(QueueName string, id uint64) (*entity.Message, error) {
	cmd := Command{
		Op: "get",
		Payload: GetPayload{
			QueueName: QueueName,
			// Group:     group,
			ID: id,
		},
	}
	data, err := json.Marshal(cmd)
	if err != nil {
		return nil, err
	}

	f := n.Raft.Apply(data, time.Duration(n.cfg.Raft.ApplyTimeout)*time.Second)
	if f.Error() != nil {
		return nil, f.Error()
	}

	r := f.Response().(*FSMResponse)
	if r.error != nil {
		return nil, r.error
	}

	return &entity.Message{
		ID:       r.ID,
		Group:    r.Group,
		Priority: r.Priority,
		Content:  r.Content,
		Metadata: r.Metadata,
	}, nil
}

func (n *Node) Delete(QueueName string, id uint64) error {
	cmd := Command{
		Op: "delete",
		Payload: DeletePayload{
			QueueName: QueueName,
			ID:        id,
		},
	}
	data, err := json.Marshal(cmd)
	if err != nil {
		return err
	}

	f := n.Raft.Apply(data, time.Duration(n.cfg.Raft.ApplyTimeout)*time.Second)
	if f.Error() != nil {
		return f.Error()
	}

	r := f.Response().(*FSMResponse)
	if r.error != nil {
		return r.error
	}

	return nil
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

	f := n.Raft.Apply(data, time.Duration(n.cfg.Raft.ApplyTimeout)*time.Second)
	if f.Error() != nil {
		return f.Error()
	}

	r := f.Response().(*FSMResponse)
	if r.error != nil {
		return r.error
	}

	return nil
}

func (n *Node) Nack(QueueName string, id uint64, priority int64, metadata map[string]string) error {
	cmd := Command{
		Op: "nack",
		Payload: NackPayload{
			QueueName: QueueName,
			ID:        id,
			Priority:  priority,
			Metadata:  metadata,
		},
	}
	data, err := json.Marshal(cmd)
	if err != nil {
		return err
	}

	f := n.Raft.Apply(data, time.Duration(n.cfg.Raft.ApplyTimeout)*time.Second)
	if f.Error() != nil {
		return f.Error()
	}

	r := f.Response().(*FSMResponse)
	if r.error != nil {
		return r.error
	}

	return nil
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

	f := n.Raft.Apply(data, time.Duration(n.cfg.Raft.ApplyTimeout)*time.Second)
	if f.Error() != nil {
		return f.Error()
	}

	r := f.Response().(*FSMResponse)
	if r.error != nil {
		return r.error
	}

	return nil
}

func (n *Node) GetQueues() []*queue.QueueInfo {
	return n.QueueManager.GetQueues()
}

func (n *Node) GetQueueInfo(queueName string) (*queue.QueueInfo, error) {
	q, err := n.QueueManager.GetQueue(queueName)
	if err != nil {
		return nil, err
	}

	return q.GetStats(), nil
}

func (n *Node) CreateQueue(queueType, queueName string, settings entity.QueueSettings) error {
	cmd := Command{
		Op: "createQueue",
		Payload: CreateQueuePayload{
			QueueType: queueType,
			QueueName: queueName,
			Settings: QueueSettings{
				Strategy:   settings.Strategy,
				MaxUnacked: settings.MaxUnacked,
			},
		},
	}
	data, err := json.Marshal(cmd)
	if err != nil {
		return err
	}

	f := n.Raft.Apply(data, time.Duration(n.cfg.Raft.ApplyTimeout)*time.Second)
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

	f := n.Raft.Apply(data, time.Duration(n.cfg.Raft.ApplyTimeout)*time.Second)
	if f.Error() != nil {
		return f.Error()
	}

	r := f.Response().(*FSMResponse)
	if r.error != nil {
		return r.error
	}

	return nil
}
