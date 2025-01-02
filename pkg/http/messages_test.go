package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/kgantsov/doq/pkg/queue"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

func TestEnqueueDequeue(t *testing.T) {
	_, api := humatest.New(t)

	h := &Handler{
		node: newTestNode("", true),
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
		"metadata": map[string]string{"retry": "3"},
	})

	enqueueOutput := &EnqueueOutputBody{}

	json.Unmarshal(resp.Body.Bytes(), enqueueOutput)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "ENQUEUED", enqueueOutput.Status)
	assert.Equal(t, "1", enqueueOutput.ID)
	assert.Equal(t, int64(100), enqueueOutput.Priority)
	assert.Equal(t, "{\"user_id\": 1, \"name\": \"John\"}", enqueueOutput.Content)
	assert.Equal(t, "3", enqueueOutput.Metadata["retry"])

	resp = api.Post("/API/v1/queues/my-queue/messages", map[string]any{
		"content":  "{\"user_id\": 2, \"name\": \"Jane\"}",
		"priority": 100,
		"metadata": map[string]string{"retry": "3"},
	})

	enqueueOutput = &EnqueueOutputBody{}

	json.Unmarshal(resp.Body.Bytes(), enqueueOutput)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "ENQUEUED", enqueueOutput.Status)
	assert.Equal(t, "2", enqueueOutput.ID)
	assert.Equal(t, int64(100), enqueueOutput.Priority)
	assert.Equal(t, "{\"user_id\": 2, \"name\": \"Jane\"}", enqueueOutput.Content)
	assert.Equal(t, "3", enqueueOutput.Metadata["retry"])

	resp = api.Get("/API/v1/queues/my-queue/messages")

	enqueueOutput = &EnqueueOutputBody{}

	json.Unmarshal(resp.Body.Bytes(), enqueueOutput)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "DEQUEUED", enqueueOutput.Status)
	assert.Equal(t, "1", enqueueOutput.ID)
	assert.Equal(t, int64(100), enqueueOutput.Priority)
	assert.Equal(t, "{\"user_id\": 1, \"name\": \"John\"}", enqueueOutput.Content)
	assert.Equal(t, "3", enqueueOutput.Metadata["retry"])

	resp = api.Post(
		fmt.Sprintf("/API/v1/queues/my-queue/messages/%s/ack", enqueueOutput.ID),
		map[string]any{},
	)

	ackOutput := &AckOutputBody{}

	json.Unmarshal(resp.Body.Bytes(), ackOutput)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "ACKNOWLEDGED", ackOutput.Status)
	assert.Equal(t, "1", ackOutput.ID)

	enqueueOutput = &EnqueueOutputBody{}

	resp = api.Get("/API/v1/queues/my-queue/messages")

	json.Unmarshal(resp.Body.Bytes(), enqueueOutput)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "DEQUEUED", enqueueOutput.Status)
	assert.Equal(t, "2", enqueueOutput.ID)
	assert.Equal(t, int64(100), enqueueOutput.Priority)
	assert.Equal(t, "{\"user_id\": 2, \"name\": \"Jane\"}", enqueueOutput.Content)
	assert.Equal(t, "3", enqueueOutput.Metadata["retry"])

	resp = api.Post(
		fmt.Sprintf("/API/v1/queues/my-queue/messages/%s/ack", enqueueOutput.ID),
		map[string]any{},
	)

	ackOutput = &AckOutputBody{}

	json.Unmarshal(resp.Body.Bytes(), ackOutput)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "ACKNOWLEDGED", ackOutput.Status)
	assert.Equal(t, "2", ackOutput.ID)
}

func TestNack(t *testing.T) {
	_, api := humatest.New(t)

	h := &Handler{
		node: newTestNode("", true),
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
	assert.Equal(t, "1", enqueueOutput.ID)
	assert.Equal(t, int64(100), enqueueOutput.Priority)
	assert.Equal(t, "{\"user_id\": 1, \"name\": \"John\"}", enqueueOutput.Content)

	resp = api.Get("/API/v1/queues/my-queue/messages?ack=false")

	dequeueOutput := &DequeueOutputBody{}

	json.Unmarshal(resp.Body.Bytes(), dequeueOutput)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "DEQUEUED", dequeueOutput.Status)
	assert.Equal(t, "1", dequeueOutput.ID)
	assert.Equal(t, int64(100), dequeueOutput.Priority)
	assert.Equal(t, "{\"user_id\": 1, \"name\": \"John\"}", enqueueOutput.Content)

	resp = api.Post(
		fmt.Sprintf("/API/v1/queues/my-queue/messages/%s/nack", dequeueOutput.ID),
		map[string]any{
			"metadata": map[string]string{"retry": "3"},
		},
	)

	nackOutput := &NackOutputBody{}

	json.Unmarshal(resp.Body.Bytes(), nackOutput)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "UNACKNOWLEDGED", nackOutput.Status)
	assert.Equal(t, "1", nackOutput.ID)
	assert.Equal(t, "3", nackOutput.Metadata["retry"])

	resp = api.Get("/API/v1/queues/my-queue/messages?ack=true")

	dequeueOutput = &DequeueOutputBody{}

	json.Unmarshal(resp.Body.Bytes(), dequeueOutput)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "DEQUEUED", dequeueOutput.Status)
	assert.Equal(t, "1", dequeueOutput.ID)
	assert.Equal(t, int64(100), dequeueOutput.Priority)
	assert.Equal(t, "{\"user_id\": 1, \"name\": \"John\"}", enqueueOutput.Content)
	assert.Equal(t, "3", dequeueOutput.Metadata["retry"])
}

