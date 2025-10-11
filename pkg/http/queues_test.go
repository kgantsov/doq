package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/kgantsov/doq/pkg/entity"
	"github.com/stretchr/testify/assert"
)

func TestCreateDeleteQueue(t *testing.T) {
	_, api := humatest.New(t)

	h := &Handler{
		node: NewTestNode("", true),
	}

	h.RegisterRoutes(api)

	type ErrorOutput struct {
		Title  string `json:"title"`
		Status int    `json:"status"`
		Detail string `json:"detail"`
	}

	resp := api.Post("/API/v1/queues", map[string]any{
		"name":     "my-queue",
		"type":     "delayed",
		"settings": map[string]any{},
	})

	createQueueOutput := &CreateQueueOutputBody{}

	json.Unmarshal(resp.Body.Bytes(), createQueueOutput)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "CREATED", createQueueOutput.Status)
	assert.Equal(t, "my-queue", createQueueOutput.Name)
	assert.Equal(t, "delayed", createQueueOutput.Type)

	resp = api.Delete("/API/v1/queues/my-queue")

	deleteQueueOutput := &DeleteQueueOutputBody{}

	json.Unmarshal(resp.Body.Bytes(), deleteQueueOutput)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "DELETED", deleteQueueOutput.Status)
}

func TestCreateQueueProxy(t *testing.T) {
	_, api := humatest.New(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/API/v1/queues", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "application/json", r.Header.Get("Accept"))

		response := CreateQueueOutputBody{
			Status: "CREATED",
			Name:   "user_indexing_queue",
			Type:   "delayed",
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	h := &Handler{
		node: NewTestNode(strings.Replace(server.URL, "http://", "", 1), false),
	}
	h.RegisterRoutes(api)

	type ErrorOutput struct {
		Title  string `json:"title"`
		Status int    `json:"status"`
		Detail string `json:"detail"`
	}

	h.node.CreateQueue("delayed", "my-queue", entity.QueueSettings{})

	resp := api.Post("/API/v1/queues", map[string]any{
		"name":     "user_indexing_queue",
		"type":     "delayed",
		"settings": map[string]any{},
	})

	output := &CreateQueueOutputBody{}

	json.Unmarshal(resp.Body.Bytes(), output)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "CREATED", output.Status)
	assert.Equal(t, "user_indexing_queue", output.Name)
	assert.Equal(t, "delayed", output.Type)
}

func TestGetQueues(t *testing.T) {
	_, api := humatest.New(t)

	h := &Handler{
		node: NewTestNode("", true),
	}
	h.RegisterRoutes(api)

	h.node.CreateQueue("delayed", "test-queue-1", entity.QueueSettings{})

	type ErrorOutput struct {
		Title  string `json:"title"`
		Status int    `json:"status"`
		Detail string `json:"detail"`
	}

	resp := api.Get("/API/v1/queues")

	queuesOutput := &QueuesOutputBody{}

	json.Unmarshal(resp.Body.Bytes(), queuesOutput)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, 1, len(queuesOutput.Queues))
	assert.Equal(t, "test-queue-1", queuesOutput.Queues[0].Name)
	assert.Equal(t, "delayed", queuesOutput.Queues[0].Type)
	assert.Equal(t, 3.1, queuesOutput.Queues[0].EnqueueRPS)
	assert.Equal(t, 2.5, queuesOutput.Queues[0].DequeueRPS)
	assert.Equal(t, 1.2, queuesOutput.Queues[0].AckRPS)
	assert.Equal(t, float64(2.3), queuesOutput.Queues[0].NackRPS)
	assert.Equal(t, int64(0), queuesOutput.Queues[0].Ready)
	assert.Equal(t, int64(0), queuesOutput.Queues[0].Unacked)
	assert.Equal(t, int64(0), queuesOutput.Queues[0].Total)
}

func TestGetQueue(t *testing.T) {
	_, api := humatest.New(t)

	h := &Handler{
		node: NewTestNode("", true),
	}
	h.RegisterRoutes(api)

	h.node.CreateQueue("delayed", "test-queue-1", entity.QueueSettings{})
	h.node.CreateQueue("delayed", "test-queue-2", entity.QueueSettings{})

	type ErrorOutput struct {
		Title  string `json:"title"`
		Status int    `json:"status"`
		Detail string `json:"detail"`
	}

	resp := api.Get("/API/v1/queues/test-queue-1")

	queueOutput := &QueueInfoOutputBody{}

	json.Unmarshal(resp.Body.Bytes(), queueOutput)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "test-queue-1", queueOutput.Name)
	assert.Equal(t, "delayed", queueOutput.Type)
	assert.Equal(t, 3.1, queueOutput.EnqueueRPS)
	assert.Equal(t, 2.5, queueOutput.DequeueRPS)
	assert.Equal(t, 1.2, queueOutput.AckRPS)
	assert.Equal(t, float64(2.3), queueOutput.NackRPS)
	assert.Equal(t, int64(0), queueOutput.Ready)
	assert.Equal(t, int64(0), queueOutput.Unacked)
	assert.Equal(t, int64(0), queueOutput.Total)
}
