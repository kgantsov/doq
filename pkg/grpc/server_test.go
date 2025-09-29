package grpc

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"testing"

	"github.com/kgantsov/doq/pkg/config"
	"github.com/kgantsov/doq/pkg/entity"
	"github.com/kgantsov/doq/pkg/errors"
	"github.com/kgantsov/doq/pkg/metrics"
	pb "github.com/kgantsov/doq/pkg/proto"
	"github.com/kgantsov/doq/pkg/queue"
	"github.com/kgantsov/doq/pkg/queue/memory"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

type testNode struct {
	nextID   uint64
	leader   string
	messages map[uint64]*entity.Message
	acks     map[uint64]*entity.Message
	queues   map[string]*memory.DelayedQueue
}

func newTestNode() *testNode {
	return &testNode{
		messages: make(map[uint64]*entity.Message),
		acks:     make(map[uint64]*entity.Message),
		queues:   make(map[string]*memory.DelayedQueue),
	}
}

func (n *testNode) Join(nodeID, addr string) error {
	return nil
}

func (n *testNode) Backup(w io.Writer, since uint64) (uint64, error) {

	return 0, nil
}

func (n *testNode) Restore(r io.Reader, maxPendingWrites int) error {

	return nil
}

func (n *testNode) IsLeader() bool {
	return true
}

func (n *testNode) GenerateID() uint64 {
	n.nextID++
	return n.nextID
}

func (n *testNode) GetQueues() []*queue.QueueInfo {
	queues := []*queue.QueueInfo{
		&queue.QueueInfo{
			Name: "test-queue", // Mock queue name
			Type: "delayed",    // Assuming all queues are of type "delayed" for this mock
			Settings: entity.QueueSettings{
				MaxUnacked: 8,
				AckTimeout: 600,
			},
			Stats: &metrics.Stats{
				EnqueueRPS: 3.1,
				DequeueRPS: 2.5,
				AckRPS:     1.2,
				NackRPS:    2.3,
			},
			Ready:   512,
			Unacked: 13,
			Total:   1024,
		},
	}

	return queues
}

func (n *testNode) GetQueueInfo(queueName string) (*queue.QueueInfo, error) {
	return &queue.QueueInfo{
		Name: "test-queue", // Mock queue name
		Type: "delayed",    // Assuming all queues are of type "delayed" for this mock
		Settings: entity.QueueSettings{
			MaxUnacked: 8,
			AckTimeout: 600,
		},
		Stats: &metrics.Stats{
			EnqueueRPS: 3.1,
			DequeueRPS: 2.5,
			AckRPS:     1.2,
			NackRPS:    2.3,
		},
		Ready:   512,
		Unacked: 13,
		Total:   1024,
	}, nil

}

func (n *testNode) PrometheusRegistry() prometheus.Registerer {
	return nil
}

func (n *testNode) CreateQueue(queueType, queueName string, settings entity.QueueSettings) error {
	n.queues[queueName] = memory.NewDelayedQueue(true)
	return nil
}

func (n *testNode) DeleteQueue(queueName string) error {
	_, ok := n.queues[queueName]
	if !ok {
		return errors.ErrQueueNotFound
	}

	delete(n.queues, queueName)
	return nil
}

func (n *testNode) Enqueue(
	queueName string, id uint64, group string, priority int64, content string, metadata map[string]string,
) (*entity.Message, error) {
	q, ok := n.queues[queueName]
	if !ok {
		return &entity.Message{}, errors.ErrQueueNotFound
	}

	n.nextID++
	message := &entity.Message{
		ID: n.nextID, Group: group, Priority: priority, Content: content, Metadata: metadata,
	}
	n.messages[message.ID] = message
	q.Enqueue(group, &memory.Item{ID: message.ID, Priority: message.Priority})
	return message, nil
}

