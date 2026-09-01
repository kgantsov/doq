package cluster

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestJoiner tests the Joiner.
func TestJoiner(t *testing.T) {
	// Start a local HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		// Test request parameters
		assert.Equal(t, "POST", req.Method)
		assert.Equal(t, "/API/v1/cluster/join", req.URL.String())
		// Send response to be tested
		rw.Write([]byte(`OK`))
	}))
	// Close the server when test finishes
	defer server.Close()

	// get host name and port from server.URL
	host := server.URL[len("http://"):]

	hosts := []string{host}
	j := NewJoiner("node0", "raftAddr", hosts, nil)

	assert.NotNil(t, j)

	err := j.Join()
	require.NoError(t, err)
}

func TestJoinerRetry(t *testing.T) {
	attemptHost1 := 0
	attemptHost2 := 0

	// Start a local HTTP server
	server1 := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		assert.Equal(t, "POST", req.Method)
		assert.Equal(t, "/API/v1/cluster/join", req.URL.String())

		if attemptHost1 < 2 {
			attemptHost1++
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}
		assert.Equal(t, 2, attemptHost1)
		rw.Write([]byte(`OK`))
	}))
	// Close the server when test finishes
	defer server1.Close()

	// Start a local HTTP server
	server2 := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		assert.Equal(t, "POST", req.Method)
		assert.Equal(t, "/API/v1/cluster/join", req.URL.String())

		if attemptHost2 < 2 {
			attemptHost2++
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}
		assert.Equal(t, 2, attemptHost2)
		rw.Write([]byte(`OK`))
	}))
	// Close the server when test finishes
	defer server2.Close()

	// get host name and port from server.URL
	host1 := server1.URL[len("http://"):]
	host2 := server2.URL[len("http://"):]

	hosts := []string{host1, host2}
	j := NewJoiner("node0", "raftAddr", hosts, nil)
	j.retryInterval = time.Millisecond

	assert.NotNil(t, j)

	err := j.Join()
	require.NoError(t, err)
}

// TestJoinerNoHosts covers the non-Kubernetes case: no peers and no resolver
// means there is nothing to join, so Join is a no-op.
func TestJoinerNoHosts(t *testing.T) {
	hosts := []string{}
	j := NewJoiner("node0", "raftAddr", hosts, nil)

	assert.NotNil(t, j)

	err := j.Join()

	assert.NoError(t, err)
}

func TestJoinerHostsUnavailable(t *testing.T) {
	hosts := []string{"host1", "host2"}
	j := NewJoiner("node0", "raftAddr", hosts, nil)
	j.maxAttempts = 3
	j.retryInterval = time.Millisecond

	assert.NotNil(t, j)

	err := j.Join()

	assert.Contains(t, err.Error(), "failed to join cluster")
}

// TestJoinerResolvesLatePeer reproduces the Parallel-start bug: the node starts
// with no peers in DNS (empty snapshot) and must keep re-resolving until the
// seed appears, then join it — instead of giving up immediately as it used to.
func TestJoinerResolvesLatePeer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Write([]byte(`OK`))
	}))
	defer server.Close()

	host := server.URL[len("http://"):]

	// Simulate SRV records converging: the first two lookups return nothing,
	// then the peer appears.
	calls := 0
	resolve := func() ([]string, error) {
		calls++
		if calls < 3 {
			return nil, nil
		}
		return []string{host}, nil
	}

	j := NewJoiner("node0", "raftAddr", nil, resolve)
	j.retryInterval = time.Millisecond

	err := j.Join()
	require.NoError(t, err)
	assert.GreaterOrEqual(t, calls, 3)
}
