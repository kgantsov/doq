package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateQueue_Success(t *testing.T) {
	// Create a mock server that returns a successful response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/API/v1/queues", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "application/json", r.Header.Get("Accept"))

		response := CreateQueueOutputBody{
			Status: "CREATED",
			Name:   "user_indexing_queue",
			Type:   "delayed",
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Initialize the proxy with the test HTTP client
	proxy := NewProxy(server.Client(), "")

	// Define input for the CreateQueue method
	input := &CreateQueueInputBody{
		Name: "user_indexing_queue",
		Type: "delayed",
	}

	// Call the CreateQueue method
	output, err := proxy.CreateQueue(context.Background(), strings.Replace(server.URL, "http://", "", 1), input)

	// Assert that no error occurred
	require.NoError(t, err)

	// Assert that the response matches the expected output
	expectedOutput := &CreateQueueOutputBody{
		Status: "CREATED",
		Name:   "user_indexing_queue",
		Type:   "delayed",
	}
	assert.Equal(t, expectedOutput, output)
}

func TestCreateQueue_HostParsingError(t *testing.T) {
	proxy := NewProxy(&http.Client{}, "")

	input := &CreateQueueInputBody{
		Name: "user_indexing_queue",
		Type: "delayed",
	}

	// Call the CreateQueue method with an invalid host
	_, err := proxy.CreateQueue(context.Background(), "invalid-host", input)

	// Assert that an error occurred
	require.Error(t, err)
	assert.Equal(t, http.StatusServiceUnavailable, err.GetStatus())
	assert.Contains(t, "Failed to proxy create queue request", err.Error())
}

func TestCreateQueue_HTTPClientError(t *testing.T) {
	// Create a mock server that simulates an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	proxy := NewProxy(server.Client(), "")

	input := &CreateQueueInputBody{
		Name: "user_indexing_queue",
		Type: "delayed",
	}

	// Call the CreateQueue method
	_, err := proxy.CreateQueue(context.Background(), strings.Replace(server.URL, "http://", "", 1), input)

	require.Error(t, err)

	assert.Equal(t, http.StatusBadRequest, err.GetStatus())
	assert.Contains(t, "Failed to create a queue", err.Error())
}

func TestDeleteQueue_Success(t *testing.T) {
	// Create a mock server that returns a successful response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/API/v1/queues/user_indexing_queue", r.URL.Path)
		assert.Equal(t, "DELETE", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "application/json", r.Header.Get("Accept"))

		response := DeleteQueueOutputBody{
			Status: "DELETED",
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Initialize the proxy with the test HTTP client
	proxy := NewProxy(server.Client(), "")

	// Call the CreateQueue method
	output, err := proxy.DeleteQueue(context.Background(), strings.Replace(server.URL, "http://", "", 1), "user_indexing_queue")

	// Assert that no error occurred
	require.NoError(t, err)

	// Assert that the response matches the expected output
	expectedOutput := &DeleteQueueOutputBody{
		Status: "DELETED",
	}
	assert.Equal(t, expectedOutput, output)
}

func TestEnqueue_Success(t *testing.T) {
	// Create a mock server that returns a successful response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/API/v1/queues/indexing-queue/messages", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "application/json", r.Header.Get("Accept"))

		response := EnqueueOutputBody{
			Status:   "ENQUEUED",
			ID:       "123",
			Group:    "customer-1",
			Priority: 61,
			Content:  "{\"user_id\": 114}",
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Initialize the proxy with the test HTTP client
	proxy := NewProxy(server.Client(), "")

	// Define input for the CreateQueue method
	input := &EnqueueInputBody{
		Group:    "customer-1",
		Priority: 61,
		Content:  "{\"user_id\": 114}",
	}

	// Call the CreateQueue method
	output, err := proxy.Enqueue(context.Background(), strings.Replace(server.URL, "http://", "", 1), "indexing-queue", input)

	// Assert that no error occurred
	require.NoError(t, err)

	// Assert that the response matches the expected output
	expectedOutput := &EnqueueOutputBody{
		Status:   "ENQUEUED",
		ID:       "123",
		Group:    "customer-1",
		Priority: 61,
		Content:  "{\"user_id\": 114}",
	}
	assert.Equal(t, expectedOutput, output)
}

func TestDequeue_Success(t *testing.T) {
	// Create a mock server that returns a successful response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/API/v1/queues/indexing-queue/messages", r.URL.Path)
		assert.Equal(t, "true", r.URL.Query().Get("ack"))
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "application/json", r.Header.Get("Accept"))

		response := DequeueOutputBody{
			Status:   "DEQUEUED",
			ID:       "75",
			Group:    "customer-1",
			Priority: 31,
			Content:  "{\"id\": 114, \"name\": \"test\"}",
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Initialize the proxy with the test HTTP client
	proxy := NewProxy(server.Client(), "")

	// Call the CreateQueue method
	output, err := proxy.Dequeue(context.Background(), strings.Replace(server.URL, "http://", "", 1), "indexing-queue", true)

	// Assert that no error occurred
	require.NoError(t, err)

	// Assert that the response matches the expected output
	expectedOutput := &DequeueOutputBody{
		Status:   "DEQUEUED",
		ID:       "75",
		Group:    "customer-1",
		Priority: 31,
		Content:  "{\"id\": 114, \"name\": \"test\"}",
	}
	assert.Equal(t, expectedOutput, output)
}

func TestGet_Success(t *testing.T) {
	// Create a mock server that returns a successful response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/API/v1/queues/indexing-queue/messages/75", r.URL.Path)
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "application/json", r.Header.Get("Accept"))

		response := GetOutputBody{
			Status:   "GOT",
			ID:       "75",
			Group:    "customer-1",
			Priority: 31,
			Content:  "{\"id\": 114, \"name\": \"test\"}",
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Initialize the proxy with the test HTTP client
	proxy := NewProxy(server.Client(), "")

	output, err := proxy.Get(
		context.Background(), strings.Replace(server.URL, "http://", "", 1), "indexing-queue", 75,
	)

	require.NoError(t, err)

	// Assert that the response matches the expected output
	expectedOutput := &GetOutputBody{
		Status:   "GOT",
		ID:       "75",
		Group:    "customer-1",
		Priority: 31,
		Content:  "{\"id\": 114, \"name\": \"test\"}",
	}
	assert.Equal(t, expectedOutput, output)
}

func TestDelete_Success(t *testing.T) {
	// Create a mock server that returns a successful response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/API/v1/queues/indexing-queue/messages/75", r.URL.Path)
		assert.Equal(t, "DELETE", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "application/json", r.Header.Get("Accept"))

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	// Initialize the proxy with the test HTTP client
	proxy := NewProxy(server.Client(), "")

	err := proxy.Delete(
		context.Background(), strings.Replace(server.URL, "http://", "", 1), "indexing-queue", 75,
	)

	require.NoError(t, err)
}

func TestAck_Success(t *testing.T) {
	// Create a mock server that returns a successful response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/API/v1/queues/indexing-queue/messages/1122/ack", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "application/json", r.Header.Get("Accept"))

		response := AckOutputBody{
			Status: "DEQUEUED",
			ID:     "1122",
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Initialize the proxy with the test HTTP client
	proxy := NewProxy(server.Client(), "")

	// Call the CreateQueue method
	output, err := proxy.Ack(context.Background(), strings.Replace(server.URL, "http://", "", 1), "indexing-queue", 1122)

	// Assert that no error occurred
	require.NoError(t, err)

	// Assert that the response matches the expected output
	expectedOutput := &AckOutputBody{
		Status: "DEQUEUED",
		ID:     "1122",
	}
	assert.Equal(t, expectedOutput, output)
}

func TestNack_Success(t *testing.T) {
	// Create a mock server that returns a successful response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/API/v1/queues/indexing-queue/messages/1122/nack", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "application/json", r.Header.Get("Accept"))

		response := NackOutputBody{
			Status: "DEQUEUED",
			ID:     "1122",
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Initialize the proxy with the test HTTP client
	proxy := NewProxy(server.Client(), "")

	// Define input for the CreateQueue method
	input := &NackInputBody{
		Metadata: map[string]string{},
	}

	// Call the CreateQueue method
	output, err := proxy.Nack(
		context.Background(),
		strings.Replace(server.URL, "http://", "", 1),
		"indexing-queue",
		1122,
		input,
	)

	// Assert that no error occurred
	require.NoError(t, err)

	// Assert that the response matches the expected output
	expectedOutput := &NackOutputBody{
		Status: "DEQUEUED",
		ID:     "1122",
	}
	assert.Equal(t, expectedOutput, output)
}

func TestUpdatePriority_Success(t *testing.T) {
	// Create a mock server that returns a successful response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/API/v1/queues/indexing-queue/messages/980/priority", r.URL.Path)
		assert.Equal(t, "PUT", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "application/json", r.Header.Get("Accept"))

		response := UpdatePriorityOutputBody{
			Status:   "ENQUEUED",
			ID:       "5634",
			Priority: 777,
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Initialize the proxy with the test HTTP client
	proxy := NewProxy(server.Client(), "")

	// Define input for the CreateQueue method
	input := &UpdatePriorityInputBody{
		Priority: 777,
	}

	// Call the CreateQueue method
	output, err := proxy.UpdatePriority(
		context.Background(), strings.Replace(server.URL, "http://", "", 1), "indexing-queue", 980, input,
	)

	// Assert that no error occurred
	require.NoError(t, err)

	// Assert that the response matches the expected output
	expectedOutput := &UpdatePriorityOutputBody{
		Status:   "ENQUEUED",
		ID:       "5634",
		Priority: 777,
	}
	assert.Equal(t, expectedOutput, output)
}
