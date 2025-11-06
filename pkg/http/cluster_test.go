package http

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/kgantsov/doq/pkg/mocks"
	"github.com/stretchr/testify/assert"
)

func TestJoin(t *testing.T) {
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

	mockNode.On("Join", "node-1", "192.168.0.1").Return(nil)

	resp := api.Post("/API/v1/cluster/join", map[string]any{
		"id":   "node-1",
		"addr": "192.168.0.1",
	})

	joinOutput := &JoinOutputBody{}

	json.Unmarshal(resp.Body.Bytes(), joinOutput)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "node-1", joinOutput.ID)
	assert.Equal(t, "192.168.0.1", joinOutput.Addr)
}
