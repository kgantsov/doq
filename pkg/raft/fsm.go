package raft

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/hashicorp/raft"
	"github.com/kgantsov/doq/pkg/config"
	"github.com/kgantsov/doq/pkg/queue"
	"github.com/rs/zerolog/log"

	"github.com/dgraph-io/badger/v4"
)

type CreateQueuePayload struct {
	QueueType string `json:"queue_type"`
	QueueName string `json:"queue_name"`
}

type DeleteQueuePayload struct {
	QueueName string `json:"queue_name"`
}

type EnqueuePayload struct {
	ID        uint64 `json:"id"`
	QueueName string `json:"queue_name"`
	Group     string `json:"group"`
	Priority  int64  `json:"priority"`
	Content   string `json:"content"`
}

type DequeuePayload struct {
	QueueName string `json:"queue_name"`
	Ack       bool   `json:"ack"`
}

type AckPayload struct {
	QueueName string `json:"queue_name"`
	ID        uint64 `json:"id"`
}

type NackPayload struct {
	QueueName string `json:"queue_name"`
	ID        uint64 `json:"id"`
}

type UpdatePriorityPayload struct {
	QueueName string `json:"queue_name"`
	ID        uint64 `json:"id"`
	Priority  int64  `json:"priority"`
}

type Command struct {
	Op      string      `json:"op"`
	Payload interface{} `json:"payload"`
}

// Custom unmarshal logic to decode the dynamic payload based on the "op" field
func (c *Command) UnmarshalJSON(data []byte) error {
	// Create an intermediate representation to capture the "op" field
	var temp struct {
		Op      string          `json:"op"`
		Payload json.RawMessage `json:"payload"`
	}

	// Unmarshal the JSON into the intermediate struct
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	c.Op = temp.Op

	// Based on the operation, unmarshal the payload into the appropriate struct
	switch c.Op {
	case "enqueue":
		var payload EnqueuePayload
		if err := json.Unmarshal(temp.Payload, &payload); err != nil {
			return err
		}
		c.Payload = payload
	case "dequeue":
		var payload DequeuePayload
		if err := json.Unmarshal(temp.Payload, &payload); err != nil {
			return err
		}
		c.Payload = payload
	case "ack":
		var payload AckPayload
		if err := json.Unmarshal(temp.Payload, &payload); err != nil {
			return err
		}
		c.Payload = payload
	case "updatePriority":
		var payload UpdatePriorityPayload
		if err := json.Unmarshal(temp.Payload, &payload); err != nil {
			return err
		}
		c.Payload = payload
	case "createQueue":
		var payload CreateQueuePayload
		if err := json.Unmarshal(temp.Payload, &payload); err != nil {
			return err
		}
		c.Payload = payload
	case "deleteQueue":
		var payload DeleteQueuePayload
		if err := json.Unmarshal(temp.Payload, &payload); err != nil {
			return err
		}
		c.Payload = payload
	default:
		return fmt.Errorf("unknown operation: %s", c.Op)
	}

	return nil
}

// Custom marshal logic to encode the dynamic payload
func (c Command) MarshalJSON() ([]byte, error) {
	// Create an intermediate representation to add the "op" field
	type Alias Command
	return json.Marshal(&struct {
		Payload interface{} `json:"payload"`
		Alias
	}{
		Payload: c.Payload,
		Alias:   (Alias)(c),
	})
}

type FSM struct {
	NodeID       string
	queueManager *queue.QueueManager
	db           *badger.DB
	config       *config.Config

	mu sync.Mutex
}

type FSMResponse struct {
	QueueName string
	ID        uint64
	Priority  int64
	Content   string
	error     error
}

