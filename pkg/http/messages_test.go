package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/bwmarrin/snowflake"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/kgantsov/doq/pkg/entity"
	"github.com/kgantsov/doq/pkg/mocks"
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
	mockNode.On("GenerateID").Return(uint64(1))

	resp := api.Post(
		"/API/v1/ids", map[string]any{
			"number": 1,
		})

	output := &GenerateIdOutputBody{}

	json.Unmarshal(resp.Body.Bytes(), output)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, 1, len(output.IDs))

	lastId := uint64(0)
	for _, idStr := range output.IDs {
		id, err := strconv.ParseUint(idStr, 10, 64)
		assert.NoError(t, err)
		assert.Greater(t, id, lastId)
		lastId = id
	}
}

func TestEnqueue(t *testing.T) {
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

	content := "{\"user_id\": 2, \"name\": \"Jane\"}"
	priority := int64(100)
	metadata := map[string]string{"retry": "3"}

	mockNode.On(
		"Enqueue",
		"my-queue",
		uint64(0),
		"default",
		int64(100),
		content,
		metadata,
	).Return(&entity.Message{
		ID:       uint64(1),
		Priority: priority,
		Group:    "default",
		Content:  content,
		Metadata: metadata,
	}, nil)

	resp := api.Post("/API/v1/queues/my-queue/messages", map[string]any{
		"content":  content,
		"priority": priority,
		"metadata": metadata,
	})

	enqueueOutput := &EnqueueOutputBody{}

	json.Unmarshal(resp.Body.Bytes(), enqueueOutput)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "ENQUEUED", enqueueOutput.Status)
	assert.Equal(t, "1", enqueueOutput.ID)
	assert.Equal(t, priority, enqueueOutput.Priority)
	assert.Equal(t, content, enqueueOutput.Content)
	assert.Equal(t, "3", enqueueOutput.Metadata["retry"])
}

func TestDequeue(t *testing.T) {
	_, api := humatest.New(t)

	mockNode := mocks.NewMockNode()
	h := &Handler{node: mockNode}
	h.RegisterRoutes(api)

	content := "{\"user_id\": 124, \"name\": \"John\"}"
	priority := int64(250)
	metadata := map[string]string{"retry": "1"}

	mockNode.On("Dequeue", "my-queue", false).Return(&entity.Message{
		ID:       uint64(5465),
		Priority: priority,
		Group:    "default",
		Content:  content,
		Metadata: metadata,
	}, nil)

	resp := api.Get("/API/v1/queues/my-queue/messages")

	dequeueOutput := &DequeueOutputBody{}

	json.Unmarshal(resp.Body.Bytes(), dequeueOutput)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "DEQUEUED", dequeueOutput.Status)
	assert.Equal(t, "5465", dequeueOutput.ID)
	assert.Equal(t, priority, dequeueOutput.Priority)
	assert.Equal(t, content, dequeueOutput.Content)
	assert.Equal(t, "1", dequeueOutput.Metadata["retry"])
}

func TestGet(t *testing.T) {
	_, api := humatest.New(t)

	mockNode := mocks.NewMockNode()
	h := &Handler{
		node: mockNode,
	}
	h.RegisterRoutes(api)

	content := "{\"user_id\": 124, \"name\": \"John\"}"
	priority := int64(250)
	metadata := map[string]string{"retry": "1"}

	mockNode.On("Get", "my-queue", uint64(6123)).Return(&entity.Message{
		ID:       uint64(6123),
		Priority: priority,
		Group:    "default",
		Content:  content,
		Metadata: metadata,
	}, nil)

	resp := api.Get(
		fmt.Sprintf("/API/v1/queues/my-queue/messages/%d", 6123),
		map[string]any{},
	)

	output := &GetOutputBody{}

	json.Unmarshal(resp.Body.Bytes(), output)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "GOT", output.Status)
	assert.Equal(t, "6123", output.ID)
	assert.Equal(t, priority, output.Priority)
	assert.Equal(t, "default", output.Group)
	assert.Equal(t, content, output.Content)
	assert.Equal(t, metadata, output.Metadata)
}

func TestDelete(t *testing.T) {
	_, api := humatest.New(t)

	mockNode := mocks.NewMockNode()
	h := &Handler{
		node: mockNode,
	}
	h.RegisterRoutes(api)

	mockNode.On("Delete", "my-queue", uint64(6123)).Return(nil)

	api.Delete(
		fmt.Sprintf("/API/v1/queues/my-queue/messages/%d", 6123),
		map[string]any{},
	)
}

func TestAck(t *testing.T) {
	_, api := humatest.New(t)

	mockNode := mocks.NewMockNode()
	h := &Handler{
		node: mockNode,
	}
	h.RegisterRoutes(api)

	mockNode.On("Ack", "my-queue", uint64(6123)).Return(nil)

	resp := api.Post(
		fmt.Sprintf("/API/v1/queues/my-queue/messages/%d/ack", 6123),
		map[string]any{},
	)

	ackOutput := &AckOutputBody{}

	json.Unmarshal(resp.Body.Bytes(), ackOutput)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "ACKNOWLEDGED", ackOutput.Status)
	assert.Equal(t, "6123", ackOutput.ID)
}

func TestNack(t *testing.T) {
	_, api := humatest.New(t)

	mockNode := mocks.NewMockNode()
	h := &Handler{
		node: mockNode,
	}
	h.RegisterRoutes(api)

	mockNode.On(
		"Nack", "my-transcode-queue", uint64(23421), int64(100), map[string]string{"retry": "5"},
	).Return(nil)

	resp := api.Post(
		fmt.Sprintf("/API/v1/queues/my-transcode-queue/messages/%d/nack", 23421),
		map[string]any{
			"priority": 100,
			"metadata": map[string]string{"retry": "5"},
		},
	)

	nackOutput := &NackOutputBody{}

	json.Unmarshal(resp.Body.Bytes(), nackOutput)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "UNACKNOWLEDGED", nackOutput.Status)
	assert.Equal(t, "23421", nackOutput.ID)
	assert.Equal(t, "5", nackOutput.Metadata["retry"])
}

func TestUpdatePriority(t *testing.T) {
	_, api := humatest.New(t)

	mockNode := mocks.NewMockNode()
	h := &Handler{
		node: mockNode,
	}
	h.RegisterRoutes(api)

	mockNode.On("UpdatePriority", "my-queue-1", uint64(32), int64(256)).Return(nil)

	resp := api.Put(
		fmt.Sprintf("/API/v1/queues/my-queue-1/messages/%d/priority", 32),
		map[string]any{
			"priority": 256,
		},
	)

	updatePriorityOutput := &UpdatePriorityOutputBody{}

	json.Unmarshal(resp.Body.Bytes(), updatePriorityOutput)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "UPDATED", updatePriorityOutput.Status)
	assert.Equal(t, "32", updatePriorityOutput.ID)
	assert.Equal(t, int64(256), updatePriorityOutput.Priority)
}

func TestTouch(t *testing.T) {
	_, api := humatest.New(t)

	mockNode := mocks.NewMockNode()
	h := &Handler{
		node: mockNode,
	}
	h.RegisterRoutes(api)

	mockNode.On("Touch", "my-queue", uint64(6123)).Return(nil)

	resp := api.Post(
		fmt.Sprintf("/API/v1/queues/my-queue/messages/%d/touch", 6123),
		map[string]any{},
	)

	output := &TouchOutputBody{}

	json.Unmarshal(resp.Body.Bytes(), output)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "TOUCHED", output.Status)
	assert.Equal(t, "6123", output.ID)
}
