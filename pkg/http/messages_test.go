package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/kgantsov/doq/pkg/queue"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

func TestEnqueueDequeue(t *testing.T) {
	_, api := humatest.New(t)

	h := &Handler{
		node: newTestNode(),
	}
	h.RegisterRoutes(api)

	type ErrorOutput struct {
		Title  string `json:"title"`
		Status int    `json:"status"`
		Detail string `json:"detail"`
	}

	h.node.CreateQueue("delayed", "my-queue")

	resp := api.Post("/API/v1/queues/my-queue/messages", map[string]any{
		"content":  "{\"user_id\": 1, \"name\": \"John\"}",
		"priority": 100,
	})

	enqueueOutput := &EnqueueOutputBody{}

	json.Unmarshal(resp.Body.Bytes(), enqueueOutput)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "ENQUEUED", enqueueOutput.Status)
	assert.Equal(t, uint64(1), enqueueOutput.ID)
	assert.Equal(t, int64(100), enqueueOutput.Priority)
	assert.Equal(t, "{\"user_id\": 1, \"name\": \"John\"}", enqueueOutput.Content)

	resp = api.Post("/API/v1/queues/my-queue/messages", map[string]any{
		"content":  "{\"user_id\": 2, \"name\": \"Jane\"}",
		"priority": 100,
	})

	enqueueOutput = &EnqueueOutputBody{}

	json.Unmarshal(resp.Body.Bytes(), enqueueOutput)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "ENQUEUED", enqueueOutput.Status)
	assert.Equal(t, uint64(2), enqueueOutput.ID)
	assert.Equal(t, int64(100), enqueueOutput.Priority)
	assert.Equal(t, "{\"user_id\": 2, \"name\": \"Jane\"}", enqueueOutput.Content)

	resp = api.Get("/API/v1/queues/my-queue/messages")

	enqueueOutput = &EnqueueOutputBody{}

	json.Unmarshal(resp.Body.Bytes(), enqueueOutput)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "DEQUEUED", enqueueOutput.Status)
	assert.Equal(t, uint64(1), enqueueOutput.ID)
	assert.Equal(t, int64(100), enqueueOutput.Priority)
	assert.Equal(t, "{\"user_id\": 1, \"name\": \"John\"}", enqueueOutput.Content)

	resp = api.Post(
		fmt.Sprintf("/API/v1/queues/my-queue/messages/%d/ack", enqueueOutput.ID),
		map[string]any{},
	)

	ackOutput := &AckOutputBody{}

	json.Unmarshal(resp.Body.Bytes(), ackOutput)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "ACKNOWLEDGED", ackOutput.Status)
	assert.Equal(t, uint64(1), ackOutput.ID)

	enqueueOutput = &EnqueueOutputBody{}

	resp = api.Get("/API/v1/queues/my-queue/messages")

	json.Unmarshal(resp.Body.Bytes(), enqueueOutput)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "DEQUEUED", enqueueOutput.Status)
	assert.Equal(t, uint64(2), enqueueOutput.ID)
	assert.Equal(t, int64(100), enqueueOutput.Priority)
	assert.Equal(t, "{\"user_id\": 2, \"name\": \"Jane\"}", enqueueOutput.Content)

	resp = api.Post(
		fmt.Sprintf("/API/v1/queues/my-queue/messages/%d/ack", enqueueOutput.ID),
		map[string]any{},
	)

	ackOutput = &AckOutputBody{}

	json.Unmarshal(resp.Body.Bytes(), ackOutput)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "ACKNOWLEDGED", ackOutput.Status)
	assert.Equal(t, uint64(2), ackOutput.ID)
}

