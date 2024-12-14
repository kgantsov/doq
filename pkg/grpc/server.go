package grpc

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/kgantsov/doq/pkg/http"
	pb "github.com/kgantsov/doq/pkg/proto"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
)

type QueueServer struct {
	pb.UnimplementedDOQServer
	node  http.Node
	proxy *GRPCProxy
	port  int

	queueConsumers map[string]map[uint64]chan struct{}
	mu             sync.RWMutex
	timerPool      sync.Pool
}

func NewQueueServer(node http.Node, port int) *QueueServer {
	return &QueueServer{
		node:           node,
		port:           port,
		queueConsumers: make(map[string]map[uint64]chan struct{}),
		proxy:          NewGRPCProxy(nil, port),
		timerPool: sync.Pool{
			New: func() interface{} {
				// Create a new timer, but immediately stop it so it can be reused.
				t := time.NewTimer(0)
				if !t.Stop() {
					<-t.C // drain the channel if it was already fired
				}
				return t
			},
		},
	}
}

func NewGRPCServer(node http.Node, port int) (*grpc.Server, error) {
	grpcServer := grpc.NewServer()
	pb.RegisterDOQServer(grpcServer, NewQueueServer(node, port))

	return grpcServer, nil
}

// Function to get a timer from the pool
func (s *QueueServer) getTimer(duration time.Duration) *time.Timer {
	timer := s.timerPool.Get().(*time.Timer)
	timer.Reset(duration) // Reset it to the desired duration
	return timer
}

// Function to put a timer back in the pool
func (s *QueueServer) putTimer(timer *time.Timer) {
	if !timer.Stop() {
		<-timer.C // Drain the channel if it fired
	}
	s.timerPool.Put(timer)
}

// CreateQueue creates a new queue
func (s *QueueServer) CreateQueue(ctx context.Context, req *pb.CreateQueueRequest) (*pb.CreateQueueResponse, error) {
	if !s.node.IsLeader() {
		return s.proxy.CreateQueue(ctx, s.node.Leader(), req)
	}

	err := s.node.CreateQueue(req.Type, req.Name)
	if err != nil {
		return &pb.CreateQueueResponse{Success: false}, fmt.Errorf("failed to create a queue %s", req.Name)
	}

	return &pb.CreateQueueResponse{Success: true}, nil
}

// DeleteQueue deletes a queue
func (s *QueueServer) DeleteQueue(ctx context.Context, req *pb.DeleteQueueRequest) (*pb.DeleteQueueResponse, error) {
	if !s.node.IsLeader() {
		return s.proxy.DeleteQueue(ctx, s.node.Leader(), req)
	}

	err := s.node.DeleteQueue(req.Name)

	if err != nil {
		return &pb.DeleteQueueResponse{Success: false}, fmt.Errorf("failed to delete a queue %s", req.Name)
	}

	return &pb.DeleteQueueResponse{Success: true}, nil
}

// Enqueue implements client-side streaming for enqueuing messages
func (s *QueueServer) Enqueue(ctx context.Context, req *pb.EnqueueRequest) (*pb.EnqueueResponse, error) {
	if !s.node.IsLeader() {
		return s.proxy.Enqueue(ctx, s.node.Leader(), req)
	}

	message, err := s.node.Enqueue(req.QueueName, req.Group, req.Priority, req.Content)

	if err != nil {
		return &pb.EnqueueResponse{Success: false}, fmt.Errorf("failed to enqueue a message")
	}

	s.broadcastMessage(req.QueueName, struct{}{})

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
	if !s.node.IsLeader() {
		return s.proxy.EnqueueStream(stream, s.node.Leader())
	}

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

		s.broadcastMessage(req.QueueName, struct{}{})

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
	}
}

// Dequeue implements server-side streaming for dequeuing messages
func (s *QueueServer) Dequeue(ctx context.Context, req *pb.DequeueRequest) (*pb.DequeueResponse, error) {
	if !s.node.IsLeader() {
		return s.proxy.Dequeue(ctx, s.node.Leader(), req)
	}

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

// DequeueStream implements server-side streaming for dequeuing messages
func (s *QueueServer) DequeueStream(req *pb.DequeueRequest, stream pb.DOQ_DequeueStreamServer) error {
	if !s.node.IsLeader() {
		return s.proxy.DequeueStream(req, stream, s.node.Leader())
	}

	consumerChan := make(chan struct{})
	consumerID := s.node.GenerateID()
	s.registerConsumer(req.QueueName, consumerID, consumerChan)
	defer s.unregisterConsumer(req.QueueName, consumerID)

	ticker := time.NewTicker(1 * time.Second)

	// Stream messages to the consumer
	for {
		select {
		case <-stream.Context().Done():
			log.Debug().Msgf("Dequeue stream for consumer %d closed", consumerID)
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
				message, err := s.node.Dequeue(req.QueueName, req.Ack)
				if err != nil {
					break
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
					log.Debug().Msgf("Failed to send dequeue response for consumer %d", consumerID)
					return err
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
		for consumerID, consumerChan := range consumers {
			log.Debug().Msgf("Broadcasting a message to a consumer %d", consumerID)

			timer := s.getTimer(1 * time.Second)
			select {
			case consumerChan <- message:
				log.Debug().Msgf("Message broadcasted to a consumer %d", consumerID)
				s.putTimer(timer)
			case <-timer.C:
				log.Debug().Msgf("Failed to broadcast a message to a consumer %d", consumerID)
			}
		}
	}
}

// Ack message handling (this could be used to confirm message processing, depending on your requirements)
func (s *QueueServer) Ack(ctx context.Context, req *pb.AckRequest) (*pb.AckResponse, error) {
	if !s.node.IsLeader() {
		return s.proxy.Ack(ctx, s.node.Leader(), req)
	}

	err := s.node.Ack(req.QueueName, req.Id)
	if err != nil {
		return &pb.AckResponse{Success: false}, fmt.Errorf("failed to ack a message")
	}

	return &pb.AckResponse{Success: true}, nil
}

// Nack message handling (this could be used to confirm message processing, depending on your requirements)
func (s *QueueServer) Nack(ctx context.Context, req *pb.NackRequest) (*pb.NackResponse, error) {
	if !s.node.IsLeader() {
		return s.proxy.Nack(ctx, s.node.Leader(), req)
	}

	err := s.node.Nack(req.QueueName, req.Id)
	if err != nil {
		return &pb.NackResponse{Success: false}, fmt.Errorf("failed to ack a message")
	}

	return &pb.NackResponse{Success: true}, nil
}

// Acknowledge message handling (this could be used to confirm message processing, depending on your requirements)
func (s *QueueServer) UpdatePriority(ctx context.Context, req *pb.UpdatePriorityRequest) (*pb.UpdatePriorityResponse, error) {
	if !s.node.IsLeader() {
		return s.proxy.UpdatePriority(ctx, s.node.Leader(), req)
	}

	err := s.node.UpdatePriority(req.QueueName, req.Id, req.Priority)
	if err != nil {
		return &pb.UpdatePriorityResponse{Success: false}, fmt.Errorf("failed to update prioprity of a message")
	}

	return &pb.UpdatePriorityResponse{Success: true}, nil
}
