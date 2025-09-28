package raft

import (
	"fmt"
	"io"
	"sync"

	"github.com/hashicorp/raft"
	"github.com/kgantsov/doq/pkg/config"
	"github.com/kgantsov/doq/pkg/entity"
	"github.com/kgantsov/doq/pkg/errors"
	pb "github.com/kgantsov/doq/pkg/proto"
	"github.com/kgantsov/doq/pkg/queue"
	"github.com/kgantsov/doq/pkg/storage"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"

	"github.com/dgraph-io/badger/v4"
)

type FSM struct {
	NodeID       string
	queueManager *queue.QueueManager
	db           *badger.DB
	config       *config.Config

	leaderConfig *LeaderConfig

	mu sync.Mutex
}

type FSMResponse struct {
	QueueName string
	ID        uint64
	Group     string
	Priority  int64
	Content   string
	Metadata  map[string]string
	error     error
}

func (f *FSM) Apply(raftLog *raft.Log) interface{} {
	f.mu.Lock()
	defer f.mu.Unlock()

	var c pb.RaftCommand
	if err := proto.Unmarshal(raftLog.Data, &c); err != nil {
		return &FSMResponse{error: err}
	}

	switch command := c.Cmd.(type) {
	case *pb.RaftCommand_LeaderConfigChange:
		return f.applyLeaderConfigChange(command)
	case *pb.RaftCommand_Enqueue:
		return f.applyEnqueue(command)
	case *pb.RaftCommand_Dequeue:
		return f.applyDequeue(command)
	case *pb.RaftCommand_Get:
		return f.applyGet(command)
	case *pb.RaftCommand_Delete:
		return f.applyDelete(command)
	case *pb.RaftCommand_Ack:
		return f.applyAck(command)
	case *pb.RaftCommand_Nack:
		return f.applyNack(command)
	case *pb.RaftCommand_UpdatePriority:
		return f.applyUpdatePriority(command)
	case *pb.RaftCommand_CreateQueue:
		return f.applyCreateQueue(command)
	case *pb.RaftCommand_DeleteQueue:
		return f.applyDeleteQueue(command)
	default:
		return fmt.Errorf("unknown command: %s", c.Cmd)
	}
}

func (f *FSM) applyLeaderConfigChange(payload *pb.RaftCommand_LeaderConfigChange) interface{} {
	log.Info().Msgf(
		"Leader config change: %s at %s (gRPC: %s)",
		payload.LeaderConfigChange.NodeId,
		payload.LeaderConfigChange.RaftAddr,
		payload.LeaderConfigChange.GrpcAddr,
	)

	f.leaderConfig.SetLeaderConfig(
		payload.LeaderConfigChange.NodeId,
		payload.LeaderConfigChange.RaftAddr,
		payload.LeaderConfigChange.GrpcAddr,
	)
	return nil
}

func (f *FSM) applyEnqueue(payload *pb.RaftCommand_Enqueue) *FSMResponse {
	queue, err := f.queueManager.GetQueue(payload.Enqueue.QueueName)
	if err != nil {
		return &FSMResponse{
			QueueName: payload.Enqueue.QueueName,
			error:     fmt.Errorf("Failed to get a queue: %s", payload.Enqueue.QueueName),
		}
	}

	msg, err := queue.Enqueue(
		payload.Enqueue.Id,
		payload.Enqueue.Group,
		payload.Enqueue.Priority,
		payload.Enqueue.Content,
		payload.Enqueue.Metadata,
	)

	log.Debug().Msgf("Node %s Enqueued a message: %+v %v", f.NodeID, msg, err)

	if err != nil {
		return &FSMResponse{
			QueueName: payload.Enqueue.QueueName,
			error: fmt.Errorf(
				"Failed to enqueue a message to a queue: %s", payload.Enqueue.QueueName,
			),
		}
	}

	return &FSMResponse{
		QueueName: payload.Enqueue.QueueName,
		ID:        msg.ID,
		Group:     msg.Group,
		Priority:  msg.Priority,
		Content:   msg.Content,
		Metadata:  msg.Metadata,
		error:     nil,
	}
}