func TestUpdatePriority(t *testing.T) {
	_, api := humatest.New(t)

	// var httpClient = &http.Client{
	// 	Timeout: time.Second * 10,
	// }

	h := &Handler{
		node: newTestNode("", true),
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
	assert.Equal(t, "1", enqueueOutput.ID)
	assert.Equal(t, int64(100), enqueueOutput.Priority)
	assert.Equal(t, "{\"user_id\": 1, \"name\": \"John\"}", enqueueOutput.Content)

	resp = api.Put(
		fmt.Sprintf("/API/v1/queues/my-queue-1/messages/%s/priority", enqueueOutput.ID),
		map[string]any{
			"priority": 256,
		},
	)

	updatePriorityOutput := &UpdatePriorityOutputBody{}

	json.Unmarshal(resp.Body.Bytes(), updatePriorityOutput)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "UPDATED", updatePriorityOutput.Status)
	assert.Equal(t, "1", updatePriorityOutput.ID)
	assert.Equal(t, int64(256), updatePriorityOutput.Priority)

	resp = api.Get("/API/v1/queues/my-queue-1/messages")

	enqueueOutput = &EnqueueOutputBody{}

	json.Unmarshal(resp.Body.Bytes(), enqueueOutput)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "DEQUEUED", enqueueOutput.Status)
	assert.Equal(t, "1", enqueueOutput.ID)
	assert.Equal(t, int64(256), enqueueOutput.Priority)
	assert.Equal(t, "{\"user_id\": 1, \"name\": \"John\"}", enqueueOutput.Content)
}

func TestEnqueueProxy(t *testing.T) {
	_, api := humatest.New(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/API/v1/queues/my-queue/messages", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "application/json", r.Header.Get("Accept"))

		response := EnqueueOutputBody{
			Status:   "ENQUEUED",
			ID:       "123",
			Group:    "customer-1",
			Priority: 100,
			Content:  "{\"user_id\": 1, \"name\": \"John\"}",
			Metadata: map[string]string{"retry": "3"},
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	h := &Handler{
		node:  newTestNode(strings.Replace(server.URL, "http://", "", 1), false),
		proxy: NewProxy(server.Client(), ""),
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
		"metadata": map[string]string{"retry": "3"},
	})

	enqueueOutput := &EnqueueOutputBody{}

	json.Unmarshal(resp.Body.Bytes(), enqueueOutput)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "ENQUEUED", enqueueOutput.Status)
	assert.Equal(t, "123", enqueueOutput.ID)
	assert.Equal(t, int64(100), enqueueOutput.Priority)
	assert.Equal(t, "{\"user_id\": 1, \"name\": \"John\"}", enqueueOutput.Content)
	assert.Equal(t, "3", enqueueOutput.Metadata["retry"])
}

func TestDequeueProxy(t *testing.T) {
	_, api := humatest.New(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/API/v1/queues/indexing-queue/messages", r.URL.Path)
		assert.Equal(t, "true", r.URL.Query().Get("ack"))
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "application/json", r.Header.Get("Accept"))

		response := DequeueOutputBody{
			Status:   "DEQUEUED",
			ID:       "75",
			Group:    "customer-1",
			Priority: 31,
			Content:  "{\"id\": 114, \"name\": \"test\"}",
			Metadata: map[string]string{"retry": "1"},
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	h := &Handler{
		node:  newTestNode(strings.Replace(server.URL, "http://", "", 1), false),
		proxy: NewProxy(server.Client(), ""),
	}
	h.RegisterRoutes(api)

	type ErrorOutput struct {
		Title  string `json:"title"`
		Status int    `json:"status"`
		Detail string `json:"detail"`
	}

	h.node.CreateQueue("delayed", "indexing-queue")

	resp := api.Get("/API/v1/queues/indexing-queue/messages?ack=true")

	dequeueOutput := &DequeueOutputBody{}

	json.Unmarshal(resp.Body.Bytes(), dequeueOutput)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "DEQUEUED", dequeueOutput.Status)
	assert.Equal(t, "75", dequeueOutput.ID)
	assert.Equal(t, int64(31), dequeueOutput.Priority)
	assert.Equal(t, "{\"id\": 114, \"name\": \"test\"}", dequeueOutput.Content)
	assert.Equal(t, "1", dequeueOutput.Metadata["retry"])
}

