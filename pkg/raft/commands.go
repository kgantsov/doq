package raft

import (
	"context"
	"time"

	"github.com/hashicorp/raft"
	pb "github.com/kgantsov/doq/pkg/proto"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"

	"github.com/kgantsov/doq/pkg/entity"
	"github.com/kgantsov/doq/pkg/queue"
)

func (n *Node) NotifyLeaderConfiguration() error {
	log.Info().Msgf("Notifying leader configuration: NodeID=%s, RaftAddr=%s, GrpcAddr=%s",
		n.cfg.Cluster.NodeID,
		n.cfg.Raft.Address,
		n.cfg.Grpc.Address,
	)

	cmd := &pb.RaftCommand{
		Cmd: &pb.RaftCommand_LeaderConfigChange{
			LeaderConfigChange: &pb.LeaderConfigChange{
				NodeId:   n.cfg.Cluster.NodeID,
				RaftAddr: n.cfg.Raft.Address,
				GrpcAddr: n.cfg.Grpc.Address,
			},
		},
	}

	data, err := proto.Marshal(cmd)
	if err != nil {
		log.Error().Msgf("Failed to marshal leader config change command: %v", err)
		return err
	}

	f := n.Raft.Apply(data, time.Duration(n.cfg.Raft.ApplyTimeout)*time.Second)

	if f.Error() != nil {
		log.Error().Msgf("Failed to apply leader config change command: %v", f.Error())
		return f.Error()
	}

	return nil
}

