package grpc

import (
	"context"
	"fmt"
	"log"
	"net"
	"strings"
	"testing"

	"github.com/kgantsov/doq/pkg/config"
	"github.com/kgantsov/doq/pkg/entity"
	"github.com/kgantsov/doq/pkg/http"
	"github.com/kgantsov/doq/pkg/metrics"
	"github.com/kgantsov/doq/pkg/mocks"
	pb "github.com/kgantsov/doq/pkg/proto"
	"github.com/kgantsov/doq/pkg/queue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

var lis *bufconn.Listener

func startTestServer(node http.Node) {
	lis = bufconn.Listen(bufSize)

	cfg := &config.Config{
		Http: config.HttpConfig{Port: "8080"},
		Prometheus: config.PrometheusConfig{
			Enabled: false,
		},
	}

	grpcServer, _ := NewGRPCServer(cfg, node, 0)

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
	mockNode := mocks.NewMockNode()
	startTestServer(mockNode)

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

	var idCounter uint64 = 0
	mockNode.NextIDFunc = func() uint64 {
		idCounter++
		return idCounter
	}
	// mockNode.On("GenerateID").Return(uint64(0), nil)

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
	mockNode := mocks.NewMockNode()
	startTestServer(mockNode)

	tests := []struct {
		name          string
		queueName     string
		queueType     string
		queueSettings *pb.QueueSettings
		error         error
	}{
		{
			name:          "Delayed Queue Nil Settings",
			queueName:     "test-queue",
			queueType:     "delayed",
			queueSettings: nil,
			error:         fmt.Errorf("rpc error: code = Unknown desc = invalid settings provided"),
		},
		{
			name:          "Delayed Queue Empty Settings",
			queueName:     "test-queue",
			queueType:     "delayed",
			queueSettings: &pb.QueueSettings{},
			error:         nil,
		},
		{
			name:          "Fair Queue Nil Settings",
			queueName:     "test-queue",
			queueType:     "fair",
			queueSettings: nil,
			error:         fmt.Errorf("rpc error: code = Unknown desc = invalid settings provided"),
		},
		{
			name:          "Fair Queue Empty Settings",
			queueName:     "test-queue",
			queueType:     "fair",
			queueSettings: &pb.QueueSettings{},
			error:         nil,
		},
		{
			name:      "Fair Queue With Settings",
			queueName: "test-queue",
			queueType: "fair",
			queueSettings: &pb.QueueSettings{
				Strategy:   pb.QueueSettings_WEIGHTED,
				MaxUnacked: 40,
				AckTimeout: 3600,
			},
			error: nil,
		},
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
		t.Run(tt.name, func(t *testing.T) {
			var settings entity.QueueSettings
			if tt.queueSettings != nil {
				settings.Strategy = strings.ToUpper(tt.queueSettings.Strategy.String())
				settings.MaxUnacked = int(tt.queueSettings.MaxUnacked)
				settings.AckTimeout = tt.queueSettings.AckTimeout
			}
			mockNode.On(
				"CreateQueue",
				tt.queueType,
				tt.queueName,
				settings,
			).Return(nil)

			req := &pb.CreateQueueRequest{
				Name:     tt.queueName,
				Type:     tt.queueType,
				Settings: tt.queueSettings,
			}

			resp, err := client.CreateQueue(ctx, req)
			if tt.error != nil {
				assert.Equal(t, tt.error.Error(), err.Error())
			} else {
				assert.True(t, resp.Success, "Queue creation should succeed")
			}

			mockNode.On(
				"CreateQueue",
				tt.queueType,
				tt.queueName,
				settings,
			).Unset()
		})
	}
}

func TestUpdateQueue(t *testing.T) {
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
	mockNode.On("UpdateQueue", "test-queue", entity.QueueSettings{
		Strategy:   "STRATEGY_UNSPECIFIED",
		MaxUnacked: 10,
		AckTimeout: 1200,
	}).Return(nil)

	// Update the queue settings
	reqUpdate := &pb.UpdateQueueRequest{
		Name: "test-queue",
		Settings: &pb.QueueSettings{
			MaxUnacked: 10,
			AckTimeout: 1200,
		},
	}
	resp, err := client.UpdateQueue(ctx, reqUpdate)
	if err != nil {
		t.Fatalf("UpdateQueue failed: %v", err)
	}

	// Use testify's assert to check the response
	assert.True(t, resp.Success, "Queue update should succeed")
}

// Test for DeleteQueue function
func TestDeleteQueue(t *testing.T) {
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

	mockNode.On("DeleteQueue", "test-queue").Return(nil)

	// Test case 1: Delete the queue successfully
	reqDelete := &pb.DeleteQueueRequest{Name: "test-queue"}
	resp, err := client.DeleteQueue(ctx, reqDelete)
	if err != nil {
		t.Fatalf("DeleteQueue failed: %v", err)
	}
	assert.True(t, resp.Success, "Queue deletion should succeed")
}

// Test for Enqueue function
func TestEnqueue(t *testing.T) {
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

	content := "{\"user_id\": 2, \"name\": \"Jane\"}"
	priority := int64(100)
	metadata := map[string]string{"retry": "3"}

	mockNode.On(
		"Enqueue",
		"test-queue",
		uint64(0),
		"default",
		int64(10),
		content,
		metadata,
	).Return(&entity.Message{
		ID:       uint64(1),
		Priority: priority,
		Group:    "default",
		Content:  content,
		Metadata: metadata,
	}, nil)

	// Test case: Enqueue a message successfully
	reqEnqueue := &pb.EnqueueRequest{
		QueueName: "test-queue",
		Content:   content,
		Group:     "default",
		Priority:  10,
		Metadata:  metadata,
	}
	resp, err := client.Enqueue(ctx, reqEnqueue)
	if err != nil {
		t.Fatalf("Enqueue failed: %v", err)
	}
	assert.True(t, resp.Success, "Enqueue should succeed")
}

// Test for Dequeue function
func TestDequeue(t *testing.T) {
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

	content := "{\"user_id\": 124, \"name\": \"John\"}"
	priority := int64(250)
	metadata := map[string]string{"retry": "1"}

	mockNode.On("Dequeue", "test-queue", false).Return(&entity.Message{
		ID:       uint64(5465),
		Priority: priority,
		Group:    "default",
		Content:  content,
		Metadata: metadata,
	}, nil)

	// Test case: Dequeue the message successfully
	reqDequeue := &pb.DequeueRequest{QueueName: "test-queue"}
	resp, err := client.Dequeue(ctx, reqDequeue)
	if err != nil {
		t.Fatalf("Dequeue failed: %v", err)
	}
	assert.Equal(t, content, resp.Content)
	assert.Equal(t, priority, resp.Priority)
	assert.Equal(t, "default", resp.Group)
	assert.Equal(t, metadata, resp.Metadata)
	assert.Equal(t, "1", resp.Metadata["retry"])
}

// Test for Get function
func TestGet(t *testing.T) {
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

	content := "{\"user_id\": 124, \"name\": \"John\"}"
	priority := int64(250)
	metadata := map[string]string{"retry": "1"}

	mockNode.On("Get", "test-queue", uint64(6123)).Return(&entity.Message{
		ID:       uint64(6123),
		Priority: priority,
		Group:    "default",
		Content:  content,
		Metadata: metadata,
	}, nil)

	// Test case: Get the message successfully
	reqGet := &pb.GetRequest{QueueName: "test-queue", Id: uint64(6123)}
	resp, err := client.Get(ctx, reqGet)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	assert.Equal(t, uint64(6123), resp.Id)
	assert.Equal(t, content, resp.Content)
	assert.Equal(t, "default", resp.Group)
	assert.Equal(t, metadata, resp.Metadata)
	assert.Equal(t, "1", resp.Metadata["retry"])
}

func TestUpdatePriority(t *testing.T) {
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

	mockNode.On("UpdatePriority", "test-queue", uint64(32), int64(256)).Return(nil)

	reqUpdatePriority := &pb.UpdatePriorityRequest{
		QueueName: "test-queue",
		Id:        uint64(32),
		Priority:  256,
	}
	respUpdatePriority, err := client.UpdatePriority(ctx, reqUpdatePriority)
	assert.Nil(t, err)
	assert.Equal(t, true, respUpdatePriority.Success)
}

// Test for Acknowledge function
func TestAck(t *testing.T) {
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

	mockNode.On("Ack", "test-queue", uint64(6123)).Return(nil)

	req := &pb.AckRequest{QueueName: "test-queue", Id: 6123}
	respAck, err := client.Ack(ctx, req)
	if err != nil {
		t.Fatalf("Acknowledge failed: %v", err)
	}
	assert.True(t, respAck.Success)
}

func TestNack(t *testing.T) {
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

	mockNode.On(
		"Nack", "my-transcode-queue", uint64(23421), int64(100), map[string]string{"retry": "5"},
	).Return(nil)

	req := &pb.NackRequest{
		QueueName: "my-transcode-queue",
		Id:        uint64(23421),
		Priority:  100,
		Metadata:  map[string]string{"retry": "5"},
	}
	respAck, err := client.Nack(ctx, req)
	require.NoError(t, err)
	assert.True(t, respAck.Success)
}

func TestTouch(t *testing.T) {
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

	mockNode.On("Touch", "test-queue", uint64(6123)).Return(nil)

	req := &pb.TouchRequest{QueueName: "test-queue", Id: uint64(6123)}
	respAck, err := client.Touch(ctx, req)
	require.NoError(t, err)
	assert.True(t, respAck.Success, "Touch should succeed")
}

func TestEnqueuetStream(t *testing.T) {
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

	enqueueStream, err := client.EnqueueStream(ctx)
	assert.Nil(t, err, "Failed to open stream")

	mockNode.On(
		"Enqueue",
		"test-queue",
		uint64(0),
		"default",
		int64(10),
		"test-message-1",
		map[string]string{"retry": "3"},
	).Return(&entity.Message{
		ID:       uint64(1),
		Priority: 10,
		Group:    "default",
		Content:  "test-message-1",
		Metadata: map[string]string{"retry": "3"},
	}, nil)

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

	mockNode.On(
		"Enqueue",
		"test-queue",
		uint64(0),
		"default",
		int64(10),
		"test-message-1",
		map[string]string{"retry": "3"},
	).Unset()

	mockNode.On(
		"Enqueue",
		"test-queue",
		uint64(0),
		"default",
		int64(132),
		"test-message-126346",
		map[string]string{"retry": "5"},
	).Return(&entity.Message{
		ID:       uint64(2),
		Priority: 132,
		Group:    "default",
		Content:  "test-message-1",
		Metadata: map[string]string{"retry": "3"},
	}, nil)

	enqueueStream.Send(
		&pb.EnqueueRequest{
			QueueName: "test-queue",
			Content:   "test-message-126346",
			Group:     "default",
			Priority:  132,
			Metadata:  map[string]string{"retry": "5"},
		},
	)
	_, err = enqueueStream.Recv()
	assert.Nil(t, err)
}

func TestDequeueStream(t *testing.T) {
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

	mockNode.On(
		"Dequeue",
		"test-queue",
		true,
	).Return(&entity.Message{
		ID:       uint64(1),
		Priority: 10,
		Group:    "default",
		Content:  "test-message-1",
		Metadata: map[string]string{"retry": "3"},
	}, nil)

	// Open a dequeueStream to receive messages from the queue
	dequeueStream, err := client.DequeueStream(ctx)
	assert.Nil(t, err, "Failed to open stream")

	err = dequeueStream.Send(&pb.DequeueRequest{
		QueueName: "test-queue",
		Ack:       true,
	})
	assert.Nil(t, err, "Failed to send DequeueRequest")

	// Consume messages from the stream
	msg, err := dequeueStream.Recv()
	require.NoError(t, err)

	assert.Equal(t, "test-message-1", msg.Content)

	dequeueStream.Send(&pb.DequeueRequest{})

	mockNode.On(
		"Dequeue",
		"test-queue",
		true,
	).Unset()

	mockNode.On(
		"Dequeue",
		"test-queue",
		true,
	).Return(&entity.Message{
		ID:       uint64(2),
		Priority: 50,
		Group:    "default",
		Content:  "test-message-2",
		Metadata: map[string]string{"retry": "3"},
	}, nil)

	// Consume messages from the stream
	msg, err = dequeueStream.Recv()
	require.NoError(t, err)

	assert.Equal(t, "test-message-2", msg.Content)
	assert.Equal(t, uint64(2), msg.Id)
	assert.Equal(t, int64(50), msg.Priority)
	assert.Equal(t, "default", msg.Group)
	assert.Equal(t, "test-message-2", msg.Content)
	assert.Equal(t, map[string]string{"retry": "3"}, msg.Metadata)

	dequeueStream.Send(&pb.DequeueRequest{})
}

func TestGetQueues(t *testing.T) {
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

	mockNode.On("GetQueues").Return([]*queue.QueueInfo{
		{
			Name: "test-queue-1",
			Type: "delayed",
			Settings: entity.QueueSettings{
				Strategy:   "WEIGHTED",
				AckTimeout: 600,
				MaxUnacked: 75,
			},
			Stats: &metrics.Stats{
				EnqueueRPS: 3.1,
				DequeueRPS: 2.5,
				AckRPS:     1.2,
				NackRPS:    2.3,
			},
			Ready:   123,
			Unacked: 456,
			Total:   789,
		},
	})

	resp, err := client.GetQueues(ctx, &pb.GetQueuesRequest{})
	if err != nil {
		t.Fatalf("GetQueues failed: %v", err)
	}

	assert.NotEmpty(t, resp.Queues)
	assert.Equal(t, "test-queue-1", resp.Queues[0].Name)
	assert.Equal(t, "delayed", resp.Queues[0].Type)
	assert.Equal(t, int64(123), resp.Queues[0].Ready)
	assert.Equal(t, int64(456), resp.Queues[0].Unacked)
	assert.Equal(t, int64(789), resp.Queues[0].Total)
	assert.Equal(t, float64(3.1), resp.Queues[0].Stats.EnqueueRPS)
	assert.Equal(t, float64(2.5), resp.Queues[0].Stats.DequeueRPS)
	assert.Equal(t, float64(1.2), resp.Queues[0].Stats.AckRPS)
	assert.Equal(t, float64(2.3), resp.Queues[0].Stats.NackRPS)
}

func TestGetQueue(t *testing.T) {
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

	mockNode.On("GetQueueInfo", "test-queue-3").Return(&queue.QueueInfo{
		Name: "test-queue-3",
		Type: "delayed",
		Settings: entity.QueueSettings{
			Strategy:   "WEIGHTED",
			AckTimeout: 600,
			MaxUnacked: 75,
		},
		Stats: &metrics.Stats{
			EnqueueRPS: 3.1,
			DequeueRPS: 2.5,
			AckRPS:     1.2,
			NackRPS:    2.3,
		},
		Ready:   123,
		Unacked: 456,
		Total:   789,
	}, nil)

	resp, err := client.GetQueue(ctx, &pb.GetQueueRequest{Name: "test-queue-3"})
	if err != nil {
		t.Fatalf("GetQueue failed: %v", err)
	}

	assert.Equal(t, "test-queue-3", resp.Name)
	assert.Equal(t, "delayed", resp.Type)
	assert.Equal(t, int64(123), resp.Ready)
	assert.Equal(t, int64(456), resp.Unacked)
	assert.Equal(t, int64(789), resp.Total)
	assert.Equal(t, float64(3.1), resp.Stats.EnqueueRPS)
	assert.Equal(t, float64(2.5), resp.Stats.DequeueRPS)
	assert.Equal(t, float64(1.2), resp.Stats.AckRPS)
	assert.Equal(t, float64(2.3), resp.Stats.NackRPS)
}
