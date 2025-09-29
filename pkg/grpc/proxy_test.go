package grpc

import (
	"context"
	"testing"

	pb "github.com/kgantsov/doq/pkg/proto"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

func TestProxyCreateQueue(t *testing.T) {
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

	// Send a request to the mock server.
	respCreateQueue, err := proxy.CreateQueue(context.Background(), "bufnet", &pb.CreateQueueRequest{
		Name: "test_queue",
		Type: "fair",
		Settings: &pb.QueueSettings{
			Strategy:   pb.QueueSettings_WEIGHTED,
			MaxUnacked: 10,
			AckTimeout: 600,
		},
	})
	assert.Nil(t, err)
	assert.Equal(t, true, respCreateQueue.Success)

	respDeleteQueue, err := proxy.DeleteQueue(context.Background(), "bufnet", &pb.DeleteQueueRequest{
		Name: "test_queue",
	})
	assert.Nil(t, err)
	assert.Equal(t, true, respDeleteQueue.Success)
}

func TestProxyEnqueueDequeue(t *testing.T) {
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

	// Send a request to the mock server.
	respCreateQueue, err := proxy.CreateQueue(context.Background(), "bufnet", &pb.CreateQueueRequest{
		Name: "test_queue",
		Type: "delayed",
		Settings: &pb.QueueSettings{
			Strategy:   pb.QueueSettings_WEIGHTED,
			MaxUnacked: 10,
			AckTimeout: 600,
		},
	})
	assert.Nil(t, err)
	assert.Equal(t, true, respCreateQueue.Success)

	respEnqueue, err := proxy.Enqueue(context.Background(), "bufnet", &pb.EnqueueRequest{
		QueueName: "test_queue",
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

	respDequeue, err := proxy.Dequeue(context.Background(), "bufnet", &pb.DequeueRequest{
		QueueName: "test_queue",
		Ack:       true,
	})
	assert.Nil(t, err)
	assert.Equal(t, respEnqueue.Id, respDequeue.Id)
	assert.Equal(t, respEnqueue.Group, respDequeue.Group)
	assert.Equal(t, respEnqueue.Priority, respDequeue.Priority)
	assert.Equal(t, respEnqueue.Content, respDequeue.Content)
	assert.Equal(t, true, respDequeue.Success)
}

func TestProxyDelete(t *testing.T) {
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

	// Send a request to the mock server.
	respCreateQueue, err := proxy.CreateQueue(context.Background(), "bufnet", &pb.CreateQueueRequest{
		Name: "test_queue",
		Type: "delayed",
		Settings: &pb.QueueSettings{
			Strategy:   pb.QueueSettings_WEIGHTED,
			MaxUnacked: 10,
			AckTimeout: 600,
		},
	})
	assert.Nil(t, err)
	assert.Equal(t, true, respCreateQueue.Success)

	respEnqueue, err := proxy.Enqueue(context.Background(), "bufnet", &pb.EnqueueRequest{
		QueueName: "test_queue",
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

	respGet, err := proxy.Get(context.Background(), "bufnet", &pb.GetRequest{
		QueueName: "test_queue",
		Id:        respEnqueue.Id,
	})
	assert.Nil(t, err)
	assert.Equal(t, respEnqueue.Id, respGet.Id)
	assert.Equal(t, respEnqueue.Group, respGet.Group)
	assert.Equal(t, respEnqueue.Priority, respGet.Priority)
	assert.Equal(t, respEnqueue.Content, respGet.Content)

	respDelete, err := proxy.Delete(context.Background(), "bufnet", &pb.DeleteRequest{
		QueueName: "test_queue",
		Id:        respEnqueue.Id,
	})
	assert.Nil(t, err)
	assert.Equal(t, true, respDelete.Success)

	respGet, err = proxy.Get(context.Background(), "bufnet", &pb.GetRequest{
		QueueName: "test_queue",
		Id:        respEnqueue.Id,
	})
	assert.NotNil(t, err)
	assert.Nil(t, respGet)
}

func TestProxyNackAck(t *testing.T) {
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

	// Send a request to the mock server.
	respCreateQueue, err := proxy.CreateQueue(context.Background(), "bufnet", &pb.CreateQueueRequest{
		Name: "test_queue",
		Type: "delayed",
		Settings: &pb.QueueSettings{
			Strategy:   pb.QueueSettings_WEIGHTED,
			MaxUnacked: 10,
			AckTimeout: 600,
		},
	})
	assert.Nil(t, err)
	assert.Equal(t, true, respCreateQueue.Success)

	respEnqueue, err := proxy.Enqueue(context.Background(), "bufnet", &pb.EnqueueRequest{
		QueueName: "test_queue",
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

	respDequeue, err := proxy.Dequeue(context.Background(), "bufnet", &pb.DequeueRequest{
		QueueName: "test_queue",
		Ack:       false,
	})
	assert.Nil(t, err)
	assert.Equal(t, respEnqueue.Id, respDequeue.Id)
	assert.Equal(t, respEnqueue.Group, respDequeue.Group)
	assert.Equal(t, respEnqueue.Priority, respDequeue.Priority)
	assert.Equal(t, respEnqueue.Content, respDequeue.Content)
	assert.Equal(t, true, respDequeue.Success)

	respNack, err := proxy.Nack(context.Background(), "bufnet", &pb.NackRequest{
		QueueName: "test_queue",
		Id:        respDequeue.Id,
	})

	assert.Nil(t, err)
	assert.Equal(t, true, respNack.Success)

	respDequeue, err = proxy.Dequeue(context.Background(), "bufnet", &pb.DequeueRequest{
		QueueName: "test_queue",
		Ack:       false,
	})
	assert.Nil(t, err)
	assert.Equal(t, respEnqueue.Id, respDequeue.Id)
	assert.Equal(t, true, respDequeue.Success)

	respAck, err := proxy.Ack(context.Background(), "bufnet", &pb.AckRequest{
		QueueName: "test_queue",
		Id:        respDequeue.Id,
	})

	assert.Nil(t, err)
	assert.Equal(t, true, respAck.Success)

	respDequeue, err = proxy.Dequeue(context.Background(), "bufnet", &pb.DequeueRequest{
		QueueName: "test_queue",
		Ack:       true,
	})
	assert.NotNil(t, err)
	assert.Nil(t, respDequeue)
}

func TestProxyUpdatePriority(t *testing.T) {
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

	// Send a request to the mock server.
	respCreateQueue, err := proxy.CreateQueue(context.Background(), "bufnet", &pb.CreateQueueRequest{
		Name: "test_queue",
		Type: "delayed",
		Settings: &pb.QueueSettings{
			Strategy:   pb.QueueSettings_WEIGHTED,
			MaxUnacked: 10,
			AckTimeout: 600,
		},
	})
	assert.Nil(t, err)
	assert.Equal(t, true, respCreateQueue.Success)

	respEnqueue1, err := proxy.Enqueue(context.Background(), "bufnet", &pb.EnqueueRequest{
		QueueName: "test_queue",
		Group:     "default",
		Priority:  10,
		Content:   "message 1",
	})
	assert.Nil(t, err)
	assert.NotNil(t, respEnqueue1.Id)
	assert.Equal(t, "default", respEnqueue1.Group)
	assert.Equal(t, int64(10), respEnqueue1.Priority)
	assert.Equal(t, "message 1", respEnqueue1.Content)
	assert.Equal(t, true, respEnqueue1.Success)

	respEnqueue2, err := proxy.Enqueue(context.Background(), "bufnet", &pb.EnqueueRequest{
		QueueName: "test_queue",
		Group:     "default",
		Priority:  20,
		Content:   "message 2",
	})
	assert.Nil(t, err)
	assert.NotNil(t, respEnqueue2.Id)
	assert.Equal(t, "default", respEnqueue2.Group)
	assert.Equal(t, int64(20), respEnqueue2.Priority)
	assert.Equal(t, "message 2", respEnqueue2.Content)
	assert.Equal(t, true, respEnqueue2.Success)

	respUpdatePriority, err := proxy.UpdatePriority(
		context.Background(),
		"bufnet",
		&pb.UpdatePriorityRequest{
			QueueName: "test_queue",
			Id:        respEnqueue1.Id,
			Priority:  30,
		},
	)
	assert.Nil(t, err)
	assert.Equal(t, true, respUpdatePriority.Success)

	respDequeue, err := proxy.Dequeue(context.Background(), "bufnet", &pb.DequeueRequest{
		QueueName: "test_queue",
		Ack:       true,
	})
	assert.Nil(t, err)
	assert.Equal(t, respEnqueue2.Id, respDequeue.Id)
	assert.Equal(t, respEnqueue2.Group, respDequeue.Group)
	assert.Equal(t, respEnqueue2.Priority, respDequeue.Priority)
	assert.Equal(t, respEnqueue2.Content, respDequeue.Content)
	assert.Equal(t, true, respDequeue.Success)

	respDequeue, err = proxy.Dequeue(context.Background(), "bufnet", &pb.DequeueRequest{
		QueueName: "test_queue",
		Ack:       true,
	})
	assert.Nil(t, err)
	assert.Equal(t, respEnqueue1.Id, respDequeue.Id)
	assert.Equal(t, respEnqueue1.Group, respDequeue.Group)
	assert.Equal(t, int64(30), respDequeue.Priority)
	assert.Equal(t, respEnqueue1.Content, respDequeue.Content)
	assert.Equal(t, true, respDequeue.Success)
}
