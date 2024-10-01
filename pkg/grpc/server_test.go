package grpc

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/url"
	"testing"

	pb "github.com/kgantsov/doq/pkg/proto"
	"github.com/kgantsov/doq/pkg/queue"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

type testNode struct {
	nextID   uint64
	leader   string
	messages []*queue.Message
	acks     map[uint64]bool
	queues   map[string]*queue.DelayedPriorityQueue
}

func newTestNode() *testNode {
	return &testNode{
		messages: []*queue.Message{},
		acks:     make(map[uint64]bool),
		queues:   make(map[string]*queue.DelayedPriorityQueue),
	}
}

func (n *testNode) Join(nodeID, addr string) error {
	return nil
}

func (n *testNode) Leader() string {
	u, _ := url.ParseRequestURI(fmt.Sprintf("http://%s", n.leader))

	return fmt.Sprintf("http://%s:8000/API/v1/queues", u.Hostname())
}
func (n *testNode) IsLeader() bool {
	return true
}

func (n *testNode) GenerateID() uint64 {
	n.nextID++
	return n.nextID
}

func (n *testNode) PrometheusRegistry() prometheus.Registerer {
	return nil
}

func (n *testNode) CreateQueue(queueType, queueName string) error {
	n.queues[queueName] = queue.NewDelayedPriorityQueue(true)
	return nil
}
func (n *testNode) DeleteQueue(queueName string) error {
	_, ok := n.queues[queueName]
	if !ok {
		return queue.ErrQueueNotFound
	}

	delete(n.queues, queueName)
	return nil
}

func (n *testNode) Enqueue(
	queueName string, group string, priority int64, content string,
) (*queue.Message, error) {
	_, ok := n.queues[queueName]
	if !ok {
		return &queue.Message{}, queue.ErrQueueNotFound
	}

	n.nextID++
	message := &queue.Message{ID: n.nextID, Group: group, Priority: priority, Content: content}
	n.messages = append(n.messages, message)
	return message, nil
}
func (n *testNode) Dequeue(QueueName string, ack bool) (*queue.Message, error) {
	if len(n.messages) == 0 {
		return nil, queue.ErrEmptyQueue
	}
	message := n.messages[0]
	n.messages = n.messages[1:]

	n.acks[message.ID] = false

	return message, nil
}
func (n *testNode) Ack(QueueName string, id uint64) error {
	if _, ok := n.acks[id]; !ok {
		return queue.ErrMessageNotFound
	}
	delete(n.acks, id)
	return nil
}

func (n *testNode) GetByID(id uint64) (*queue.Message, error) {
	for _, m := range n.messages {
		if m.ID == id {
			return m, nil
		}
	}
	return nil, queue.ErrMessageNotFound
}

func (n *testNode) UpdatePriority(queueName string, id uint64, priority int64) error {
	for _, m := range n.messages {
		if m.ID == id {
			m.Priority = priority
			return nil
		}
	}
	return queue.ErrMessageNotFound
}

const bufSize = 1024 * 1024

var lis *bufconn.Listener

// Initialize a buffer listener to mock the gRPC connection
func init() {
	lis = bufconn.Listen(bufSize)
	// grpcServer := grpc.NewServer()
	// pb.RegisterDOQServer(grpcServer, NewQueueServer(newTestNode()))

	grpcServer, _ := NewGRPCServer(newTestNode())

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Server exited with error: %v", err)
		}
	}()
}

// Create a gRPC client connection to the mock server
func bufDialer(context.Context, string) (net.Conn, error) {
	return lis.Dial()
}

func TestCreateQueue(t *testing.T) {
	// Create a connection to the test gRPC server
	ctx := context.Background()
	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()

	// Create a gRPC client
	client := pb.NewDOQClient(conn)

	// Test case 1: Create a queue successfully
	req := &pb.CreateQueueRequest{Name: "test-queue"}
	resp, err := client.CreateQueue(ctx, req)
	if err != nil {
		t.Fatalf("CreateQueue failed: %v", err)
	}

	// Use testify's assert to check the response
	assert.True(t, resp.Success, "Queue creation should succeed")
}

