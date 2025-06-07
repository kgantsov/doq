package grpc

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/kgantsov/doq/pkg/entity"
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
}

func NewQueueServer(node http.Node, port int) *QueueServer {
	return &QueueServer{
		node:  node,
		port:  port,
		proxy: NewGRPCProxy(nil, port),
	}
}

func NewGRPCServer(node http.Node, port int) (*grpc.Server, error) {
	grpcServer := grpc.NewServer()
	pb.RegisterDOQServer(grpcServer, NewQueueServer(node, port))

	return grpcServer, nil
}

// CreateQueue creates a new queue
func (s *QueueServer) CreateQueue(
	ctx context.Context,
	req *pb.CreateQueueRequest,
) (*pb.CreateQueueResponse, error) {
	if !s.node.IsLeader() {
		return s.proxy.CreateQueue(ctx, s.node.Leader(), req)
	}

	err := s.node.CreateQueue(
		req.Type,
		req.Name,
		entity.QueueSettings{
			Strategy:   req.Settings.Strategy.String(),
			MaxUnacked: int(req.Settings.MaxUnacked),
		},
	)
	if err != nil {
		return &pb.CreateQueueResponse{Success: false}, fmt.Errorf(
			"failed to create a queue %s", req.Name,
		)
	}

	return &pb.CreateQueueResponse{Success: true}, nil
}

// DeleteQueue deletes a queue with all messages
func (s *QueueServer) DeleteQueue(
	ctx context.Context,
	req *pb.DeleteQueueRequest,
) (*pb.DeleteQueueResponse, error) {
	if !s.node.IsLeader() {
		return s.proxy.DeleteQueue(ctx, s.node.Leader(), req)
	}

	err := s.node.DeleteQueue(req.Name)

	if err != nil {
		return &pb.DeleteQueueResponse{Success: false}, fmt.Errorf(
			"failed to delete a queue %s", req.Name,
		)
	}

	return &pb.DeleteQueueResponse{Success: true}, nil
}

// Enqueue enqueues a message to a queue
func (s *QueueServer) Enqueue(
	ctx context.Context,
	req *pb.EnqueueRequest,
) (*pb.EnqueueResponse, error) {
	if !s.node.IsLeader() {
		return s.proxy.Enqueue(ctx, s.node.Leader(), req)
	}

	message, err := s.node.Enqueue(
		req.QueueName, req.Group, req.Priority, req.Content, req.Metadata,
	)

	if err != nil {
		return &pb.EnqueueResponse{Success: false}, fmt.Errorf("failed to enqueue a message")
	}

	return &pb.EnqueueResponse{
		Success:  true,
		Id:       message.ID,
		Group:    message.Group,
		Priority: message.Priority,
		Content:  message.Content,
		Metadata: message.Metadata,
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

		message, err := s.node.Enqueue(
			req.QueueName, req.Group, req.Priority, req.Content, req.Metadata,
		)
		if err != nil {
			return fmt.Errorf("failed to enqueue a message")
		}

		err = stream.Send(&pb.EnqueueResponse{
			Success:  true,
			Id:       message.ID,
			Group:    message.Group,
			Priority: message.Priority,
			Content:  message.Content,
			Metadata: message.Metadata,
		})
		if err != nil {
			return err
		}
	}
}

// Dequeue dequeues a message from a queue
func (s *QueueServer) Dequeue(
	ctx context.Context,
	req *pb.DequeueRequest,
) (*pb.DequeueResponse, error) {
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
		Metadata: message.Metadata,
	}, nil
}

// DequeueStream implements server-side streaming for dequeuing messages
func (s *QueueServer) DequeueStream(stream pb.DOQ_DequeueStreamServer) error {
	if !s.node.IsLeader() {
		return s.proxy.DequeueStream(stream, s.node.Leader())
	}

	var queueName string
	var ack bool

	req, err := stream.Recv()
	if err != nil {
		log.Warn().Msg("Client closed the stream or encountered an error")
		return err
	}

	// Capture initial params
	if req.QueueName != "" {
		queueName = req.QueueName
		ack = req.Ack
	}

	for {
		select {
		case <-stream.Context().Done():
			log.Info().Msgf("Dequeue stream for consumer closed")
			return nil
		default:
			message, err := s.node.Dequeue(queueName, ack)
			if err != nil {
				log.Info().Err(err).Msg("Failed to dequeue message")

				time.Sleep(1 * time.Second)
				continue
			}

			log.Info().Msgf("Dequeued message %d from queue %s", message.ID, queueName)

			err = stream.Send(&pb.DequeueResponse{
				Success:  true,
				Id:       message.ID,
				Group:    message.Group,
				Priority: message.Priority,
				Content:  message.Content,
				Metadata: message.Metadata,
			})
			if err != nil {
				log.Warn().Msgf("Failed to send dequeue response for consumer")
				return err
			}

			// receive the signal from the consumer that it is ready for the next message
			req, err = stream.Recv()
			if err != nil {
				log.Warn().Msg("Client closed the stream or encountered an error")
				return err
			}
		}
	}
}

// Get returns a message from a queue by ID
func (s *QueueServer) Get(ctx context.Context, req *pb.GetRequest) (*pb.GetResponse, error) {
	if !s.node.IsLeader() {
		return s.proxy.Get(ctx, s.node.Leader(), req)
	}

	message, err := s.node.Get(req.QueueName, req.Id)
	if err != nil {
		return &pb.GetResponse{Success: false}, fmt.Errorf("failed to get a message")
	}

	return &pb.GetResponse{
		Success:  true,
		Id:       message.ID,
		Group:    message.Group,
		Priority: message.Priority,
		Content:  message.Content,
		Metadata: message.Metadata,
	}, nil
}

// Delete deletes a message from a queue by ID (this could be used to remove a message from a queue
// in case of processing failure if max attempts reached)
func (s *QueueServer) Delete(
	ctx context.Context,
	req *pb.DeleteRequest,
) (*pb.DeleteResponse, error) {
	if !s.node.IsLeader() {
		return s.proxy.Delete(ctx, s.node.Leader(), req)
	}

	err := s.node.Delete(req.QueueName, req.Id)
	if err != nil {
		return &pb.DeleteResponse{Success: false}, fmt.Errorf("failed to delete a message")
	}

	return &pb.DeleteResponse{
		Success: true,
	}, nil
}

// Ack acknowledges a message (this could be used to confirm message processing)
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

// Nack negatively acknowledges a message (this could be used to put a message back to the queue
// in case of processing failure)
func (s *QueueServer) Nack(ctx context.Context, req *pb.NackRequest) (*pb.NackResponse, error) {
	if !s.node.IsLeader() {
		return s.proxy.Nack(ctx, s.node.Leader(), req)
	}

	err := s.node.Nack(req.QueueName, req.Id, req.Priority, req.Metadata)
	if err != nil {
		return &pb.NackResponse{Success: false}, fmt.Errorf("failed to ack a message")
	}

	return &pb.NackResponse{Success: true}, nil
}

// UpdatePriority updates priority of a message
func (s *QueueServer) UpdatePriority(
	ctx context.Context,
	req *pb.UpdatePriorityRequest,
) (*pb.UpdatePriorityResponse, error) {
	if !s.node.IsLeader() {
		return s.proxy.UpdatePriority(ctx, s.node.Leader(), req)
	}

	err := s.node.UpdatePriority(req.QueueName, req.Id, req.Priority)
	if err != nil {
		return &pb.UpdatePriorityResponse{Success: false}, fmt.Errorf(
			"failed to update prioprity of a message",
		)
	}

	return &pb.UpdatePriorityResponse{Success: true}, nil
}
