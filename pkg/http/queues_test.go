package http

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/stretchr/testify/assert"
)

func TestCreateDeleteQueue(t *testing.T) {
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

	resp := api.Post("/API/v1/queues", map[string]any{
		"name": "my-queue",
		"type": "delayed",
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

func TestGetQueues(t *testing.T) {
	_, api := humatest.New(t)

	h := &Handler{
		node: newTestNode(),
	}
	h.RegisterRoutes(api)

	h.node.CreateQueue("delayed", "test-queue-1")
	// h.node.CreateQueue("delayed", "test-queue-2")
	// h.node.CreateQueue("delayed", "test-queue-66")

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
	assert.Equal(t, 1.6, queuesOutput.Queues[0].EnqueueRPS)
	assert.Equal(t, 1.1, queuesOutput.Queues[0].DequeueRPS)
	assert.Equal(t, 1.1, queuesOutput.Queues[0].AckRPS)
	assert.Equal(t, float64(0), queuesOutput.Queues[0].NackRPS)
	assert.Equal(t, int64(0), queuesOutput.Queues[0].Ready)
	assert.Equal(t, int64(0), queuesOutput.Queues[0].Unacked)
	assert.Equal(t, int64(0), queuesOutput.Queues[0].Total)
}

func TestGetQueue(t *testing.T) {
	_, api := humatest.New(t)

	h := &Handler{
		node: newTestNode(),
	}
	h.RegisterRoutes(api)

	h.node.CreateQueue("delayed", "test-queue-1")
	h.node.CreateQueue("delayed", "test-queue-2")

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
	assert.Equal(t, 1.6, queueOutput.EnqueueRPS)
	assert.Equal(t, 1.1, queueOutput.DequeueRPS)
	assert.Equal(t, 1.1, queueOutput.AckRPS)
	assert.Equal(t, float64(0), queueOutput.NackRPS)
	assert.Equal(t, int64(0), queueOutput.Ready)
	assert.Equal(t, int64(0), queueOutput.Unacked)
	assert.Equal(t, int64(0), queueOutput.Total)
}