func (f *FSM) Apply(raftLog *raft.Log) interface{} {
	f.mu.Lock()
	defer f.mu.Unlock()

	var c Command
	if err := json.Unmarshal(raftLog.Data, &c); err != nil {
		return &FSMResponse{error: err}
	}

	log.Debug().Msgf("Node [%s] Received a command: %+v", f.NodeID, c)

	switch c.Op {
	case "enqueue":
		payload, ok := c.Payload.(EnqueuePayload)
		if !ok {
			return &FSMResponse{error: fmt.Errorf("Failed to decode payload: %v", c.Payload)}
		}
		return f.enqueueApply(payload)
	case "dequeue":
		payload, ok := c.Payload.(DequeuePayload)
		if !ok {
			return &FSMResponse{error: fmt.Errorf("Failed to decode payload: %v %v", c, c.Payload)}
		}
		return f.dequeueApply(payload)
	case "ack":
		payload, ok := c.Payload.(AckPayload)
		if !ok {
			return &FSMResponse{error: fmt.Errorf("Failed to decode payload: %v", c.Payload)}
		}
		return f.ackApply(payload)
	case "updatePriority":
		payload, ok := c.Payload.(UpdatePriorityPayload)
		if !ok {
			return &FSMResponse{error: fmt.Errorf("Failed to decode payload: %v", c.Payload)}
		}
		return f.updatePriorityApply(payload)
	case "createQueue":
		payload, ok := c.Payload.(CreateQueuePayload)
		if !ok {
			return &FSMResponse{error: fmt.Errorf("Failed to decode payload: %v", c.Payload)}
		}
		return f.createQueueApply(payload)
	case "deleteQueue":
		payload, ok := c.Payload.(DeleteQueuePayload)
		if !ok {
			return &FSMResponse{error: fmt.Errorf("Failed to decode payload: %v", c.Payload)}
		}
		return f.deleteQueueApply(payload)
	}

	return &FSMResponse{error: fmt.Errorf("Unknown command: %s", c.Op)}
}

func (f *FSM) enqueueApply(payload EnqueuePayload) *FSMResponse {
	queue, err := f.queueManager.GetQueue(payload.QueueName)
	if err != nil {
		return &FSMResponse{
			QueueName: payload.QueueName,
			error:     fmt.Errorf("Failed to get a queue: %s", payload.QueueName),
		}
	}

	msg, err := queue.Enqueue(payload.ID, payload.Group, payload.Priority, payload.Content)

	log.Debug().Msgf("Node %s Enqueued a message: %+v %v", f.NodeID, msg, err)

	if err != nil {
		return &FSMResponse{
			QueueName: payload.QueueName,
			error:     fmt.Errorf("Failed to enqueue a message to a queue: %s", payload.QueueName),
		}
	}

	return &FSMResponse{
		QueueName: payload.QueueName,
		ID:        msg.ID,
		Priority:  msg.Priority,
		Content:   msg.Content,
		error:     nil,
	}
}

func (f *FSM) dequeueApply(payload DequeuePayload) *FSMResponse {
	q, err := f.queueManager.GetQueue(payload.QueueName)
	if err != nil {
		return &FSMResponse{
			QueueName: payload.QueueName,
			error:     fmt.Errorf("Failed to get a queue: %s", payload.QueueName),
		}
	}

	msg, err := q.Dequeue(payload.Ack)

	log.Debug().Msgf("Node %s Dequeued a message: %+v %v", f.NodeID, msg, err)

	if err != nil {
		if err == queue.ErrEmptyQueue {
			return &FSMResponse{
				QueueName: payload.QueueName,
				error:     fmt.Errorf("Queue is empty: %s", payload.QueueName),
			}
		}
		return &FSMResponse{
			QueueName: payload.QueueName,
			error:     fmt.Errorf("Failed to dequeue a message from a queue: %s", payload.QueueName),
		}
	}

	return &FSMResponse{
		QueueName: payload.QueueName,
		ID:        msg.ID,
		Priority:  msg.Priority,
		Content:   msg.Content,
		error:     nil,
	}
}

