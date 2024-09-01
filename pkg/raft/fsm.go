package raft

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/hashicorp/raft"
	"github.com/kgantsov/doq/pkg/queue"
	"github.com/rs/zerolog/log"

	badgerdb "github.com/kgantsov/doq/pkg/badger-store"
)

type Command struct {
	ID        uint64 `json:"id,omitempty"`
	Op        string `json:"op"`
	QueueType string `json:"queue_type"`
	QueueName string `json:"queue_name"`
	Group     string `json:"group"`
	Priority  int64  `json:"priority"`
	Content   string `json:"content"`
	Ack       bool   `json:"ack,omitempty"`
}

type FSM struct {
	NodeID       string
	queueManager *queue.QueueManager
	store        badgerdb.Store
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
		return &FSMResponse{QueueName: c.QueueName, error: err}
	}

	log.Debug().Msgf("Node %s Received a command: %+v", f.NodeID, c)

	switch c.Op {
	case "enqueue":
		return f.enqueueApply(c)
	case "dequeue":
		return f.dequeueApply(c)
	case "ack":
		return f.ackApply(c)
	case "updatePriority":
		return f.updatePriorityApply(c)
	case "createQueue":
		return f.createQueueApply(c)
	case "deleteQueue":
		return f.deleteQueueApply(c)
	}

	return &FSMResponse{QueueName: c.QueueName, error: fmt.Errorf("Unknown command: %s", c.Op)}
}

func (f *FSM) enqueueApply(c Command) *FSMResponse {
	queue, err := f.queueManager.GetQueue(c.QueueName)
	if err != nil {
		return &FSMResponse{
			QueueName: c.QueueName,
			error:     fmt.Errorf("Failed to get a queue: %s", c.QueueName),
		}
	}

	msg, err := queue.Enqueue(c.Group, c.Priority, c.Content)

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
}

func (f *FSM) dequeueApply(c Command) *FSMResponse {
	q, err := f.queueManager.GetQueue(c.QueueName)
	if err != nil {
		return &FSMResponse{
			QueueName: c.QueueName,
			error:     fmt.Errorf("Failed to get a queue: %s", c.QueueName),
		}
	}

	msg, err := q.Dequeue(c.Ack)

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
}

func (f *FSM) ackApply(c Command) *FSMResponse {
	q, err := f.queueManager.GetQueue(c.QueueName)
	if err != nil {
		return &FSMResponse{
			QueueName: c.QueueName,
			error:     fmt.Errorf("Failed to get a queue: %s", c.QueueName),
		}
	}

	err = q.Ack(c.ID)

	log.Debug().Msgf("Node %s Acked a message: %v", f.NodeID, err)

	if err != nil {
		if err == queue.ErrEmptyQueue {
			return &FSMResponse{
				QueueName: c.QueueName,
				error:     fmt.Errorf("Queue is empty: %s", c.QueueName),
			}
		} else if err == queue.ErrMessageNotFound {
			return &FSMResponse{
				QueueName: c.QueueName,
				error:     fmt.Errorf("Message not found: %s", c.QueueName),
			}
		}
		return &FSMResponse{
			QueueName: c.QueueName,
			error:     fmt.Errorf("Failed to ack a message from a queue: %s", c.QueueName),
		}
	}

	return &FSMResponse{
		QueueName: c.QueueName,
		ID:        c.ID,
		error:     nil,
	}
}

func (f *FSM) updatePriorityApply(c Command) *FSMResponse {
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

func (f *FSM) createQueueApply(c Command) *FSMResponse {
	_, err := f.queueManager.Create(c.QueueType, c.QueueName)
	if err != nil {
		return &FSMResponse{
			QueueName: c.QueueName,
			error:     fmt.Errorf("Failed to create a queue: %s", c.QueueName),
		}
	}

	log.Debug().Msgf("Node %s Created a queue: %s", f.NodeID, c.QueueName)
	return &FSMResponse{
		QueueName: c.QueueName,
		error:     nil,
	}
}

func (f *FSM) deleteQueueApply(c Command) *FSMResponse {
	err := f.queueManager.Delete(c.QueueName)
	if err != nil {
		return &FSMResponse{
			QueueName: c.QueueName,
			error:     fmt.Errorf("Failed to delete a queue: %s", c.QueueName),
		}
	}

	log.Debug().Msgf("Node %s Deleted a queue: %s", f.NodeID, c.QueueName)

	return &FSMResponse{
		QueueName: c.QueueName,
		error:     nil,
	}
}

func (f *FSM) Snapshot() (raft.FSMSnapshot, error) {
	return &FSMSnapshot{queueManager: f.queueManager, store: f.store}, nil
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
	store        badgerdb.Store
}

func (f *FSMSnapshot) Persist(sink raft.SnapshotSink) error {
	if err := f.store.CopyLogs(sink); err != nil {
		log.Debug().Msg("Error copying logs to sink")
		sink.Cancel()
		return err
	}
	return nil
}

func (f *FSMSnapshot) Release() {}
