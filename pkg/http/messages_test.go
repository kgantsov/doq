package http

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/bwmarrin/snowflake"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/kgantsov/doq/pkg/entity"
	"github.com/kgantsov/doq/pkg/errors"
	"github.com/kgantsov/doq/pkg/metrics"
	"github.com/kgantsov/doq/pkg/queue"
	"github.com/kgantsov/doq/pkg/queue/memory"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

func TestGenerateIDs(t *testing.T) {
	_, api := humatest.New(t)

	idGenerator, err := snowflake.NewNode(1)
	assert.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/API/v1/ids", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "application/json", r.Header.Get("Accept"))

		var request GenerateIdInputBody
		err := json.NewDecoder(r.Body).Decode(&request)
		assert.NoError(t, err)

		var ids []string
		for i := 0; i < request.Number; i++ {
			ids = append(ids, fmt.Sprintf("%d", idGenerator.Generate().Int64()))
		}

		response := GenerateIdOutputBody{IDs: ids}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	h := &Handler{
		node: newTestNode(strings.Replace(server.URL, "http://", "", 1), false),
	}
	h.RegisterRoutes(api)

	type ErrorOutput struct {
		Title  string `json:"title"`
		Status int    `json:"status"`
		Detail string `json:"detail"`
	}

	h.node.CreateQueue("delayed", "indexing-queue", entity.QueueSettings{})

	resp := api.Post(
		"/API/v1/ids", map[string]any{
			"number": 234,
		})

	output := &GenerateIdOutputBody{}

	json.Unmarshal(resp.Body.Bytes(), output)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, 234, len(output.IDs))

	lastId := uint64(0)
	for _, idStr := range output.IDs {
		id, err := strconv.ParseUint(idStr, 10, 64)
		assert.NoError(t, err)
		assert.Greater(t, id, lastId)
		lastId = id
	}
}

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

	h.node.CreateQueue("delayed", "my-queue", entity.QueueSettings{})

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

func TestEnqueueGet(t *testing.T) {
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

	h.node.CreateQueue("delayed", "my-queue", entity.QueueSettings{})

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

	resp = api.Get(fmt.Sprintf("/API/v1/queues/my-queue/messages/%s", enqueueOutput.ID))

	getOutput := &GetOutputBody{}

	json.Unmarshal(resp.Body.Bytes(), getOutput)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "GOT", getOutput.Status)
	assert.Equal(t, "1", getOutput.ID)
	assert.Equal(t, int64(100), getOutput.Priority)
	assert.Equal(t, "{\"user_id\": 1, \"name\": \"John\"}", getOutput.Content)
	assert.Equal(t, "3", getOutput.Metadata["retry"])

	resp = api.Delete(fmt.Sprintf("/API/v1/queues/my-queue/messages/%s", enqueueOutput.ID))

	assert.Equal(t, http.StatusNoContent, resp.Code)

	resp = api.Get(fmt.Sprintf("/API/v1/queues/my-queue/messages/%s", enqueueOutput.ID))

	getOutput = &GetOutputBody{}

	json.Unmarshal(resp.Body.Bytes(), getOutput)
	assert.Equal(t, http.StatusBadRequest, resp.Code)
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

	h.node.CreateQueue("delayed", "my-queue", entity.QueueSettings{})

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
			"priority": 100,
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

	h.node.CreateQueue("delayed", "my-queue-1", entity.QueueSettings{})

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

type testNode struct {
	nextID   uint64
	isLeader bool
	leader   string
	messages map[uint64]*entity.Message
	acks     map[uint64]*entity.Message
	queues   map[string]*memory.DelayedQueue
}

func newTestNode(leader string, isLeader bool) *testNode {
	return &testNode{
		messages: make(map[uint64]*entity.Message),
		acks:     make(map[uint64]*entity.Message),
		queues:   make(map[string]*memory.DelayedQueue),
		leader:   leader,
		isLeader: isLeader,
	}
}