func (f *FSM) ackApply(payload AckPayload) *FSMResponse {
	q, err := f.queueManager.GetQueue(payload.QueueName)
	if err != nil {
		return &FSMResponse{
			QueueName: payload.QueueName,
			error:     fmt.Errorf("Failed to get a queue: %s", payload.QueueName),
		}
	}

	err = q.Ack(payload.ID)

	log.Debug().Msgf("Node %s Acked a message: %v", f.NodeID, err)

	if err != nil {
		if err == queue.ErrEmptyQueue {
			return &FSMResponse{
				QueueName: payload.QueueName,
				error:     fmt.Errorf("Queue is empty: %s", payload.QueueName),
			}
		} else if err == queue.ErrMessageNotFound {
			return &FSMResponse{
				QueueName: payload.QueueName,
				error:     fmt.Errorf("Message not found: %s", payload.QueueName),
			}
		}
		return &FSMResponse{
			QueueName: payload.QueueName,
			error:     fmt.Errorf("Failed to ack a message from a queue: %s", payload.QueueName),
		}
	}

	return &FSMResponse{
		QueueName: payload.QueueName,
		ID:        payload.ID,
		error:     nil,
	}
}

func (f *FSM) updatePriorityApply(payload UpdatePriorityPayload) *FSMResponse {
	q, err := f.queueManager.GetQueue(payload.QueueName)
	if err != nil {
		return &FSMResponse{
			QueueName: payload.QueueName,
			error:     fmt.Errorf("Failed to get a queue: %s", payload.QueueName),
		}
	}

	err = q.UpdatePriority(payload.ID, payload.Priority)

	log.Debug().Msgf(
		"Node %s Updated priority for a message: %d %d %v",
		f.NodeID,
		payload.ID,
		payload.Priority,
		err,
	)

	if err != nil {
		if err == queue.ErrEmptyQueue {
			return &FSMResponse{
				QueueName: payload.QueueName,
				error:     fmt.Errorf("Queue is empty: %s", payload.QueueName),
			}
		}
		return &FSMResponse{
			QueueName: payload.QueueName,
			error:     fmt.Errorf("Failed to dequeue a message from a queue: %s", payload.QueueName),
		}
	}

	return &FSMResponse{
		QueueName: payload.QueueName,
		ID:        payload.ID,
		Priority:  payload.Priority,
		Content:   "",
		error:     nil,
	}
}

func (f *FSM) createQueueApply(payload CreateQueuePayload) *FSMResponse {
	_, err := f.queueManager.Create(payload.QueueType, payload.QueueName)
	if err != nil {
		return &FSMResponse{
			QueueName: payload.QueueName,
			error:     fmt.Errorf("Failed to create a queue: %s", payload.QueueName),
		}
	}

	log.Debug().Msgf("Node %s Created a queue: %s", f.NodeID, payload.QueueName)
	return &FSMResponse{
		QueueName: payload.QueueName,
		error:     nil,
	}
}

func (f *FSM) deleteQueueApply(payload DeleteQueuePayload) *FSMResponse {
	err := f.queueManager.Delete(payload.QueueName)
	if err != nil {
		return &FSMResponse{
			QueueName: payload.QueueName,
			error:     fmt.Errorf("Failed to delete a queue: %s", payload.QueueName),
		}
	}

	log.Debug().Msgf("Node %s Deleted a queue: %s", f.NodeID, payload.QueueName)

	return &FSMResponse{
		QueueName: payload.QueueName,
		error:     nil,
	}
}

func (f *FSM) Snapshot() (raft.FSMSnapshot, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	return &FSMSnapshot{
		queueManager: f.queueManager,
		db:           f.db,
		config:       f.config,
		NodeID:       f.NodeID,
	}, nil
}

func (f *FSM) Restore(rc io.ReadCloser) error {
	defer rc.Close()

	f.mu.Lock()
	defer f.mu.Unlock()

	return nil
}

type FSMSnapshot struct {
	NodeID       string
	queueManager *queue.QueueManager
	db           *badger.DB
	config       *config.Config
}

func (f *FSMSnapshot) Persist(sink raft.SnapshotSink) error {
	return nil
}

func (f *FSMSnapshot) Release() {}
