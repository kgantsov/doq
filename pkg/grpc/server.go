package grpc

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/kgantsov/doq/pkg/http"
	pb "github.com/kgantsov/doq/pkg/proto"
	"google.golang.org/grpc"
)

type QueueServer struct {
	pb.UnimplementedDOQServer
	node http.Node

	queueConsumers map[string]map[uint64]chan struct{}
	mu             sync.RWMutex
}

// CreateQueue creates a new queue
func (s *QueueServer) CreateQueue(ctx context.Context, req *pb.CreateQueueRequest) (*pb.CreateQueueResponse, error) {
	err := s.node.CreateQueue(req.Type, req.Name)
	if err != nil {
		return &pb.CreateQueueResponse{Success: false}, fmt.Errorf("failed to create a queue %s", req.Name)
	}

	return &pb.CreateQueueResponse{Success: true}, nil
}

// DeleteQueue deletes a queue
func (s *QueueServer) DeleteQueue(ctx context.Context, req *pb.DeleteQueueRequest) (*pb.DeleteQueueResponse, error) {
	err := s.node.DeleteQueue(req.Name)

	if err != nil {
		return &pb.DeleteQueueResponse{Success: false}, fmt.Errorf("failed to delete a queue %s", req.Name)
	}

	return &pb.DeleteQueueResponse{Success: true}, nil
}

// Enqueue implements client-side streaming for enqueuing messages
func (s *QueueServer) Enqueue(ctx context.Context, req *pb.EnqueueRequest) (*pb.EnqueueResponse, error) {
	message, err := s.node.Enqueue(req.QueueName, req.Group, req.Priority, req.Content)

	if err != nil {
		return &pb.EnqueueResponse{Success: false}, fmt.Errorf("failed to enqueue a message")
	}

	return &pb.EnqueueResponse{
		Success:  true,
		Id:       message.ID,
		Group:    message.Group,
		Priority: message.Priority,
		Content:  message.Content,
	}, nil
}

// EnqueueStream implements client-side streaming for enqueuing messages
func (s *QueueServer) EnqueueStream(stream pb.DOQ_EnqueueStreamServer) error {
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			return nil
		}

		if err != nil {
			return err
		}

		message, err := s.node.Enqueue(req.QueueName, req.Group, req.Priority, req.Content)
		if err != nil {
			return fmt.Errorf("failed to enqueue a message")
		}

		err = stream.Send(&pb.EnqueueResponse{
			Success:  true,
			Id:       message.ID,
			Group:    message.Group,
			Priority: message.Priority,
			Content:  message.Content,
		})
		if err != nil {
			return err
		}

		s.broadcastMessage(req.QueueName, struct{}{})
	}
}

// Dequeue implements server-side streaming for dequeuing messages
func (s *QueueServer) Dequeue(ctx context.Context, req *pb.DequeueRequest) (*pb.DequeueResponse, error) {
	message, err := s.node.Dequeue(req.QueueName, req.Ack)
	if err != nil {
		return &pb.DequeueResponse{Success: false}, fmt.Errorf("failed to dequeue a message")
	}

	return &pb.DequeueResponse{
		Success:  true,
		Id:       message.ID,
		Group:    message.Group,
		Priority: message.Priority,
		Content:  message.Content,
	}, nil
}

// registerConsumer registers a consumer for a queue
func (s *QueueServer) registerConsumer(queueName string, id uint64, consumerChan chan struct{}) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.queueConsumers[queueName]; !ok {
		s.queueConsumers[queueName] = make(map[uint64]chan struct{})
	}

	s.queueConsumers[queueName][id] = consumerChan
}

// unregisterConsumer unregisters a consumer for a queue
func (s *QueueServer) unregisterConsumer(queueName string, id uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.queueConsumers[queueName]; ok {
		delete(s.queueConsumers[queueName], id)
	}
}

func (s *QueueServer) dequeueStream(req *pb.DequeueRequest, stream pb.DOQ_DequeueStreamServer) error {
	message, err := s.node.Dequeue(req.QueueName, req.Ack)
	if err != nil {
		return fmt.Errorf("failed to dequeue a message")
	}

	err = stream.Send(
		&pb.DequeueResponse{
			Success:  true,
			Id:       message.ID,
			Group:    message.Group,
			Priority: message.Priority,
			Content:  message.Content,
		},
	)

	if err != nil {
		return err
	}
	return nil
}

// DequeueStream implements server-side streaming for dequeuing messages
func (s *QueueServer) DequeueStream(req *pb.DequeueRequest, stream pb.DOQ_DequeueStreamServer) error {
	consumerChan := make(chan struct{})
	consumerID := s.node.GenerateID()
	s.registerConsumer(req.QueueName, consumerID, consumerChan)
	defer s.unregisterConsumer(req.QueueName, consumerID)

	ticker := time.NewTicker(1 * time.Second)

	// Stream messages to the consumer
	for {
		select {
		case <-stream.Context().Done():
			return nil
		case <-consumerChan:
			message, err := s.node.Dequeue(req.QueueName, req.Ack)
			if err != nil {
				continue
			}

			err = stream.Send(
				&pb.DequeueResponse{
					Success:  true,
					Id:       message.ID,
					Group:    message.Group,
					Priority: message.Priority,
					Content:  message.Content,
				},
			)

			if err != nil {
				return err
			}
		case <-ticker.C:
			for {
				err := s.dequeueStream(req, stream)
				if err != nil {
					break
				}
			}
		}
	}
}

// broadcastMessage broadcasts a message to all consumers of a queue
func (s *QueueServer) broadcastMessage(queueName string, message struct{}) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if consumers, ok := s.queueConsumers[queueName]; ok {
		for _, consumerChan := range consumers {
			consumerChan <- message
		}
	}
}

// Acknowledge message handling (this could be used to confirm message processing, depending on your requirements)
func (s *QueueServer) Acknowledge(ctx context.Context, req *pb.AcknowledgeRequest) (*pb.AcknowledgeResponse, error) {
	err := s.node.Ack(req.QueueName, req.Id)
	if err != nil {
		return &pb.AcknowledgeResponse{Success: false}, fmt.Errorf("failed to acknowledge a message")
	}

	return &pb.AcknowledgeResponse{Success: true}, nil
}

// Acknowledge message handling (this could be used to confirm message processing, depending on your requirements)
func (s *QueueServer) UpdatePriority(ctx context.Context, req *pb.UpdatePriorityRequest) (*pb.UpdatePriorityResponse, error) {
	err := s.node.UpdatePriority(req.QueueName, req.Id, req.Priority)
	if err != nil {
		return &pb.UpdatePriorityResponse{Success: false}, fmt.Errorf("failed to update prioprity of a message")
	}

	return &pb.UpdatePriorityResponse{Success: true}, nil
}

func NewQueueServer(node http.Node) *QueueServer {
	return &QueueServer{
		node:           node,
		queueConsumers: make(map[string]map[uint64]chan struct{}),
	}
}

func NewGRPCServer(node http.Node) (*grpc.Server, error) {
	grpcServer := grpc.NewServer()
	pb.RegisterDOQServer(grpcServer, NewQueueServer(node))

	return grpcServer, nil
}
