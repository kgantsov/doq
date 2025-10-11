package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/bwmarrin/snowflake"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/kgantsov/doq/pkg/entity"
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
		node: NewTestNode(strings.Replace(server.URL, "http://", "", 1), false),
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
		node: NewTestNode("", true),
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
		node: NewTestNode("", true),
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
		node: NewTestNode("", true),
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
		node: NewTestNode("", true),
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
