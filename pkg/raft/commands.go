package raft

import (
	"time"

	pb "github.com/kgantsov/doq/pkg/proto"
	"google.golang.org/protobuf/proto"

	"github.com/kgantsov/doq/pkg/entity"
	"github.com/kgantsov/doq/pkg/queue"
)

func (n *Node) Enqueue(
	queueName string, id uint64, group string, priority int64, content string, metadata map[string]string,
) (*entity.Message, error) {
	if id == 0 {
		id = uint64(n.idGenerator.Generate().Int64())
	}

	cmd := &pb.RaftCommand{
		Cmd: &pb.RaftCommand_Enqueue{
			Enqueue: &pb.EnqueueRequest{
				QueueName: queueName,
				Id:        id,
				Group:     group,
				Priority:  priority,
				Content:   content,
				Metadata:  metadata,
			},
		},
	}

	data, err := proto.Marshal(cmd)
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
	cmd := &pb.RaftCommand{
		Cmd: &pb.RaftCommand_Dequeue{
			Dequeue: &pb.DequeueRequest{
				QueueName: QueueName,
				Ack:       ack,
			},
		},
	}
	data, err := proto.Marshal(cmd)
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
	cmd := &pb.RaftCommand{
		Cmd: &pb.RaftCommand_Get{
			Get: &pb.GetRequest{
				QueueName: QueueName,
				Id:        id,
			},
		},
	}
	data, err := proto.Marshal(cmd)
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
	cmd := &pb.RaftCommand{
		Cmd: &pb.RaftCommand_Delete{
			Delete: &pb.DeleteRequest{
				QueueName: QueueName,
				Id:        id,
			},
		},
	}

	data, err := proto.Marshal(cmd)
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
	cmd := &pb.RaftCommand{
		Cmd: &pb.RaftCommand_Ack{
			Ack: &pb.AckRequest{
				QueueName: QueueName,
				Id:        id,
			},
		},
	}
	data, err := proto.Marshal(cmd)
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
	cmd := &pb.RaftCommand{
		Cmd: &pb.RaftCommand_Nack{
			Nack: &pb.NackRequest{
				QueueName: QueueName,
				Id:        id,
				Priority:  priority,
				Metadata:  metadata,
			},
		},
	}
	data, err := proto.Marshal(cmd)
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
	cmd := &pb.RaftCommand{
		Cmd: &pb.RaftCommand_UpdatePriority{
			UpdatePriority: &pb.UpdatePriorityRequest{
				Id:        id,
				QueueName: queueName,
				Priority:  priority,
			},
		},
	}
	data, err := proto.Marshal(cmd)
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
	cmd := &pb.RaftCommand{
		Cmd: &pb.RaftCommand_CreateQueue{
			CreateQueue: &pb.CreateQueueRequest{
				Type: queueType,
				Name: queueName,
				Settings: &pb.QueueSettings{
					Strategy: pb.QueueSettings_Strategy(
						pb.QueueSettings_Strategy_value[settings.Strategy],
					),
					MaxUnacked: uint32(settings.MaxUnacked),
				},
			},
		},
	}
	data, err := proto.Marshal(cmd)
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
	cmd := &pb.RaftCommand{
		Cmd: &pb.RaftCommand_DeleteQueue{
			DeleteQueue: &pb.DeleteQueueRequest{
				Name: queueName,
			},
		},
	}
	data, err := proto.Marshal(cmd)
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
