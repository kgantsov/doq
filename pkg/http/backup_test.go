package http

import (
	"io"
	"testing"

	"github.com/danielgtaylor/huma/v2/humatest"
)

func TestBackup(t *testing.T) {
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

	resp := api.Post("/db/backup", map[string]any{
		"since": 0,
	})

	io.Copy(io.Discard, resp.Body)
}
