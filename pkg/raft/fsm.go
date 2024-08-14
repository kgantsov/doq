package raft

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/hashicorp/raft"
	"github.com/kgantsov/doq/pkg/queue"
	"github.com/rs/zerolog/log"
)

type Command struct {
	ID        uint64 `json:"id,omitempty"`
	Op        string `json:"op"`
	QueueName string `json:"queue_name"`
	Priority  int64  `json:"priority"`
	Content   string `json:"content"`
}

type FSM struct {
	NodeID       string
	queueManager *queue.QueueManager
}

type FSMResponse struct {
	QueueName string
	ID        uint64
	Priority  int64
	Content   string
	error     error
}

func (f *FSM) Apply(raftLog *raft.Log) interface{} {
	var c Command
	if err := json.Unmarshal(raftLog.Data, &c); err != nil {
		panic(fmt.Sprintf("failed to unmarshal command: %s", err.Error()))
	}

	switch c.Op {
	case "enqueue":
		queue, err := f.queueManager.GetQueue(c.QueueName)
		if err != nil {
			return &FSMResponse{
				QueueName: c.QueueName,
				error:     fmt.Errorf("Failed to get a queue: %s", c.QueueName),
			}
		}

		msg, err := queue.Enqueue(c.Priority, c.Content)

		log.Debug().Msgf("Node %s Enqueued a message: %+v %v", f.NodeID, msg, err)

		if err != nil {
			return &FSMResponse{
				QueueName: c.QueueName,
				error:     fmt.Errorf("Failed to enqueue a message to a queue: %s", c.QueueName),
			}
		}

		return &FSMResponse{
			QueueName: c.QueueName,
			ID:        msg.ID,
			Priority:  msg.Priority,
			Content:   msg.Content,
			error:     nil,
		}
	case "dequeue":
		q, err := f.queueManager.GetQueue(c.QueueName)
		if err != nil {
			return &FSMResponse{
				QueueName: c.QueueName,
				error:     fmt.Errorf("Failed to get a queue: %s", c.QueueName),
			}
		}

		msg, err := q.Dequeue()

		log.Debug().Msgf("Node %s Dequeued a message: %+v %v", f.NodeID, msg, err)

		if err != nil {
			if err == queue.ErrEmptyQueue {
				return &FSMResponse{
					QueueName: c.QueueName,
					error:     fmt.Errorf("Queue is empty: %s", c.QueueName),
				}
			}
			return &FSMResponse{
				QueueName: c.QueueName,
				error:     fmt.Errorf("Failed to dequeue a message from a queue: %s", c.QueueName),
			}
		}

		return &FSMResponse{
			QueueName: c.QueueName,
			ID:        msg.ID,
			Priority:  msg.Priority,
			Content:   msg.Content,
			error:     nil,
		}
	case "updatePriority":
		q, err := f.queueManager.GetQueue(c.QueueName)
		if err != nil {
			return &FSMResponse{
				QueueName: c.QueueName,
				error:     fmt.Errorf("Failed to get a queue: %s", c.QueueName),
			}
		}

		err = q.UpdatePriority(c.ID, c.Priority)

		log.Debug().Msgf(
			"Node %s Updated priority for a message: %d %d %v",
			f.NodeID,
			c.ID,
			c.Priority,
			err,
		)

		if err != nil {
			if err == queue.ErrEmptyQueue {
				return &FSMResponse{
					QueueName: c.QueueName,
					error:     fmt.Errorf("Queue is empty: %s", c.QueueName),
				}
			}
			return &FSMResponse{
				QueueName: c.QueueName,
				error:     fmt.Errorf("Failed to dequeue a message from a queue: %s", c.QueueName),
			}
		}

		return &FSMResponse{
			QueueName: c.QueueName,
			ID:        c.ID,
			Priority:  c.Priority,
			Content:   "",
			error:     nil,
		}
	}

	return &FSMResponse{QueueName: c.QueueName, error: fmt.Errorf("Unknown command: %s", c.Op)}
}

func (f *FSM) Snapshot() (raft.FSMSnapshot, error) {
	return &FSMSnapshot{queueManager: f.queueManager}, nil
}

func (f *FSM) Restore(rc io.ReadCloser) error {
	// var store map[string]string
	// if err := json.NewDecoder(rc).Decode(&store); err != nil {
	// 	return err
	// }

	// f.queueManager = queue.NewQueueManager()
	return nil
}

type FSMSnapshot struct {
	queueManager *queue.QueueManager
}

func (f *FSMSnapshot) Persist(sink raft.SnapshotSink) error {
	// if err := f.store.CopyLogs(sink); err != nil {
	// 	log.Debug().Msg("Error copying logs to sink")
	// 	sink.Cancel()
	// 	return err
	// }
	return nil
}

func (f *FSMSnapshot) Release() {}