func (n *testNode) Dequeue(QueueName string, ack bool) (*entity.Message, error) {
	q, ok := n.queues[QueueName]
	if !ok {
		return &entity.Message{}, errors.ErrQueueNotFound
	}

	if q.Len() == 0 {
		return nil, errors.ErrEmptyQueue
	}

	item := q.Dequeue(false)

	message := n.messages[item.ID]

	if ack {
		delete(n.messages, item.ID)
	} else {
		n.acks[item.ID] = message
	}

	return message, nil
}

func (n *testNode) Ack(QueueName string, id uint64) error {
	_, ok := n.queues[QueueName]
	if !ok {
		return errors.ErrQueueNotFound
	}

	if _, ok := n.acks[id]; !ok {
		return errors.ErrMessageNotFound
	}
	delete(n.acks, id)
	delete(n.messages, id)
	return nil
}

func (n *testNode) Nack(QueueName string, id uint64, priority int64, metadata map[string]string) error {
	q, ok := n.queues[QueueName]
	if !ok {
		return errors.ErrQueueNotFound
	}

	message, ok := n.acks[id]
	if !ok {
		return errors.ErrMessageNotFound
	}

	message.Metadata = metadata
	n.messages[message.ID] = message

	q.Enqueue(message.Group, &memory.Item{ID: message.ID, Priority: message.Priority})

	delete(n.acks, id)
	return nil
}

func (n *testNode) Get(QueueName string, id uint64) (*entity.Message, error) {
	for _, m := range n.messages {
		if m.ID == id {
			return m, nil
		}
	}
	return nil, errors.ErrMessageNotFound
}

func (n *testNode) Delete(QueueName string, id uint64) error {
	q, ok := n.queues[QueueName]
	if !ok {
		return errors.ErrQueueNotFound
	}

	msg, ok := n.messages[id]
	if !ok {
		return errors.ErrMessageNotFound
	}

	q.Delete(msg.Group, id)

	delete(n.acks, id)
	delete(n.messages, id)

	return nil
}

func (n *testNode) UpdatePriority(queueName string, id uint64, priority int64) error {
	q, ok := n.queues[queueName]
	if !ok {
		return errors.ErrQueueNotFound
	}

	message, ok := n.messages[id]
	if !ok {
		return errors.ErrMessageNotFound
	}

	message.Priority = priority
	q.UpdatePriority(message.Group, id, priority)
	return nil
}

const bufSize = 1024 * 1024

var lis *bufconn.Listener

