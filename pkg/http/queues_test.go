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