func TestNack(t *testing.T) {
	_, api := humatest.New(t)

	h := &Handler{
		node: newTestNode(),
	}
	h.RegisterRoutes(api)

	type ErrorOutput struct {
		Title  string `json:"title"`
		Status int    `json:"status"`
		Detail string `json:"detail"`
	}

	h.node.CreateQueue("delayed", "my-queue")

	resp := api.Post("/API/v1/queues/my-queue/messages", map[string]any{
		"content":  "{\"user_id\": 1, \"name\": \"John\"}",
		"priority": 100,
	})

	enqueueOutput := &EnqueueOutputBody{}

	json.Unmarshal(resp.Body.Bytes(), enqueueOutput)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "ENQUEUED", enqueueOutput.Status)
	assert.Equal(t, uint64(1), enqueueOutput.ID)
	assert.Equal(t, int64(100), enqueueOutput.Priority)
	assert.Equal(t, "{\"user_id\": 1, \"name\": \"John\"}", enqueueOutput.Content)

	resp = api.Get("/API/v1/queues/my-queue/messages?ack=false")

	dequeueOutput := &DequeueOutputBody{}

	json.Unmarshal(resp.Body.Bytes(), dequeueOutput)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "DEQUEUED", dequeueOutput.Status)
	assert.Equal(t, uint64(1), dequeueOutput.ID)
	assert.Equal(t, int64(100), dequeueOutput.Priority)
	assert.Equal(t, "{\"user_id\": 1, \"name\": \"John\"}", enqueueOutput.Content)

	resp = api.Post(
		fmt.Sprintf("/API/v1/queues/my-queue/messages/%d/nack", dequeueOutput.ID),
		map[string]any{},
	)

	nackOutput := &NackOutputBody{}

	json.Unmarshal(resp.Body.Bytes(), nackOutput)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "UNACKNOWLEDGED", nackOutput.Status)
	assert.Equal(t, uint64(1), nackOutput.ID)

	resp = api.Get("/API/v1/queues/my-queue/messages?ack=true")

	dequeueOutput = &DequeueOutputBody{}

	json.Unmarshal(resp.Body.Bytes(), dequeueOutput)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "DEQUEUED", dequeueOutput.Status)
	assert.Equal(t, uint64(1), dequeueOutput.ID)
	assert.Equal(t, int64(100), dequeueOutput.Priority)
	assert.Equal(t, "{\"user_id\": 1, \"name\": \"John\"}", enqueueOutput.Content)
}

func TestUpdatePriority(t *testing.T) {
	_, api := humatest.New(t)

	// var httpClient = &http.Client{
	// 	Timeout: time.Second * 10,
	// }

	h := &Handler{
		node: newTestNode(),
	}
	h.RegisterRoutes(api)

	h.node.CreateQueue("delayed", "my-queue-1")

	type ErrorOutput struct {
		Title  string `json:"title"`
		Status int    `json:"status"`
		Detail string `json:"detail"`
	}

	resp := api.Post("/API/v1/queues/my-queue-1/messages", map[string]any{
		"content":  "{\"user_id\": 1, \"name\": \"John\"}",
		"priority": 100,
	})

	enqueueOutput := &EnqueueOutputBody{}

	json.Unmarshal(resp.Body.Bytes(), enqueueOutput)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "ENQUEUED", enqueueOutput.Status)
	assert.Equal(t, uint64(1), enqueueOutput.ID)
	assert.Equal(t, int64(100), enqueueOutput.Priority)
	assert.Equal(t, "{\"user_id\": 1, \"name\": \"John\"}", enqueueOutput.Content)

	resp = api.Put(
		fmt.Sprintf("/API/v1/queues/my-queue-1/messages/%d/priority", enqueueOutput.ID),
		map[string]any{
			"priority": 256,
		},
	)

	updatePriorityOutput := &UpdatePriorityOutputBody{}

	json.Unmarshal(resp.Body.Bytes(), updatePriorityOutput)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "UPDATED", updatePriorityOutput.Status)
	assert.Equal(t, uint64(1), updatePriorityOutput.ID)
	assert.Equal(t, int64(256), updatePriorityOutput.Priority)

	resp = api.Get("/API/v1/queues/my-queue-1/messages")

	enqueueOutput = &EnqueueOutputBody{}

	json.Unmarshal(resp.Body.Bytes(), enqueueOutput)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "DEQUEUED", enqueueOutput.Status)
	assert.Equal(t, uint64(1), enqueueOutput.ID)
	assert.Equal(t, int64(256), enqueueOutput.Priority)
	assert.Equal(t, "{\"user_id\": 1, \"name\": \"John\"}", enqueueOutput.Content)
}

type testNode struct {
	nextID   uint64
	leader   string
	messages map[uint64]*queue.Message
	acks     map[uint64]*queue.Message
	queues   map[string]*queue.DelayedPriorityQueue
}

func newTestNode() *testNode {
	return &testNode{
		messages: make(map[uint64]*queue.Message),
		acks:     make(map[uint64]*queue.Message),
		queues:   make(map[string]*queue.DelayedPriorityQueue),
	}
}

func (n *testNode) Join(nodeID, addr string) error {
	return nil
}