// Test for DeleteQueue function
func TestDeleteQueue(t *testing.T) {
	ctx := context.Background()
	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()

	client := pb.NewDOQClient(conn)

	// Create a queue first
	reqCreate := &pb.CreateQueueRequest{Name: "test-queue"}
	_, err = client.CreateQueue(ctx, reqCreate)
	if err != nil {
		t.Fatalf("CreateQueue failed: %v", err)
	}

	// Test case 1: Delete the queue successfully
	reqDelete := &pb.DeleteQueueRequest{Name: "test-queue"}
	resp, err := client.DeleteQueue(ctx, reqDelete)
	if err != nil {
		t.Fatalf("DeleteQueue failed: %v", err)
	}
	assert.True(t, resp.Success, "Queue deletion should succeed")

	// // Test case 2: Try to delete a non-existent queue (should fail)
	resp, err = client.DeleteQueue(ctx, reqDelete)
	assert.Equal(t, "rpc error: code = Unknown desc = failed to delete a queue test-queue", err.Error(), "Error message should match")
	// assert.False(t, resp.Success, "Queue deletion should fail for a non-existent queue")
}

// Test for Enqueue function
func TestEnqueue(t *testing.T) {
	ctx := context.Background()
	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()

	client := pb.NewDOQClient(conn)

	// Create a queue first
	reqCreate := &pb.CreateQueueRequest{Name: "test-queue"}
	_, err = client.CreateQueue(ctx, reqCreate)
	if err != nil {
		t.Fatalf("CreateQueue failed: %v", err)
	}

	// Test case: Enqueue a message successfully
	reqEnqueue := &pb.EnqueueRequest{QueueName: "test-queue", Content: "test-message", Group: "default", Priority: 10}
	resp, err := client.Enqueue(ctx, reqEnqueue)
	if err != nil {
		t.Fatalf("Enqueue failed: %v", err)
	}
	assert.True(t, resp.Success, "Enqueue should succeed")

	// // Test case: Try to enqueue a message to a non-existent queue (should fail)
	reqEnqueue = &pb.EnqueueRequest{QueueName: "non-existent-queue", Content: "test-message", Group: "default", Priority: 10}
	resp, err = client.Enqueue(ctx, reqEnqueue)
	assert.Equal(t, "rpc error: code = Unknown desc = failed to enqueue a message", err.Error(), "Error message should match")
	// assert.False(t, resp.Success, "Enqueue should fail for a non-existent queue")
}

// Test for Dequeue function
func TestDequeue(t *testing.T) {
	ctx := context.Background()
	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()

	client := pb.NewDOQClient(conn)

	// Create a queue and enqueue a message first
	reqCreate := &pb.CreateQueueRequest{Name: "test-queue"}
	_, err = client.CreateQueue(ctx, reqCreate)
	if err != nil {
		t.Fatalf("CreateQueue failed: %v", err)
	}

	reqEnqueue := &pb.EnqueueRequest{QueueName: "test-queue", Content: "test-message", Group: "default", Priority: 10}
	_, err = client.Enqueue(ctx, reqEnqueue)
	if err != nil {
		t.Fatalf("Enqueue failed: %v", err)
	}

	// Test case: Dequeue the message successfully
	reqDequeue := &pb.DequeueRequest{QueueName: "test-queue"}
	resp, err := client.Dequeue(ctx, reqDequeue)
	if err != nil {
		t.Fatalf("Dequeue failed: %v", err)
	}
	assert.Equal(t, "test-message", resp.Content, "Dequeued message should match the enqueued message")
}

// Test for Acknowledge function
func TestAcknowledge(t *testing.T) {
	ctx := context.Background()
	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()

	client := pb.NewDOQClient(conn)

	// Create a queue and enqueue a message first
	reqCreate := &pb.CreateQueueRequest{Name: "test-queue"}
	_, err = client.CreateQueue(ctx, reqCreate)
	if err != nil {
		t.Fatalf("CreateQueue failed: %v", err)
	}

	reqEnqueue := &pb.EnqueueRequest{QueueName: "test-queue", Content: "test-message", Group: "default", Priority: 10}
	_, err = client.Enqueue(ctx, reqEnqueue)
	if err != nil {
		t.Fatalf("Enqueue failed: %v", err)
	}

	// Test case: Dequeue the message successfully
	reqDequeue := &pb.DequeueRequest{QueueName: "test-queue", Ack: false}
	resp, err := client.Dequeue(ctx, reqDequeue)
	if err != nil {
		t.Fatalf("Dequeue failed: %v", err)
	}
	assert.Equal(t, "test-message", resp.Content, "Dequeued message should match the enqueued message")

	// Test case: Acknowledge message successfully (implement actual logic in the server if needed)
	req := &pb.AcknowledgeRequest{QueueName: "test-queue", Id: resp.Id}
	respAck, err := client.Acknowledge(ctx, req)
	if err != nil {
		t.Fatalf("Acknowledge failed: %v", err)
	}
	assert.True(t, respAck.Success, "Acknowledge should succeed")
}
