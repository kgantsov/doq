package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/gofiber/fiber/v2"
	"github.com/kgantsov/doq/pkg/queue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	addr := "8080"

	service := NewHttpService(addr, newTestNode())

	assert.NotNil(t, service)
	assert.NotNil(t, service.router)
	assert.NotNil(t, service.api)
	assert.NotNil(t, service.h)
	assert.Equal(t, addr, service.addr)

	tests := []struct {
		description  string
		method       string
		url          string
		expectedCode int
	}{
		{"Healthcheck Middleware", "GET", "/readyz", fiber.StatusOK},
		{"Prometheus Middleware", "GET", "/metrics", fiber.StatusOK},
		{"Monitor Middleware", "GET", "/service/metrics", fiber.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.url, nil)
			resp, err := service.router.Test(req)

			require.NoError(t, err)
			assert.Equal(t, tt.expectedCode, resp.StatusCode)
		})
	}

	// check that the correct headers are set by middlewares
	jsonBody := []byte(`{"content": "{\"user_id\": 1}", "priority": 60}`)
	bodyReader := bytes.NewReader(jsonBody)
	req := httptest.NewRequest("POST", "/API/v1/queues/my-queue/messages", bodyReader)
	resp, err := service.router.Test(req)

	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	require.Equal(t, "0", resp.Header.Get(fiber.HeaderXXSSProtection))
	require.Equal(t, "nosniff", resp.Header.Get(fiber.HeaderXContentTypeOptions))
	require.Equal(t, "SAMEORIGIN", resp.Header.Get(fiber.HeaderXFrameOptions))
	require.Equal(t, "", resp.Header.Get(fiber.HeaderContentSecurityPolicy))
	require.Equal(t, "no-referrer", resp.Header.Get(fiber.HeaderReferrerPolicy))
	require.Equal(t, "", resp.Header.Get(fiber.HeaderPermissionsPolicy))
	require.Equal(t, "require-corp", resp.Header.Get("Cross-Origin-Embedder-Policy"))
	require.Equal(t, "same-origin", resp.Header.Get("Cross-Origin-Opener-Policy"))
	require.Equal(t, "same-origin", resp.Header.Get("Cross-Origin-Resource-Policy"))
	require.Equal(t, "?1", resp.Header.Get("Origin-Agent-Cluster"))
	require.Equal(t, "off", resp.Header.Get("X-DNS-Prefetch-Control"))
	require.Equal(t, "noopen", resp.Header.Get("X-Download-Options"))
	require.Equal(t, "none", resp.Header.Get("X-Permitted-Cross-Domain-Policies"))
	require.NotEqual(t, "", resp.Header.Get("X-Request-Id"))
}

func TestEnqueueDequeue(t *testing.T) {
	_, api := humatest.New(t)

	var httpClient = &http.Client{
		Timeout: time.Second * 10,
	}

	h := &Handler{
		node:       newTestNode(),
		httpClient: httpClient,
	}
	h.RegisterRoutes(api)

	type ErrorOutput struct {
		Title  string `json:"title"`
		Status int    `json:"status"`
		Detail string `json:"detail"`
	}

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

func TestUpdatePriority(t *testing.T) {
	_, api := humatest.New(t)

	var httpClient = &http.Client{
		Timeout: time.Second * 10,
	}

	h := &Handler{
		node:       newTestNode(),
		httpClient: httpClient,
	}
	h.RegisterRoutes(api)

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

	resp = api.Get("/API/v1/queues/my-queue/messages")

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
	messages []*queue.Message
	acks     map[uint64]bool
}

func newTestNode() *testNode {
	return &testNode{
		messages: []*queue.Message{},
		acks:     make(map[uint64]bool),
	}
}

func (n *testNode) Leader() string {
	return n.leader
}
func (n *testNode) IsLeader() bool {
	return true
}

func (n *testNode) Enqueue(queueName string, priority int64, content string) (*queue.Message, error) {
	n.nextID++
	message := &queue.Message{ID: n.nextID, Priority: priority, Content: content}
	n.messages = append(n.messages, message)
	return message, nil
}
func (n *testNode) Dequeue(QueueName string, ack bool) (*queue.Message, error) {
	if len(n.messages) == 0 {
		return nil, queue.ErrEmptyQueue
	}
	message := n.messages[0]
	n.messages = n.messages[1:]

	n.acks[message.ID] = false

	return message, nil
}
func (n *testNode) Ack(QueueName string, id uint64) error {
	if _, ok := n.acks[id]; !ok {
		return queue.ErrMessageNotFound
	}
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
	for _, m := range n.messages {
		if m.ID == id {
			m.Priority = priority
			return nil
		}
	}
	return queue.ErrMessageNotFound
}