func (f *FSM) applyDequeue(payload *pb.RaftCommand_Dequeue) *FSMResponse {
	q, err := f.queueManager.GetQueue(payload.Dequeue.QueueName)
	if err != nil {
		return &FSMResponse{
			QueueName: payload.Dequeue.QueueName,
			error:     fmt.Errorf("Failed to get a queue: %s", payload.Dequeue.QueueName),
		}
	}

	msg, err := q.Dequeue(payload.Dequeue.Ack)

	log.Debug().Msgf("Node %s Dequeued a message: %+v %v", f.NodeID, msg, err)

	if err != nil {
		if err == errors.ErrEmptyQueue {
			return &FSMResponse{
				QueueName: payload.Dequeue.QueueName,
				error:     fmt.Errorf("Queue is empty: %s", payload.Dequeue.QueueName),
			}
		}
		return &FSMResponse{
			QueueName: payload.Dequeue.QueueName,
			error: fmt.Errorf(
				"Failed to dequeue a message from a queue: %s", payload.Dequeue.QueueName,
			),
		}
	}

	return &FSMResponse{
		QueueName: payload.Dequeue.QueueName,
		ID:        msg.ID,
		Group:     msg.Group,
		Priority:  msg.Priority,
		Content:   msg.Content,
		Metadata:  msg.Metadata,
		error:     nil,
	}
}

func (f *FSM) applyGet(payload *pb.RaftCommand_Get) *FSMResponse {
	q, err := f.queueManager.GetQueue(payload.Get.QueueName)
	if err != nil {
		return &FSMResponse{
			QueueName: payload.Get.QueueName,
			error:     fmt.Errorf("Failed to get a queue: %s", payload.Get.QueueName),
		}
	}

	msg, err := q.Get(payload.Get.Id)

	log.Debug().Msgf("Node %s got a message: %+v %v", f.NodeID, msg, err)

	if err != nil {
		if err == errors.ErrEmptyQueue {
			return &FSMResponse{
				QueueName: payload.Get.QueueName,
				error:     fmt.Errorf("Queue is empty: %s", payload.Get.QueueName),
			}
		}
		return &FSMResponse{
			QueueName: payload.Get.QueueName,
			error: fmt.Errorf(
				"Failed to get a message from a queue: %s", payload.Get.QueueName,
			),
		}
	}

	return &FSMResponse{
		QueueName: payload.Get.QueueName,
		ID:        msg.ID,
		Group:     msg.Group,
		Priority:  msg.Priority,
		Content:   msg.Content,
		Metadata:  msg.Metadata,
		error:     nil,
	}
}

func (f *FSM) applyDelete(payload *pb.RaftCommand_Delete) *FSMResponse {
	q, err := f.queueManager.GetQueue(payload.Delete.QueueName)
	if err != nil {
		return &FSMResponse{
			QueueName: payload.Delete.QueueName,
			error:     fmt.Errorf("Failed to get a queue: %s", payload.Delete.QueueName),
		}
	}

	err = q.Delete(payload.Delete.Id)

	if err != nil {
		if err == errors.ErrEmptyQueue {
			return &FSMResponse{
				QueueName: payload.Delete.QueueName,
				error:     fmt.Errorf("Queue is empty: %s", payload.Delete.QueueName),
			}
		}

		return &FSMResponse{
			QueueName: payload.Delete.QueueName,
			error: fmt.Errorf(
				"Failed to delete a message from a queue: %s", payload.Delete.QueueName,
			),
		}
	}

	return &FSMResponse{
		QueueName: payload.Delete.QueueName,
		ID:        payload.Delete.Id,
		error:     nil,
	}
}