func (n *Node) Enqueue(
	queueName string, id uint64, group string, priority int64, content string, metadata map[string]string,
) (*entity.Message, error) {
	req := &pb.EnqueueRequest{
		Id:        id,
		Group:     group,
		Priority:  priority,
		Content:   content,
		Metadata:  metadata,
		QueueName: queueName,
	}

	if n.Raft.State() != raft.Leader {
		leaderGrpcAddr := n.leaderConfig.GetLeaderGrpcAddress()

		msg, err := n.proxy.Enqueue(context.Background(), leaderGrpcAddr, req)
		if err != nil {
			return nil, err
		}
		return &entity.Message{
			ID:       msg.Id,
			Group:    group,
			Priority: msg.Priority,
			Content:  msg.Content,
			Metadata: msg.Metadata,
		}, nil
	}

	if id == 0 {
		req.Id = uint64(n.GenerateID())
	}

	cmd := &pb.RaftCommand{Cmd: &pb.RaftCommand_Enqueue{Enqueue: req}}

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
	req := &pb.DequeueRequest{
		QueueName: QueueName,
		Ack:       ack,
	}

	if n.Raft.State() != raft.Leader {
		leaderGrpcAddr := n.leaderConfig.GetLeaderGrpcAddress()

		msg, err := n.proxy.Dequeue(context.Background(), leaderGrpcAddr, req)
		if err != nil {
			return nil, err
		}

		return &entity.Message{
			ID:       msg.Id,
			Group:    msg.Group,
			Priority: msg.Priority,
			Content:  msg.Content,
			Metadata: msg.Metadata,
		}, nil
	}

	cmd := &pb.RaftCommand{Cmd: &pb.RaftCommand_Dequeue{Dequeue: req}}
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
	req := &pb.GetRequest{
		QueueName: QueueName,
		Id:        id,
	}

	if n.Raft.State() != raft.Leader {
		leaderGrpcAddr := n.leaderConfig.GetLeaderGrpcAddress()

		msg, err := n.proxy.Get(context.Background(), leaderGrpcAddr, req)
		if err != nil {
			return nil, err
		}
		return &entity.Message{
			ID:       msg.Id,
			Group:    msg.Group,
			Priority: msg.Priority,
			Content:  msg.Content,
			Metadata: msg.Metadata,
		}, nil
	}

	cmd := &pb.RaftCommand{Cmd: &pb.RaftCommand_Get{Get: req}}
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
	req := &pb.DeleteRequest{
		QueueName: QueueName,
		Id:        id,
	}

	if n.Raft.State() != raft.Leader {
		leaderGrpcAddr := n.leaderConfig.GetLeaderGrpcAddress()

		_, err := n.proxy.Delete(context.Background(), leaderGrpcAddr, req)
		return err
	}

	cmd := &pb.RaftCommand{Cmd: &pb.RaftCommand_Delete{Delete: req}}

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
	req := &pb.AckRequest{
		QueueName: QueueName,
		Id:        id,
	}

	if n.Raft.State() != raft.Leader {
		leaderGrpcAddr := n.leaderConfig.GetLeaderGrpcAddress()

		_, err := n.proxy.Ack(context.Background(), leaderGrpcAddr, req)
		return err
	}

	cmd := &pb.RaftCommand{Cmd: &pb.RaftCommand_Ack{Ack: req}}
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
	req := &pb.NackRequest{
		QueueName: QueueName,
		Id:        id,
		Priority:  priority,
		Metadata:  metadata,
	}

	if n.Raft.State() != raft.Leader {
		leaderGrpcAddr := n.leaderConfig.GetLeaderGrpcAddress()

		_, err := n.proxy.Nack(context.Background(), leaderGrpcAddr, req)
		return err
	}

	cmd := &pb.RaftCommand{Cmd: &pb.RaftCommand_Nack{Nack: req}}
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

func (n *Node) Touch(QueueName string, id uint64) error {
	req := &pb.TouchRequest{
		QueueName: QueueName,
		Id:        id,
	}

	if n.Raft.State() != raft.Leader {
		leaderGrpcAddr := n.leaderConfig.GetLeaderGrpcAddress()

		_, err := n.proxy.Touch(context.Background(), leaderGrpcAddr, req)
		return err
	}

	cmd := &pb.RaftCommand{Cmd: &pb.RaftCommand_Touch{Touch: req}}
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
	req := &pb.UpdatePriorityRequest{
		QueueName: queueName,
		Id:        id,
		Priority:  priority,
	}

	if n.Raft.State() != raft.Leader {
		leaderGrpcAddr := n.leaderConfig.GetLeaderGrpcAddress()

		_, err := n.proxy.UpdatePriority(context.Background(), leaderGrpcAddr, req)
		return err
	}

	cmd := &pb.RaftCommand{Cmd: &pb.RaftCommand_UpdatePriority{UpdatePriority: req}}
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

	req := &pb.CreateQueueRequest{
		Name: queueName,
		Type: queueType,
		Settings: &pb.QueueSettings{
			Strategy: pb.QueueSettings_Strategy(
				pb.QueueSettings_Strategy_value[settings.Strategy],
			),
			MaxUnacked: uint32(settings.MaxUnacked),
			AckTimeout: settings.AckTimeout,
		},
	}

	if n.Raft.State() != raft.Leader {
		leaderGrpcAddr := n.leaderConfig.GetLeaderGrpcAddress()

		_, err := n.proxy.CreateQueue(context.Background(), leaderGrpcAddr, req)
		return err
	}

	cmd := &pb.RaftCommand{Cmd: &pb.RaftCommand_CreateQueue{CreateQueue: req}}
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

func (n *Node) UpdateQueue(queueName string, settings entity.QueueSettings) error {
	req := &pb.UpdateQueueRequest{
		Name: queueName,
		Settings: &pb.QueueSettings{
			Strategy: pb.QueueSettings_Strategy(
				pb.QueueSettings_Strategy_value[settings.Strategy],
			),
			MaxUnacked: uint32(settings.MaxUnacked),
			AckTimeout: settings.AckTimeout,
		},
	}
	if n.Raft.State() != raft.Leader {
		leaderGrpcAddr := n.leaderConfig.GetLeaderGrpcAddress()

		_, err := n.proxy.UpdateQueue(context.Background(), leaderGrpcAddr, req)
		return err
	}

	cmd := &pb.RaftCommand{Cmd: &pb.RaftCommand_UpdateQueue{UpdateQueue: req}}
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
	req := &pb.DeleteQueueRequest{
		Name: queueName,
	}

	if n.Raft.State() != raft.Leader {
		leaderGrpcAddr := n.leaderConfig.GetLeaderGrpcAddress()

		_, err := n.proxy.DeleteQueue(context.Background(), leaderGrpcAddr, req)
		return err
	}

	cmd := &pb.RaftCommand{Cmd: &pb.RaftCommand_DeleteQueue{DeleteQueue: req}}
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
