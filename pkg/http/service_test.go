package http

import (
	"bytes"
	"embed"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	addr := "8080"

	node := newTestNode()
	service := NewHttpService(addr, node, embed.FS{}, embed.FS{})
	node.CreateQueue("delayed", "my-queue")

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