func (n *testNode) Join(nodeID, addr string) error {
	return nil
}

func (n *testNode) Backup(w io.Writer, since uint64) (uint64, error) {

	return 0, nil
}
func (n *testNode) Restore(r io.Reader, maxPendingWrites int) error {

	return nil
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
			Stats: &metrics.Stats{
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
		return nil, errors.ErrQueueNotFound
	}

	return &queue.QueueInfo{
		Name:    queueName,
		Type:    "delayed",
		Ready:   int64(q.Len()),
		Unacked: 0,
		Total:   int64(q.Len()),
		Stats: &metrics.Stats{
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

func (n *testNode) CreateQueue(queueType, queueName string, settings entity.QueueSettings) error {
	n.queues[queueName] = memory.NewDelayedQueue(true)
	return nil
}

func (n *testNode) DeleteQueue(queueName string) error {
	_, ok := n.queues[queueName]
	if !ok {
		return errors.ErrQueueNotFound
	}

	delete(n.queues, queueName)
	return nil
}

func (n *testNode) Enqueue(
	queueName string, id uint64, group string, priority int64, content string, metadata map[string]string,
) (*entity.Message, error) {
	q, ok := n.queues[queueName]
	if !ok {
		return &entity.Message{}, errors.ErrQueueNotFound
	}

	n.nextID++
	if id == 0 {
		id = n.nextID
	}
	message := &entity.Message{
		ID: id, Group: group, Priority: priority, Content: content, Metadata: metadata,
	}
	n.messages[message.ID] = message
	q.Enqueue(group, &memory.Item{ID: message.ID, Priority: message.Priority})
	return message, nil
}

func (n *testNode) Dequeue(QueueName string, ack bool) (*entity.Message, error) {
	q, ok := n.queues[QueueName]
	if !ok {
		return &entity.Message{}, errors.ErrQueueNotFound
	}

	if q.Len() == 0 {
		return nil, errors.ErrEmptyQueue
	}

	item := q.Dequeue(false)

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
		return errors.ErrQueueNotFound
	}

	if _, ok := n.acks[id]; !ok {
		return errors.ErrMessageNotFound
	}
	delete(n.acks, id)
	delete(n.messages, id)
	return nil
}

func (n *testNode) Nack(QueueName string, id uint64, priority int64, metadata map[string]string) error {
	q, ok := n.queues[QueueName]
	if !ok {
		return errors.ErrQueueNotFound
	}

	message, ok := n.acks[id]
	if !ok {
		return errors.ErrMessageNotFound
	}

	if priority != 0 {
		message.Priority = priority
	}

	message.Metadata = metadata
	n.messages[message.ID] = message

	q.Enqueue(message.Group, &memory.Item{ID: message.ID, Priority: message.Priority})

	delete(n.acks, id)
	return nil
}

func (n *testNode) Get(QueueName string, id uint64) (*entity.Message, error) {
	for _, m := range n.messages {
		if m.ID == id {
			return m, nil
		}
	}
	return nil, errors.ErrMessageNotFound
}

func (n *testNode) Delete(QueueName string, id uint64) error {
	q, ok := n.queues[QueueName]
	if !ok {
		return errors.ErrQueueNotFound
	}

	msg, ok := n.messages[id]
	if !ok {
		return errors.ErrMessageNotFound
	}

	q.Delete(msg.Group, id)

	delete(n.acks, id)
	delete(n.messages, id)

	return nil
}

func (n *testNode) UpdatePriority(queueName string, id uint64, priority int64) error {
	q, ok := n.queues[queueName]
	if !ok {
		return errors.ErrQueueNotFound
	}

	message, ok := n.messages[id]
	if !ok {
		return errors.ErrMessageNotFound
	}

	message.Priority = priority
	q.UpdatePriority(message.Group, id, priority)
	return nil
}