func (n *testNode) Leader() string {
	u, _ := url.ParseRequestURI(fmt.Sprintf("http://%s", n.leader))

	return fmt.Sprintf("http://%s:8000/API/v1/queues", u.Hostname())
}
func (n *testNode) IsLeader() bool {
	return true
}

func (n *testNode) GenerateID() uint64 {
	n.nextID++
	return n.nextID
}

func (n *testNode) GetQueues() []*queue.QueueInfo {
	queues := make([]*queue.QueueInfo, 0, len(n.queues))
	for name, q := range n.queues {
		queues = append(queues, &queue.QueueInfo{
			Name:    name,
			Type:    "delayed",
			Ready:   int64(q.Len()),
			Unacked: 0,
			Total:   int64(q.Len()),
			Stats: &queue.QueueStats{
				EnqueueRPS: 1.6,
				DequeueRPS: 1.1,
				AckRPS:     1.1,
				NackRPS:    0,
			},
		})
	}

	return queues
}

func (n *testNode) GetQueueInfo(queueName string) (*queue.QueueInfo, error) {
	q, ok := n.queues[queueName]
	if !ok {
		return nil, queue.ErrQueueNotFound
	}

	return &queue.QueueInfo{
		Name:    queueName,
		Type:    "delayed",
		Ready:   int64(q.Len()),
		Unacked: 0,
		Total:   int64(q.Len()),
		Stats: &queue.QueueStats{
			EnqueueRPS: 1.6,
			DequeueRPS: 1.1,
			AckRPS:     1.1,
			NackRPS:    0,
		},
	}, nil
}

func (n *testNode) PrometheusRegistry() prometheus.Registerer {
	return nil
}

func (n *testNode) CreateQueue(queueType, queueName string) error {
	n.queues[queueName] = queue.NewDelayedPriorityQueue(true)
	return nil
}

func (n *testNode) DeleteQueue(queueName string) error {
	_, ok := n.queues[queueName]
	if !ok {
		return queue.ErrQueueNotFound
	}

	delete(n.queues, queueName)
	return nil
}

func (n *testNode) Enqueue(
	queueName string, group string, priority int64, content string,
) (*queue.Message, error) {
	q, ok := n.queues[queueName]
	if !ok {
		return &queue.Message{}, queue.ErrQueueNotFound
	}

	n.nextID++
	message := &queue.Message{ID: n.nextID, Group: group, Priority: priority, Content: content}
	n.messages[message.ID] = message
	q.Enqueue(group, &queue.Item{ID: message.ID, Priority: message.Priority})
	return message, nil
}

func (n *testNode) Dequeue(QueueName string, ack bool) (*queue.Message, error) {
	q, ok := n.queues[QueueName]
	if !ok {
		return &queue.Message{}, queue.ErrQueueNotFound
	}

	if q.Len() == 0 {
		return nil, queue.ErrEmptyQueue
	}

	item := q.Dequeue()

	message := n.messages[item.ID]

	if ack {
		delete(n.messages, item.ID)
	} else {
		n.acks[item.ID] = message
	}

	return message, nil
}

func (n *testNode) Ack(QueueName string, id uint64) error {
	_, ok := n.queues[QueueName]
	if !ok {
		return queue.ErrQueueNotFound
	}

	if _, ok := n.acks[id]; !ok {
		return queue.ErrMessageNotFound
	}
	delete(n.acks, id)
	delete(n.messages, id)
	return nil
}

func (n *testNode) Nack(QueueName string, id uint64) error {
	q, ok := n.queues[QueueName]
	if !ok {
		return queue.ErrQueueNotFound
	}

	message, ok := n.acks[id]
	if !ok {
		return queue.ErrMessageNotFound
	}

	q.Enqueue(message.Group, &queue.Item{ID: message.ID, Priority: message.Priority})

	delete(n.acks, id)
	return nil
}

func (n *testNode) GetByID(id uint64) (*queue.Message, error) {
	for _, m := range n.messages {
		if m.ID == id {
			return m, nil
		}
	}
	return nil, queue.ErrMessageNotFound
}

func (n *testNode) UpdatePriority(queueName string, id uint64, priority int64) error {
	q, ok := n.queues[queueName]
	if !ok {
		return queue.ErrQueueNotFound
	}

	message, ok := n.messages[id]
	if !ok {
		return queue.ErrMessageNotFound
	}

	message.Priority = priority
	q.UpdatePriority(message.Group, id, priority)
	return nil
}
