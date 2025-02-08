package raft

import (
	"bufio"
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
	ID        uint64            `json:"id"`
	QueueName string            `json:"queue_name"`
	Group     string            `json:"group"`
	Priority  int64             `json:"priority"`
	Content   string            `json:"content"`
	Metadata  map[string]string `json:"metadata"`
}

type DequeuePayload struct {
	QueueName string `json:"queue_name"`
	Ack       bool   `json:"ack"`
}

type GetPayload struct {
	QueueName string `json:"queue_name"`
	ID        uint64 `json:"id"`
}

type DeletePayload struct {
	QueueName string `json:"queue_name"`
	ID        uint64 `json:"id"`
}

type AckPayload struct {
	QueueName string `json:"queue_name"`
	ID        uint64 `json:"id"`
}

type NackPayload struct {
	QueueName string            `json:"queue_name"`
	ID        uint64            `json:"id"`
	Priority  int64             `json:"priority"`
	Metadata  map[string]string `json:"metadata"`
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
	case "get":
		var payload GetPayload
		if err := json.Unmarshal(temp.Payload, &payload); err != nil {
			return err
		}
		c.Payload = payload
	case "delete":
		var payload DeletePayload
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
	case "nack":
		var payload NackPayload
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
	Group     string
	Priority  int64
	Content   string
	Metadata  map[string]string
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
		return f.applyEnqueue(payload)
	case "dequeue":
		payload, ok := c.Payload.(DequeuePayload)
		if !ok {
			return &FSMResponse{error: fmt.Errorf("Failed to decode payload: %v %v", c, c.Payload)}
		}
		return f.applyDequeue(payload)
	case "get":
		payload, ok := c.Payload.(GetPayload)
		if !ok {
			return &FSMResponse{error: fmt.Errorf("Failed to decode payload: %v %v", c, c.Payload)}
		}
		return f.applyGet(payload)
	case "delete":
		payload, ok := c.Payload.(DeletePayload)
		if !ok {
			return &FSMResponse{error: fmt.Errorf("Failed to decode payload: %v %v", c, c.Payload)}
		}
		return f.applyDelete(payload)
	case "ack":
		payload, ok := c.Payload.(AckPayload)
		if !ok {
			return &FSMResponse{error: fmt.Errorf("Failed to decode payload: %v", c.Payload)}
		}
		return f.applyAck(payload)
	case "nack":
		payload, ok := c.Payload.(NackPayload)
		if !ok {
			return &FSMResponse{error: fmt.Errorf("Failed to decode payload: %v", c.Payload)}
		}
		return f.applyNack(payload)
	case "updatePriority":
		payload, ok := c.Payload.(UpdatePriorityPayload)
		if !ok {
			return &FSMResponse{error: fmt.Errorf("Failed to decode payload: %v", c.Payload)}
		}
		return f.applyUpdatePriority(payload)
	case "createQueue":
		payload, ok := c.Payload.(CreateQueuePayload)
		if !ok {
			return &FSMResponse{error: fmt.Errorf("Failed to decode payload: %v", c.Payload)}
		}
		return f.applyCreateQueue(payload)
	case "deleteQueue":
		payload, ok := c.Payload.(DeleteQueuePayload)
		if !ok {
			return &FSMResponse{error: fmt.Errorf("Failed to decode payload: %v", c.Payload)}
		}
		return f.applyDeleteQueue(payload)
	}

	return &FSMResponse{error: fmt.Errorf("Unknown command: %s", c.Op)}
}

func (f *FSM) applyEnqueue(payload EnqueuePayload) *FSMResponse {
	queue, err := f.queueManager.GetQueue(payload.QueueName)
	if err != nil {
		return &FSMResponse{
			QueueName: payload.QueueName,
			error:     fmt.Errorf("Failed to get a queue: %s", payload.QueueName),
		}
	}

	msg, err := queue.Enqueue(
		payload.ID, payload.Group, payload.Priority, payload.Content, payload.Metadata,
	)

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
		Group:     msg.Group,
		Priority:  msg.Priority,
		Content:   msg.Content,
		Metadata:  msg.Metadata,
		error:     nil,
	}
}

func (f *FSM) applyDequeue(payload DequeuePayload) *FSMResponse {
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
		Group:     msg.Group,
		Priority:  msg.Priority,
		Content:   msg.Content,
		Metadata:  msg.Metadata,
		error:     nil,
	}
}