func TestAckProxy(t *testing.T) {
	_, api := humatest.New(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/API/v1/queues/indexing-queue/messages/1122/ack", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "application/json", r.Header.Get("Accept"))

		response := AckOutputBody{
			Status: "ACKED",
			ID:     "1122",
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	h := &Handler{
		node:  newTestNode(strings.Replace(server.URL, "http://", "", 1), false),
		proxy: NewProxy(server.Client(), ""),
	}
	h.RegisterRoutes(api)

	type ErrorOutput struct {
		Title  string `json:"title"`
		Status int    `json:"status"`
		Detail string `json:"detail"`
	}

	h.node.CreateQueue("delayed", "indexing-queue")

	resp := api.Post(
		"/API/v1/queues/indexing-queue/messages/1122/ack", map[string]any{})

	output := &AckOutputBody{}

	json.Unmarshal(resp.Body.Bytes(), output)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "1122", output.ID)
	assert.Equal(t, "ACKED", output.Status)
}

func TestNackProxy(t *testing.T) {
	_, api := humatest.New(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/API/v1/queues/indexing-queue/messages/1122/nack", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "application/json", r.Header.Get("Accept"))

		response := AckOutputBody{
			Status: "ACKED",
			ID:     "1122",
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	h := &Handler{
		node:  newTestNode(strings.Replace(server.URL, "http://", "", 1), false),
		proxy: NewProxy(server.Client(), ""),
	}
	h.RegisterRoutes(api)

	type ErrorOutput struct {
		Title  string `json:"title"`
		Status int    `json:"status"`
		Detail string `json:"detail"`
	}

	h.node.CreateQueue("delayed", "indexing-queue")

	resp := api.Post(
		"/API/v1/queues/indexing-queue/messages/1122/nack", map[string]any{})

	output := &NackOutputBody{}

	json.Unmarshal(resp.Body.Bytes(), output)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "1122", output.ID)
	assert.Equal(t, "ACKED", output.Status)
}

func TestUpdatePriorityProxy(t *testing.T) {
	_, api := humatest.New(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/API/v1/queues/indexing-queue/messages/980/priority", r.URL.Path)
		assert.Equal(t, "PUT", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "application/json", r.Header.Get("Accept"))

		response := UpdatePriorityOutputBody{
			Status:   "UPDATED",
			ID:       "5634",
			Priority: 777,
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	h := &Handler{
		node:  newTestNode(strings.Replace(server.URL, "http://", "", 1), false),
		proxy: NewProxy(server.Client(), ""),
	}
	h.RegisterRoutes(api)

	type ErrorOutput struct {
		Title  string `json:"title"`
		Status int    `json:"status"`
		Detail string `json:"detail"`
	}

	h.node.CreateQueue("delayed", "indexing-queue")

	resp := api.Put("/API/v1/queues/indexing-queue/messages/980/priority", map[string]any{
		"priority": 777,
	})

	output := &UpdatePriorityOutputBody{}

	json.Unmarshal(resp.Body.Bytes(), output)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "5634", output.ID)
	assert.Equal(t, "UPDATED", output.Status)
	assert.Equal(t, int64(777), output.Priority)
}

type testNode struct {
	nextID   uint64
	isLeader bool
	leader   string
	messages map[uint64]*queue.Message
	acks     map[uint64]*queue.Message
	queues   map[string]*queue.DelayedPriorityQueue
}

func newTestNode(leader string, isLeader bool) *testNode {
	return &testNode{
		messages: make(map[uint64]*queue.Message),
		acks:     make(map[uint64]*queue.Message),
		queues:   make(map[string]*queue.DelayedPriorityQueue),
		leader:   leader,
		isLeader: isLeader,
	}
}

func (n *testNode) Join(nodeID, addr string) error {
	return nil
}

func (n *testNode) Leader() string {
	return n.leader
	// u, _ := url.ParseRequestURI(fmt.Sprintf("http://%s", n.leader))

	// return fmt.Sprintf("http://%s:8000/API/v1/queues", u.Hostname())
}
func (n *testNode) IsLeader() bool {
	return n.isLeader
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
	queueName string, group string, priority int64, content string, metadata map[string]string,
) (*queue.Message, error) {
	q, ok := n.queues[queueName]
	if !ok {
		return &queue.Message{}, queue.ErrQueueNotFound
	}

	n.nextID++
	message := &queue.Message{
		ID: n.nextID, Group: group, Priority: priority, Content: content, Metadata: metadata,
	}
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

func (n *testNode) Nack(QueueName string, id uint64, metadata map[string]string) error {
	q, ok := n.queues[QueueName]
	if !ok {
		return queue.ErrQueueNotFound
	}

	message, ok := n.acks[id]
	if !ok {
		return queue.ErrMessageNotFound
	}

	message.Metadata = metadata
	n.messages[message.ID] = message

	q.Enqueue(message.Group, &queue.Item{ID: message.ID, Priority: message.Priority})

	delete(n.acks, id)
	return nil
}

func (n *testNode) Get(id uint64) (*queue.Message, error) {
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