// Initialize a buffer listener to mock the gRPC connection
func init() {
	lis = bufconn.Listen(bufSize)

	cfg := &config.Config{
		Http: config.HttpConfig{Port: "8080"},
		Prometheus: config.PrometheusConfig{
			Enabled: false,
		},
	}

	grpcServer, _ := NewGRPCServer(cfg, newTestNode(), 0)

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

func TestGenerateIDs(t *testing.T) {
	tests := []struct {
		name   string
		number int
	}{
		{"Single ID", 1},
		{"Multiple IDs", 5},
		{"Large Number", 1000},
	}

	ctx := context.Background()
	conn, err := grpc.DialContext(
		ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure(),
	)
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()

	client := pb.NewDOQClient(conn)

	for _, tt := range tests {
		lastId := uint64(0)

		t.Run(tt.name, func(t *testing.T) {
			req := &pb.GenerateIDsRequest{Number: int32(tt.number)}
			resp, err := client.GenerateIDs(ctx, req)
			assert.Nil(t, err)
			assert.Equal(t, tt.number, len(resp.Ids))

			for _, id := range resp.Ids {
				assert.Greater(t, id, lastId)
				lastId = id
			}
		})
	}
}

func TestCreateQueue(t *testing.T) {
	// Create a connection to the test gRPC server
	ctx := context.Background()
	conn, err := grpc.DialContext(
		ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure(),
	)
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()

	// Create a gRPC client
	client := pb.NewDOQClient(conn)

	// Test case 1: Create a queue successfully
	req := &pb.CreateQueueRequest{
		Name: "test-queue",
		Type: "fair",
		Settings: &pb.QueueSettings{
			Strategy:   pb.QueueSettings_WEIGHTED,
			MaxUnacked: 10,
			AckTimeout: 600,
		},
	}

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
	conn, err := grpc.DialContext(
		ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure(),
	)
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()

	client := pb.NewDOQClient(conn)

	// Create a queue first
	reqCreate := &pb.CreateQueueRequest{
		Name: "test-queue",
		Type: "delayed",
		Settings: &pb.QueueSettings{
			AckTimeout: 600,
		},
	}
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
	assert.Equal(
		t,
		"rpc error: code = Unknown desc = failed to delete a queue test-queue",
		err.Error(),
		"Error message should match",
	)
	// assert.False(t, resp.Success, "Queue deletion should fail for a non-existent queue")
}

// Test for Enqueue function
func TestEnqueue(t *testing.T) {
	ctx := context.Background()
	conn, err := grpc.DialContext(
		ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure(),
	)
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()

	client := pb.NewDOQClient(conn)

	// Create a queue first
	reqCreate := &pb.CreateQueueRequest{
		Name:     "test-queue",
		Type:     "delayed",
		Settings: &pb.QueueSettings{AckTimeout: 600},
	}
	_, err = client.CreateQueue(ctx, reqCreate)
	if err != nil {
		t.Fatalf("CreateQueue failed: %v", err)
	}

	// Test case: Enqueue a message successfully
	reqEnqueue := &pb.EnqueueRequest{
		QueueName: "test-queue",
		Content:   "test-message",
		Group:     "default",
		Priority:  10,
	}
	resp, err := client.Enqueue(ctx, reqEnqueue)
	if err != nil {
		t.Fatalf("Enqueue failed: %v", err)
	}
	assert.True(t, resp.Success, "Enqueue should succeed")

	// // Test case: Try to enqueue a message to a non-existent queue (should fail)
	reqEnqueue = &pb.EnqueueRequest{
		QueueName: "non-existent-queue",
		Content:   "test-message",
		Group:     "default",
		Priority:  10,
	}
	resp, err = client.Enqueue(ctx, reqEnqueue)
	assert.Equal(
		t,
		"rpc error: code = Unknown desc = failed to enqueue a message",
		err.Error(),
		"Error message should match",
	)
	// assert.False(t, resp.Success, "Enqueue should fail for a non-existent queue")
}

// Test for Dequeue function
func TestDequeue(t *testing.T) {
	ctx := context.Background()
	conn, err := grpc.DialContext(
		ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure(),
	)
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()

	client := pb.NewDOQClient(conn)

	// Create a queue and enqueue a message first
	reqCreate := &pb.CreateQueueRequest{
		Name: "test-queue",
		Type: "delayed",
		Settings: &pb.QueueSettings{
			AckTimeout: 600,
		},
	}
	_, err = client.CreateQueue(ctx, reqCreate)
	if err != nil {
		t.Fatalf("CreateQueue failed: %v", err)
	}

	reqEnqueue := &pb.EnqueueRequest{
		QueueName: "test-queue",
		Content:   "test-message",
		Group:     "default",
		Priority:  10,
		Metadata:  map[string]string{"retry": "3"},
	}
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
	assert.Equal(
		t,
		"test-message",
		resp.Content,
		"Dequeued message should match the enqueued message",
	)
	assert.Equal(t, "3", resp.Metadata["retry"], "Metadata should match the enqueued message")
}

// Test for Get function
func TestGet(t *testing.T) {
	ctx := context.Background()
	conn, err := grpc.DialContext(
		ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure(),
	)
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()

	client := pb.NewDOQClient(conn)

	// Create a queue and enqueue a message first
	reqCreate := &pb.CreateQueueRequest{
		Name: "test-queue",
		Type: "delayed",
		Settings: &pb.QueueSettings{
			AckTimeout: 600,
		},
	}
	_, err = client.CreateQueue(ctx, reqCreate)
	if err != nil {
		t.Fatalf("CreateQueue failed: %v", err)
	}

	reqEnqueue := &pb.EnqueueRequest{
		QueueName: "test-queue",
		Content:   "test-message",
		Group:     "default",
		Priority:  10,
		Metadata:  map[string]string{"retry": "3"},
	}
	respEnqueue, err := client.Enqueue(ctx, reqEnqueue)
	if err != nil {
		t.Fatalf("Enqueue failed: %v", err)
	}

	// Test case: Get the message successfully
	reqGet := &pb.GetRequest{QueueName: "test-queue", Id: respEnqueue.Id}
	resp, err := client.Get(ctx, reqGet)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	assert.Equal(t, "test-message", resp.Content, "Got message should match the enqueued message")
	assert.Equal(t, "3", resp.Metadata["retry"], "Metadata should match the enqueued message")

	// Test case: Delete the message successfully
	reqDelete := &pb.DeleteRequest{QueueName: "test-queue", Id: respEnqueue.Id}
	respDelete, err := client.Delete(ctx, reqDelete)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	assert.True(t, respDelete.Success, "Delete should succeed")

	// Test case: Get the message successfully
	reqGet = &pb.GetRequest{QueueName: "test-queue", Id: respEnqueue.Id}
	resp, err = client.Get(ctx, reqGet)
	assert.Equal(t, "rpc error: code = Unknown desc = failed to get a message", err.Error())
}

func TestUpdatePriority(t *testing.T) {
	ctx := context.Background()
	conn, err := grpc.DialContext(
		ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure(),
	)
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()

	client := pb.NewDOQClient(conn)

	// Create a queue and enqueue a message first
	reqCreate := &pb.CreateQueueRequest{
		Name: "test-queue",
		Type: "delayed",
		Settings: &pb.QueueSettings{
			AckTimeout: 600,
		}}
	_, err = client.CreateQueue(ctx, reqCreate)
	if err != nil {
		t.Fatalf("CreateQueue failed: %v", err)
	}

	reqEnqueue := &pb.EnqueueRequest{
		QueueName: "test-queue",
		Content:   "test-message-1",
		Group:     "default",
		Priority:  10,
	}
	_, err = client.Enqueue(ctx, reqEnqueue)
	if err != nil {
		t.Fatalf("Enqueue failed: %v", err)
	}

	reqEnqueue = &pb.EnqueueRequest{
		QueueName: "test-queue",
		Content:   "test-message-2",
		Group:     "default",
		Priority:  20,
	}
	resEnqueue, err := client.Enqueue(ctx, reqEnqueue)
	if err != nil {
		t.Fatalf("Enqueue failed: %v", err)
	}

	reqUpdatePriority := &pb.UpdatePriorityRequest{
		QueueName: "test-queue",
		Id:        resEnqueue.Id,
		Priority:  2,
	}
	respUpdatePriority, err := client.UpdatePriority(ctx, reqUpdatePriority)
	assert.Nil(t, err, "UpdatePriority should succeed")
	assert.Equal(t, true, respUpdatePriority.Success, "UpdatePriority should succeed")

	reqDequeue := &pb.DequeueRequest{QueueName: "test-queue", Ack: true}
	resp, err := client.Dequeue(ctx, reqDequeue)
	if err != nil {
		t.Fatalf("Dequeue failed: %v", err)
	}
	assert.Equal(
		t,
		"test-message-2",
		resp.Content,
		"Dequeued message should match the enqueued message",
	)

	reqDequeue = &pb.DequeueRequest{QueueName: "test-queue", Ack: true}
	resp, err = client.Dequeue(ctx, reqDequeue)
	if err != nil {
		t.Fatalf("Dequeue failed: %v", err)
	}
	assert.Equal(
		t, "test-message-1", resp.Content, "Dequeued message should match the enqueued message",
	)
}

// Test for Acknowledge function
func TestAck(t *testing.T) {
	ctx := context.Background()
	conn, err := grpc.DialContext(
		ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure(),
	)
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()

	client := pb.NewDOQClient(conn)

	// Create a queue and enqueue a message first
	reqCreate := &pb.CreateQueueRequest{
		Name: "test-queue",
		Type: "delayed",
		Settings: &pb.QueueSettings{
			AckTimeout: 600,
		},
	}
	_, err = client.CreateQueue(ctx, reqCreate)
	if err != nil {
		t.Fatalf("CreateQueue failed: %v", err)
	}

	reqEnqueue := &pb.EnqueueRequest{
		QueueName: "test-queue",
		Content:   "test-message",
		Group:     "default",
		Priority:  10,
	}
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
	assert.Equal(
		t, "test-message", resp.Content, "Dequeued message should match the enqueued message",
	)

	// Test case: Acknowledge message successfully (implement actual logic in the server if needed)
	req := &pb.AckRequest{QueueName: "test-queue", Id: resp.Id}
	respAck, err := client.Ack(ctx, req)
	if err != nil {
		t.Fatalf("Acknowledge failed: %v", err)
	}
	assert.True(t, respAck.Success, "Acknowledge should succeed")
}

func TestNack(t *testing.T) {
	ctx := context.Background()
	conn, err := grpc.DialContext(
		ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure(),
	)
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()

	client := pb.NewDOQClient(conn)

	// Create a queue and enqueue a message first
	reqCreate := &pb.CreateQueueRequest{
		Name: "test-queue",
		Type: "delayed",
		Settings: &pb.QueueSettings{
			AckTimeout: 600,
		},
	}
	_, err = client.CreateQueue(ctx, reqCreate)
	if err != nil {
		t.Fatalf("CreateQueue failed: %v", err)
	}

	reqEnqueue := &pb.EnqueueRequest{
		QueueName: "test-queue",
		Content:   "test-message",
		Group:     "default",
		Priority:  10,
	}
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
	assert.Equal(
		t, "test-message", resp.Content, "Dequeued message should match the enqueued message",
	)

	// Test case: Acknowledge message successfully (implement actual logic in the server if needed)
	req := &pb.NackRequest{QueueName: "test-queue", Id: resp.Id}
	respAck, err := client.Nack(ctx, req)
	if err != nil {
		t.Fatalf("Nack failed: %v", err)
	}
	assert.True(t, respAck.Success, "Unacknowledge should succeed")

	reqDequeue = &pb.DequeueRequest{QueueName: "test-queue", Ack: true}
	resp, err = client.Dequeue(ctx, reqDequeue)
	if err != nil {
		t.Fatalf("Dequeue failed: %v", err)
	}
	assert.Equal(
		t, "test-message", resp.Content, "Dequeued message should match the enqueued message",
	)
}

func TestEnqueuetDequeueStream(t *testing.T) {
	ctx := context.Background()
	conn, err := grpc.DialContext(
		ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure(),
	)
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()

	client := pb.NewDOQClient(conn)

	// Create a queue first
	req := &pb.CreateQueueRequest{
		Name: "test-queue",
		Type: "delayed",
		Settings: &pb.QueueSettings{
			AckTimeout: 600,
		},
	}
	_, err = client.CreateQueue(ctx, req)
	if err != nil {
		t.Fatalf("CreateQueue failed: %v", err)
	}

	enqueueStream, err := client.EnqueueStream(ctx)
	assert.Nil(t, err, "Failed to open stream")

	enqueueStream.Send(
		&pb.EnqueueRequest{
			QueueName: "test-queue",
			Content:   "test-message-1",
			Group:     "default",
			Priority:  10,
			Metadata:  map[string]string{"retry": "3"},
		},
	)
	_, err = enqueueStream.Recv()
	assert.Nil(t, err)

	enqueueStream.Send(
		&pb.EnqueueRequest{
			QueueName: "test-queue",
			Content:   "test-message-2",
			Group:     "default",
			Priority:  10,
			Metadata:  map[string]string{"retry": "1"},
		},
	)
	_, err = enqueueStream.Recv()
	assert.Nil(t, err)

	enqueueStream.Send(
		&pb.EnqueueRequest{
			QueueName: "test-queue",
			Content:   "test-message-3",
			Group:     "default",
			Priority:  10,
		},
	)
	_, err = enqueueStream.Recv()
	assert.Nil(t, err)

	// Open a dequeueStream to receive messages from the queue
	dequeueStream, err := client.DequeueStream(ctx)
	assert.Nil(t, err, "Failed to open stream")

	err = dequeueStream.Send(&pb.DequeueRequest{
		QueueName: "test-queue",
		Ack:       true,
	})
	assert.Nil(t, err, "Failed to send DequeueRequest")

	// Consume messages from the stream
	for i := 1; i <= 3; i++ {
		msg, err := dequeueStream.Recv()
		if err != nil {
			t.Fatalf("Failed to receive message: %v", err)
		}

		assert.Equal(t, fmt.Sprintf("test-message-%d", i), msg.Content)

		dequeueStream.Send(&pb.DequeueRequest{})
	}
}

func TestGetQueues(t *testing.T) {
	ctx := context.Background()
	conn, err := grpc.DialContext(
		ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure(),
	)
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()

	client := pb.NewDOQClient(conn)

	// Create a queue first
	req := &pb.CreateQueueRequest{
		Name: "test-queue",
		Type: "delayed",
		Settings: &pb.QueueSettings{
			AckTimeout: 600,
		},
	}
	_, err = client.CreateQueue(ctx, req)
	if err != nil {
		t.Fatalf("CreateQueue failed: %v", err)
	}

	resp, err := client.GetQueues(ctx, &pb.GetQueuesRequest{})
	if err != nil {
		t.Fatalf("GetQueues failed: %v", err)
	}

	assert.NotEmpty(t, resp.Queues)
	assert.Equal(t, "test-queue", resp.Queues[0].Name)
	assert.Equal(t, "delayed", resp.Queues[0].Type)
	assert.Equal(t, int64(512), resp.Queues[0].Ready)
	assert.Equal(t, int64(13), resp.Queues[0].Unacked)
	assert.Equal(t, int64(1024), resp.Queues[0].Total)
	assert.Equal(t, float64(3.1), resp.Queues[0].Stats.EnqueueRPS)
	assert.Equal(t, float64(2.5), resp.Queues[0].Stats.DequeueRPS)
	assert.Equal(t, float64(1.2), resp.Queues[0].Stats.AckRPS)
	assert.Equal(t, float64(2.3), resp.Queues[0].Stats.NackRPS)
}

func TestGetQueue(t *testing.T) {
	ctx := context.Background()
	conn, err := grpc.DialContext(
		ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure(),
	)
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()

	client := pb.NewDOQClient(conn)

	// Create a queue first
	req := &pb.CreateQueueRequest{
		Name: "test-queue",
		Type: "delayed",
		Settings: &pb.QueueSettings{
			AckTimeout: 600,
		},
	}
	_, err = client.CreateQueue(ctx, req)
	if err != nil {
		t.Fatalf("CreateQueue failed: %v", err)
	}

	resp, err := client.GetQueue(ctx, &pb.GetQueueRequest{Name: "test-queue"})
	if err != nil {
		t.Fatalf("GetQueue failed: %v", err)
	}

	assert.Equal(t, "test-queue", resp.Name)
	assert.Equal(t, "delayed", resp.Type)
	assert.Equal(t, int64(512), resp.Ready)
	assert.Equal(t, int64(13), resp.Unacked)
	assert.Equal(t, int64(1024), resp.Total)
	assert.Equal(t, float64(3.1), resp.Stats.EnqueueRPS)
	assert.Equal(t, float64(2.5), resp.Stats.DequeueRPS)
	assert.Equal(t, float64(1.2), resp.Stats.AckRPS)
	assert.Equal(t, float64(2.3), resp.Stats.NackRPS)
}
