package http

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/kgantsov/doq/pkg/entity"
	"github.com/kgantsov/doq/pkg/metrics"
	"github.com/kgantsov/doq/pkg/mocks"
	"github.com/kgantsov/doq/pkg/queue"
	"github.com/stretchr/testify/assert"
)

func TestCreateQueue(t *testing.T) {
	_, api := humatest.New(t)

	mockNode := mocks.NewMockNode()
	h := &Handler{node: mockNode}

	h.RegisterRoutes(api)

	type ErrorOutput struct {
		Title  string `json:"title"`
		Status int    `json:"status"`
		Detail string `json:"detail"`
	}

	mockNode.On(
		"CreateQueue",
		"delayed",
		"my-queue",
		entity.QueueSettings{},
	).Return(nil)

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

}

func TestDeleteQueue(t *testing.T) {
	_, api := humatest.New(t)

	mockNode := mocks.NewMockNode()
	h := &Handler{node: mockNode}

	h.RegisterRoutes(api)

	type ErrorOutput struct {
		Title  string `json:"title"`
		Status int    `json:"status"`
		Detail string `json:"detail"`
	}

	mockNode.On("DeleteQueue", "my-queue").Return(nil)

	resp := api.Delete("/API/v1/queues/my-queue")

	deleteQueueOutput := &DeleteQueueOutputBody{}

	json.Unmarshal(resp.Body.Bytes(), deleteQueueOutput)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "DELETED", deleteQueueOutput.Status)
}

func TestUpdateQueue(t *testing.T) {
	_, api := humatest.New(t)

	mockNode := mocks.NewMockNode()

	h := &Handler{
		node: mockNode,
	}

	h.RegisterRoutes(api)

	type ErrorOutput struct {
		Title  string `json:"title"`
		Status int    `json:"status"`
		Detail string `json:"detail"`
	}

	mockNode.On("UpdateQueue", "my-queue", entity.QueueSettings{
		Strategy:   "WEIGHTED",
		MaxUnacked: 75,
		AckTimeout: 600,
	}).Return(nil)

	resp := api.Put("/API/v1/queues/my-queue", map[string]any{
		"settings": map[string]any{
			"strategy":    "weighted",
			"max_unacked": 75,
			"ack_timeout": 600,
		},
	})

	updateQueueOutput := &UpdateQueueOutputBody{}

	json.Unmarshal(resp.Body.Bytes(), updateQueueOutput)
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "UPDATED", updateQueueOutput.Status)
	assert.Equal(t, "my-queue", updateQueueOutput.Name)
	assert.Equal(t, "WEIGHTED", updateQueueOutput.Settings.Strategy)
	assert.Equal(t, uint32(600), updateQueueOutput.Settings.AckTimeout)
	assert.Equal(t, 75, updateQueueOutput.Settings.MaxUnacked)
}

func TestGetQueues(t *testing.T) {
	_, api := humatest.New(t)

	mockNode := mocks.NewMockNode()
	h := &Handler{node: mockNode}
	h.RegisterRoutes(api)

	type ErrorOutput struct {
		Title  string `json:"title"`
		Status int    `json:"status"`
		Detail string `json:"detail"`
	}

	mockNode.On("GetQueues").Return([]*queue.QueueInfo{
		{
			Name: "test-queue-1",
			Type: "delayed",
			Settings: entity.QueueSettings{
				Strategy:   "WEIGHTED",
				AckTimeout: 600,
				MaxUnacked: 75,
			},
			Stats: &metrics.Stats{
				EnqueueRPS: 3.1,
				DequeueRPS: 2.5,
				AckRPS:     1.2,
				NackRPS:    2.3,
			},
			Ready:   123,
			Unacked: 456,
			Total:   789,
		},
	})

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
	assert.Equal(t, int64(123), queuesOutput.Queues[0].Ready)
	assert.Equal(t, int64(456), queuesOutput.Queues[0].Unacked)
	assert.Equal(t, int64(789), queuesOutput.Queues[0].Total)
}

func TestGetQueue(t *testing.T) {
	_, api := humatest.New(t)

	mockNode := mocks.NewMockNode()
	h := &Handler{
		node: mockNode,
	}
	h.RegisterRoutes(api)

	type ErrorOutput struct {
		Title  string `json:"title"`
		Status int    `json:"status"`
		Detail string `json:"detail"`
	}

	mockNode.On("GetQueueInfo", "test-queue-1").Return(&queue.QueueInfo{
		Name: "test-queue-1",
		Type: "delayed",
		Settings: entity.QueueSettings{
			Strategy:   "WEIGHTED",
			AckTimeout: 600,
			MaxUnacked: 75,
		},
		Stats: &metrics.Stats{
			EnqueueRPS: 3.1,
			DequeueRPS: 2.5,
			AckRPS:     1.2,
			NackRPS:    2.3,
		},
		Ready:   123,
		Unacked: 456,
		Total:   789,
	}, nil)

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
	assert.Equal(t, int64(123), queueOutput.Ready)
	assert.Equal(t, int64(456), queueOutput.Unacked)
	assert.Equal(t, int64(789), queueOutput.Total)
}

// TestCreateQueueWithDeliveryLimit checks delivery_limit is accepted on create
// and echoed back, so it is a first-class queue setting over HTTP rather than
// being silently dropped between the JSON body and the Raft command.
func TestCreateQueueWithDeliveryLimit(t *testing.T) {
	_, api := humatest.New(t)

	mockNode := mocks.NewMockNode()
	h := &Handler{node: mockNode}
	h.RegisterRoutes(api)

	mockNode.On(
		"CreateQueue",
		"delayed",
		"my-queue",
		entity.QueueSettings{AckTimeout: 60, DeliveryLimit: 3},
	).Return(nil)

	resp := api.Post("/API/v1/queues", map[string]any{
		"name": "my-queue",
		"type": "delayed",
		"settings": map[string]any{
			"ack_timeout":    60,
			"delivery_limit": 3,
		},
	})

	out := &CreateQueueOutputBody{}
	assert.NoError(t, json.Unmarshal(resp.Body.Bytes(), out))

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, uint32(3), out.Settings.DeliveryLimit)
	mockNode.AssertCalled(
		t, "CreateQueue", "delayed", "my-queue",
		entity.QueueSettings{AckTimeout: 60, DeliveryLimit: 3},
	)
}

// TestUpdateQueueWithDeliveryLimit covers the same for the update path, which
// replaces the queue's settings wholesale.
func TestUpdateQueueWithDeliveryLimit(t *testing.T) {
	_, api := humatest.New(t)

	mockNode := mocks.NewMockNode()
	h := &Handler{node: mockNode}
	h.RegisterRoutes(api)

	mockNode.On(
		"UpdateQueue",
		"my-queue",
		entity.QueueSettings{AckTimeout: 60, DeliveryLimit: 5},
	).Return(nil)

	resp := api.Put("/API/v1/queues/my-queue", map[string]any{
		"settings": map[string]any{
			"ack_timeout":    60,
			"delivery_limit": 5,
		},
	})

	out := &UpdateQueueOutputBody{}
	assert.NoError(t, json.Unmarshal(resp.Body.Bytes(), out))

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, uint32(5), out.Settings.DeliveryLimit)
}
