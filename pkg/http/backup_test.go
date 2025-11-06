package http

import (
	"io"
	"testing"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/kgantsov/doq/pkg/mocks"
	"github.com/stretchr/testify/mock"
)

func TestBackup(t *testing.T) {
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

	mockNode.On("IsLeader").Return(true)
	mockNode.On("Backup", mock.Anything, uint64(0)).Return(uint64(141), nil)

	resp := api.Post("/db/backup", map[string]any{
		"since": 0,
	})

	io.Copy(io.Discard, resp.Body)
}