func (f *FSM) applyAck(payload *pb.RaftCommand_Ack) *FSMResponse {
	q, err := f.queueManager.GetQueue(payload.Ack.QueueName)
	if err != nil {
		return &FSMResponse{
			QueueName: payload.Ack.QueueName,
			error:     fmt.Errorf("Failed to get a queue: %s", payload.Ack.QueueName),
		}
	}

	err = q.Ack(payload.Ack.Id)

	log.Debug().Msgf("Node %s Acked a message: %v", f.NodeID, err)

	if err != nil {
		if err == errors.ErrEmptyQueue {
			return &FSMResponse{
				QueueName: payload.Ack.QueueName,
				error:     fmt.Errorf("Queue is empty: %s", payload.Ack.QueueName),
			}
		} else if err == errors.ErrMessageNotFound {
			return &FSMResponse{
				QueueName: payload.Ack.QueueName,
				error:     fmt.Errorf("Message not found: %s", payload.Ack.QueueName),
			}
		}
		return &FSMResponse{
			QueueName: payload.Ack.QueueName,
			error: fmt.Errorf(
				"Failed to ack a message from a queue: %s", payload.Ack.QueueName,
			),
		}
	}

	return &FSMResponse{
		QueueName: payload.Ack.QueueName,
		ID:        payload.Ack.Id,
		error:     nil,
	}
}

func (f *FSM) applyNack(payload *pb.RaftCommand_Nack) *FSMResponse {
	q, err := f.queueManager.GetQueue(payload.Nack.QueueName)
	if err != nil {
		return &FSMResponse{
			QueueName: payload.Nack.QueueName,
			error:     fmt.Errorf("Failed to get a queue: %s", payload.Nack.QueueName),
		}
	}

	err = q.Nack(payload.Nack.Id, payload.Nack.Priority, payload.Nack.Metadata)

	log.Debug().Msgf("Node %s Nacked a message: %v", f.NodeID, err)

	if err != nil {
		if err == errors.ErrEmptyQueue {
			return &FSMResponse{
				QueueName: payload.Nack.QueueName,
				error:     fmt.Errorf("Queue is empty: %s", payload.Nack.QueueName),
			}
		} else if err == errors.ErrMessageNotFound {
			return &FSMResponse{
				QueueName: payload.Nack.QueueName,
				error:     fmt.Errorf("Message not found: %s", payload.Nack.QueueName),
			}
		}
		return &FSMResponse{
			QueueName: payload.Nack.QueueName,
			error: fmt.Errorf(
				"Failed to nack a message from a queue: %s", payload.Nack.QueueName,
			),
		}
	}

	return &FSMResponse{
		QueueName: payload.Nack.QueueName,
		ID:        payload.Nack.Id,
		Metadata:  payload.Nack.Metadata,
		error:     nil,
	}
}

func (f *FSM) applyUpdatePriority(payload *pb.RaftCommand_UpdatePriority) *FSMResponse {
	q, err := f.queueManager.GetQueue(payload.UpdatePriority.QueueName)
	if err != nil {
		return &FSMResponse{
			QueueName: payload.UpdatePriority.QueueName,
			error:     fmt.Errorf("Failed to get a queue: %s", payload.UpdatePriority.QueueName),
		}
	}

	err = q.UpdatePriority(payload.UpdatePriority.Id, payload.UpdatePriority.Priority)

	log.Debug().Msgf(
		"Node %s Updated priority for a message: %d %d %v",
		f.NodeID,
		payload.UpdatePriority.Id,
		payload.UpdatePriority.Priority,
		err,
	)

	if err != nil {
		if err == errors.ErrEmptyQueue {
			return &FSMResponse{
				QueueName: payload.UpdatePriority.QueueName,
				error:     fmt.Errorf("Queue is empty: %s", payload.UpdatePriority.QueueName),
			}
		}
		return &FSMResponse{
			QueueName: payload.UpdatePriority.QueueName,
			error: fmt.Errorf(
				"Failed to dequeue a message from a queue: %s", payload.UpdatePriority.QueueName,
			),
		}
	}

	return &FSMResponse{
		QueueName: payload.UpdatePriority.QueueName,
		ID:        payload.UpdatePriority.Id,
		Priority:  payload.UpdatePriority.Priority,
		Content:   "",
		error:     nil,
	}
}