func (f *FSM) applyGet(payload GetPayload) *FSMResponse {
	q, err := f.queueManager.GetQueue(payload.QueueName)
	if err != nil {
		return &FSMResponse{
			QueueName: payload.QueueName,
			error:     fmt.Errorf("Failed to get a queue: %s", payload.QueueName),
		}
	}

	msg, err := q.Get(payload.ID)

	log.Debug().Msgf("Node %s got a message: %+v %v", f.NodeID, msg, err)

	if err != nil {
		if err == queue.ErrEmptyQueue {
			return &FSMResponse{
				QueueName: payload.QueueName,
				error:     fmt.Errorf("Queue is empty: %s", payload.QueueName),
			}
		}
		return &FSMResponse{
			QueueName: payload.QueueName,
			error:     fmt.Errorf("Failed to get a message from a queue: %s", payload.QueueName),
		}
	}

	return &FSMResponse{
		QueueName: payload.QueueName,
		ID:        msg.ID,
		Group:     msg.Group,
		Priority:  msg.Priority,
		Content:   msg.Content,
		Metadata:  msg.Metadata,
		error:     nil,
	}
}

func (f *FSM) applyDelete(payload DeletePayload) *FSMResponse {
	q, err := f.queueManager.GetQueue(payload.QueueName)
	if err != nil {
		return &FSMResponse{
			QueueName: payload.QueueName,
			error:     fmt.Errorf("Failed to get a queue: %s", payload.QueueName),
		}
	}

	err = q.Delete(payload.ID)

	if err != nil {
		if err == queue.ErrEmptyQueue {
			return &FSMResponse{
				QueueName: payload.QueueName,
				error:     fmt.Errorf("Queue is empty: %s", payload.QueueName),
			}
		}

		return &FSMResponse{
			QueueName: payload.QueueName,
			error:     fmt.Errorf("Failed to delete a message from a queue: %s", payload.QueueName),
		}
	}

	return &FSMResponse{
		QueueName: payload.QueueName,
		ID:        payload.ID,
		error:     nil,
	}
}

func (f *FSM) applyAck(payload AckPayload) *FSMResponse {
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

func (f *FSM) applyNack(payload NackPayload) *FSMResponse {
	q, err := f.queueManager.GetQueue(payload.QueueName)
	if err != nil {
		return &FSMResponse{
			QueueName: payload.QueueName,
			error:     fmt.Errorf("Failed to get a queue: %s", payload.QueueName),
		}
	}

	err = q.Nack(payload.ID, payload.Priority, payload.Metadata)

	log.Debug().Msgf("Node %s Nacked a message: %v", f.NodeID, err)

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
			error:     fmt.Errorf("Failed to nack a message from a queue: %s", payload.QueueName),
		}
	}

	return &FSMResponse{
		QueueName: payload.QueueName,
		ID:        payload.ID,
		Metadata:  payload.Metadata,
		error:     nil,
	}
}

func (f *FSM) applyUpdatePriority(payload UpdatePriorityPayload) *FSMResponse {
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

func (f *FSM) applyCreateQueue(payload CreateQueuePayload) *FSMResponse {
	_, err := f.queueManager.CreateQueue(payload.QueueType, payload.QueueName)
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

func (f *FSM) applyDeleteQueue(payload DeleteQueuePayload) *FSMResponse {
	err := f.queueManager.DeleteQueue(payload.QueueName)
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
		NodeID:       f.NodeID,
		queueManager: f.queueManager,
	}, nil
}

func (f *FSM) Restore(rc io.ReadCloser) error {
	log.Info().Msgf("=====> Restoring snapshot <=====")

	defer rc.Close()

	f.mu.Lock()
	defer f.mu.Unlock()

	scanner := bufio.NewScanner(rc)
	linesTotal := 0
	linesRestored := 0

	for scanner.Scan() {
		line := scanner.Bytes()
		linesTotal++

		msg, err := queue.MessageFromBytes(line)
		if err != nil {
			log.Warn().Msgf("Failed to unmarshal command: %v %v", err, string(line))
			continue
		}

		q, err := f.queueManager.CreateQueue(msg.QueueType, msg.QueueName)
		if err != nil {
			log.Warn().Msgf("Failed to create a queue: %v", err)
			continue
		}

		q.Enqueue(msg.ID, msg.Group, msg.Priority, msg.Content, msg.Metadata)

		linesRestored++
	}

	if err := scanner.Err(); err != nil {
		log.Info().Msgf(
			"Error while reading snapshot: %v. Restored %d out of %d lines",
			err,
			linesRestored,
			linesTotal,
		)
		return err
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
