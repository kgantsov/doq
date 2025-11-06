package grpc

import (
	"context"
	"testing"

	"github.com/kgantsov/doq/pkg/entity"
	"github.com/kgantsov/doq/pkg/mocks"
	pb "github.com/kgantsov/doq/pkg/proto"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

func TestProxyCreateQueue(t *testing.T) {
	mockNode := mocks.NewMockNode()
	startTestServer(mockNode)

	// Create a connection to the test gRPC server
	ctx := context.Background()
	conn, err := grpc.DialContext(
		ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure(),
	)
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()

	client := pb.NewDOQClient(conn)

	proxy := NewGRPCProxy()
	proxy.leader = "bufnet"
	proxy.client = client

	mockNode.On(
		"CreateQueue",
		"fair",
		"test-queue",
		entity.QueueSettings{
			Strategy:   "WEIGHTED",
			MaxUnacked: 10,
			AckTimeout: 600,
		},
	).Return(nil)

	// Send a request to the mock server.
	respCreateQueue, err := proxy.CreateQueue(context.Background(), "bufnet", &pb.CreateQueueRequest{
		Name: "test-queue",
		Type: "fair",
		Settings: &pb.QueueSettings{
			Strategy:   pb.QueueSettings_WEIGHTED,
			MaxUnacked: 10,
			AckTimeout: 600,
		},
	})
	assert.Nil(t, err)
	assert.Equal(t, true, respCreateQueue.Success)
}

func TestProxyDeleteQueue(t *testing.T) {
	mockNode := mocks.NewMockNode()
	startTestServer(mockNode)

	// Create a connection to the test gRPC server
	ctx := context.Background()
	conn, err := grpc.DialContext(
		ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure(),
	)
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()

	client := pb.NewDOQClient(conn)

	proxy := NewGRPCProxy()
	proxy.leader = "bufnet"
	proxy.client = client

	mockNode.On(
		"DeleteQueue",
		"test-queue",
	).Return(nil)

	respDeleteQueue, err := proxy.DeleteQueue(context.Background(), "bufnet", &pb.DeleteQueueRequest{
		Name: "test-queue",
	})
	assert.Nil(t, err)
	assert.Equal(t, true, respDeleteQueue.Success)
}

func TestProxyUpdateQueue(t *testing.T) {
	mockNode := mocks.NewMockNode()
	startTestServer(mockNode)

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

	// Create a new proxy.
	proxy := NewGRPCProxy()
	proxy.leader = "bufnet"
	proxy.client = client

	mockNode.On(
		"UpdateQueue",
		"test-queue",
		entity.QueueSettings{
			Strategy:   "WEIGHTED",
			MaxUnacked: 20,
			AckTimeout: 1200,
		},
	).Return(nil)

	// Update the queue settings.
	respUpdateQueue, err := proxy.UpdateQueue(context.Background(), "bufnet", &pb.UpdateQueueRequest{
		Name: "test-queue",
		Settings: &pb.QueueSettings{
			Strategy:   pb.QueueSettings_WEIGHTED,
			MaxUnacked: 20,
			AckTimeout: 1200,
		},
	})

	assert.Nil(t, err)
	assert.Equal(t, true, respUpdateQueue.Success)
}

func TestProxyEnqueue(t *testing.T) {
	mockNode := mocks.NewMockNode()
	startTestServer(mockNode)

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

	// Create a new proxy.
	proxy := NewGRPCProxy()
	proxy.leader = "bufnet"
	proxy.client = client

	mockNode.On(
		"Enqueue",
		"test-queue",
		uint64(0),
		"default",
		int64(10),
		"message 1",
		map[string]string(nil),
	).Return(&entity.Message{
		ID:       uint64(1),
		Priority: 10,
		Group:    "default",
		Content:  "message 1",
		Metadata: nil,
	}, nil)

	respEnqueue, err := proxy.Enqueue(context.Background(), "bufnet", &pb.EnqueueRequest{
		QueueName: "test-queue",
		Group:     "default",
		Priority:  10,
		Content:   "message 1",
	})
	assert.Nil(t, err)
	assert.NotNil(t, respEnqueue.Id)
	assert.Equal(t, "default", respEnqueue.Group)
	assert.Equal(t, int64(10), respEnqueue.Priority)
	assert.Equal(t, "message 1", respEnqueue.Content)
	assert.Equal(t, true, respEnqueue.Success)
}

func TestProxyDequeue(t *testing.T) {
	mockNode := mocks.NewMockNode()
	startTestServer(mockNode)

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

	// Create a new proxy.
	proxy := NewGRPCProxy()
	proxy.leader = "bufnet"
	proxy.client = client

	mockNode.On(
		"Dequeue",
		"test-queue",
		true,
	).Return(&entity.Message{
		ID:       uint64(1123),
		Priority: 10,
		Group:    "default",
		Content:  "message 1",
		Metadata: nil,
	}, nil)

	respDequeue, err := proxy.Dequeue(context.Background(), "bufnet", &pb.DequeueRequest{
		QueueName: "test-queue",
		Ack:       true,
	})
	assert.Nil(t, err)
	assert.Equal(t, uint64(1123), respDequeue.Id)
	assert.Equal(t, "default", respDequeue.Group)
	assert.Equal(t, int64(10), respDequeue.Priority)
	assert.Equal(t, "message 1", respDequeue.Content)
	assert.Equal(t, true, respDequeue.Success)
}

func TestProxyGet(t *testing.T) {
	mockNode := mocks.NewMockNode()
	startTestServer(mockNode)

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

	// Create a new proxy.
	proxy := NewGRPCProxy()
	proxy.leader = "bufnet"
	proxy.client = client

	mockNode.On("Get", "test-queue", uint64(6123)).Return(&entity.Message{
		ID:       uint64(6123),
		Priority: 50,
		Group:    "default",
		Content:  "message 1",
		Metadata: map[string]string{"key": "value"},
	}, nil)

	respGet, err := proxy.Get(context.Background(), "bufnet", &pb.GetRequest{
		QueueName: "test-queue",
		Id:        uint64(6123),
	})
	assert.Nil(t, err)
	assert.Equal(t, uint64(6123), respGet.Id)
	assert.Equal(t, "default", respGet.Group)
	assert.Equal(t, int64(50), respGet.Priority)
	assert.Equal(t, "message 1", respGet.Content)
}

func TestProxyDelete(t *testing.T) {
	mockNode := mocks.NewMockNode()
	startTestServer(mockNode)

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

	// Create a new proxy.
	proxy := NewGRPCProxy()
	proxy.leader = "bufnet"
	proxy.client = client

	mockNode.On("Delete", "test-queue-1231", uint64(5364)).Return(nil)

	respDelete, err := proxy.Delete(context.Background(), "bufnet", &pb.DeleteRequest{
		QueueName: "test-queue-1231",
		Id:        5364,
	})
	assert.Nil(t, err)
	assert.Equal(t, true, respDelete.Success)

}

func TestProxyAck(t *testing.T) {
	mockNode := mocks.NewMockNode()
	startTestServer(mockNode)

	ctx := context.Background()
	conn, err := grpc.DialContext(
		ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure(),
	)
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()

	client := pb.NewDOQClient(conn)

	proxy := NewGRPCProxy()
	proxy.leader = "bufnet"
	proxy.client = client

	mockNode.On("Ack", "my-transcode-queue-1", uint64(23421)).Return(nil)

	respAck, err := proxy.Ack(context.Background(), "bufnet", &pb.AckRequest{
		QueueName: "my-transcode-queue-1",
		Id:        uint64(23421),
	})

	assert.Nil(t, err)
	assert.Equal(t, true, respAck.Success)
}

func TestProxyNack(t *testing.T) {
	mockNode := mocks.NewMockNode()
	startTestServer(mockNode)

	ctx := context.Background()
	conn, err := grpc.DialContext(
		ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure(),
	)
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()

	client := pb.NewDOQClient(conn)

	proxy := NewGRPCProxy()
	proxy.leader = "bufnet"
	proxy.client = client

	mockNode.On(
		"Nack", "my-transcode-queue", uint64(23421), int64(100), map[string]string{"retry": "5"},
	).Return(nil)

	respNack, err := proxy.Nack(context.Background(), "bufnet", &pb.NackRequest{
		QueueName: "my-transcode-queue",
		Id:        uint64(23421),
		Priority:  int64(100),
		Metadata:  map[string]string{"retry": "5"},
	})

	assert.Nil(t, err)
	assert.Equal(t, true, respNack.Success)
}

func TestProxyUpdatePriority(t *testing.T) {
	mockNode := mocks.NewMockNode()
	startTestServer(mockNode)

	ctx := context.Background()
	conn, err := grpc.DialContext(
		ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure(),
	)
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()

	client := pb.NewDOQClient(conn)

	proxy := NewGRPCProxy()
	proxy.leader = "bufnet"
	proxy.client = client

	mockNode.On("UpdatePriority", "test-queue", uint64(32), int64(256)).Return(nil)

	respUpdatePriority, err := proxy.UpdatePriority(
		context.Background(),
		"bufnet",
		&pb.UpdatePriorityRequest{
			QueueName: "test-queue",
			Id:        uint64(32),
			Priority:  int64(256),
		},
	)
	assert.Nil(t, err)
	assert.Equal(t, true, respUpdatePriority.Success)
}