func (f *FSM) applyCreateQueue(payload *pb.RaftCommand_CreateQueue) *FSMResponse {
	_, err := f.queueManager.CreateQueue(
		payload.CreateQueue.Type,
		payload.CreateQueue.Name,
		entity.QueueSettings{
			Strategy:   payload.CreateQueue.Settings.Strategy.String(),
			MaxUnacked: int(payload.CreateQueue.Settings.MaxUnacked),
		},
	)
	if err != nil {
		return &FSMResponse{
			QueueName: payload.CreateQueue.Name,
			error:     fmt.Errorf("Failed to create a queue: %s", payload.CreateQueue.Name),
		}
	}

	log.Debug().Msgf("Node %s Created a queue: %s", f.NodeID, payload.CreateQueue.Name)
	return &FSMResponse{
		QueueName: payload.CreateQueue.Name,
		error:     nil,
	}
}

func (f *FSM) applyDeleteQueue(payload *pb.RaftCommand_DeleteQueue) *FSMResponse {
	err := f.queueManager.DeleteQueue(payload.DeleteQueue.Name)
	if err != nil {
		return &FSMResponse{
			QueueName: payload.DeleteQueue.Name,
			error:     fmt.Errorf("Failed to delete a queue: %s", payload.DeleteQueue.Name),
		}
	}

	log.Debug().Msgf("Node %s Deleted a queue: %s", f.NodeID, payload.DeleteQueue.Name)

	return &FSMResponse{
		QueueName: payload.DeleteQueue.Name,
		error:     nil,
	}
}

func (f *FSM) Snapshot() (raft.FSMSnapshot, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	return &FSMSnapshot{
		NodeID:       f.NodeID,
		queueManager: f.queueManager,
	}, nil
}

func (f *FSM) Restore(rc io.ReadCloser) error {
	log.Info().Msgf("=====> Restoring snapshot <=====")

	defer rc.Close()

	f.mu.Lock()
	defer f.mu.Unlock()

	linesTotal := 0
	linesRestored := 0
	queuesTotal := 0

	var q *queue.Queue

	for {
		item, err := storage.ReadSnapshotItem(rc)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		linesTotal++

		switch v := item.Item.(type) {
		case *pb.SnapshotItem_Queue:
			log.Debug().Msgf("Restoring queue: %s", v.Queue.Name)
			q, err = f.queueManager.CreateQueue(v.Queue.Type, v.Queue.Name, entity.QueueSettings{
				Strategy:   v.Queue.Settings.Strategy.String(),
				MaxUnacked: int(v.Queue.Settings.MaxUnacked),
			})
			if err != nil {
				log.Warn().Msgf("Failed to create a queue: %v", err)
				continue
			}
			queuesTotal++
		case *pb.SnapshotItem_Message:
			log.Debug().Msgf(
				"Restoring message: %d %s %d",
				v.Message.Id,
				v.Message.Group,
				v.Message.Priority,
			)
			q.Enqueue(
				v.Message.Id,
				v.Message.Group,
				v.Message.Priority,
				v.Message.Content,
				v.Message.Metadata,
			)
		default:
			return fmt.Errorf("unknown item in snapshot")
		}
		linesRestored++
	}

	log.Warn().Msgf("Restored %d out of %d lines", linesRestored, linesTotal)

	return nil
}

type FSMSnapshot struct {
	NodeID       string
	queueManager *queue.QueueManager
}

func (f *FSMSnapshot) Persist(sink raft.SnapshotSink) error {
	if err := f.queueManager.PersistSnapshot(sink); err != nil {
		log.Debug().Msg("Error copying logs to sink")
		sink.Cancel()
		return err
	}

	return nil
}

func (f *FSMSnapshot) Release() {}
