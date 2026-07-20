package grpc

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/kgantsov/doq/pkg/config"
	"github.com/kgantsov/doq/pkg/entity"
	"github.com/kgantsov/doq/pkg/errors"
	"github.com/kgantsov/doq/pkg/http"
	pb "github.com/kgantsov/doq/pkg/proto"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
)

// emptyQueuePollInterval bounds how long DequeueStream blocks waiting for a
// message before retrying. It is a safety net behind the enqueue notification
// (NotifyChan): the consumer is normally woken the instant a message is
// enqueued, and only falls back to this interval if a wake-up is missed.
const emptyQueuePollInterval = 1 * time.Second

type QueueServer struct {
	pb.UnimplementedDOQServer
	node http.Node
	port int
}

func NewQueueServer(node http.Node, port int) *QueueServer {
	return &QueueServer{
		node: node,
		port: port,
	}
}

func NewGRPCServer(config *config.Config, node http.Node, port int) (*grpc.Server, error) {
	var prometheusMetrics *PrometheusMetrics
	if config.Prometheus.Enabled {
		promRegistry := node.PrometheusRegistry()
		prometheusMetrics = NewPrometheusMetrics(promRegistry, "doq", "grpc")
		promRegistry.Register(collectors.NewGoCollector())
	}

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(UnaryInterceptor(config.Prometheus.Enabled, prometheusMetrics)),
		grpc.StreamInterceptor(StreamInterceptor(config.Prometheus.Enabled, prometheusMetrics)),
	)
	pb.RegisterDOQServer(grpcServer, NewQueueServer(node, port))

	return grpcServer, nil
}

func (s *QueueServer) GenerateIDs(
	ctx context.Context,
	req *pb.GenerateIDsRequest,
) (*pb.GenerateIDsResponse, error) {

	if req.Number <= 0 || req.Number > 1000 {
		return &pb.GenerateIDsResponse{Success: false}, fmt.Errorf("number must be between 1 and 1000")
	}

	ids := make([]uint64, 0, req.Number)
	for i := 0; i < int(req.Number); i++ {
		id := s.node.GenerateID()
		ids = append(ids, id)
	}

	return &pb.GenerateIDsResponse{Ids: ids, Success: true}, nil
}

// CreateQueue creates a new queue
func (s *QueueServer) CreateQueue(
	ctx context.Context,
	req *pb.CreateQueueRequest,
) (*pb.CreateQueueResponse, error) {
	if req.Settings == nil {
		return &pb.CreateQueueResponse{Success: false}, fmt.Errorf("invalid settings provided")
	}

	err := s.node.CreateQueue(
		req.Type,
		req.Name,
		entity.QueueSettings{
			Strategy:   strings.ToUpper(req.Settings.Strategy.String()),
			MaxUnacked: int(req.Settings.MaxUnacked),
			AckTimeout: req.Settings.AckTimeout,
		},
	)
	if err != nil {
		log.Error().Str("component", "grpc").Err(err).Msgf("Failed to create a queue %s", req.Name)
		return &pb.CreateQueueResponse{Success: false}, fmt.Errorf(
			"failed to create a queue %s", req.Name,
		)
	}

	return &pb.CreateQueueResponse{Success: true}, nil
}

// UpdateQueue updates a queue settings
func (s *QueueServer) UpdateQueue(
	ctx context.Context,
	req *pb.UpdateQueueRequest,
) (*pb.UpdateQueueResponse, error) {
	if req.Settings == nil {
		return &pb.UpdateQueueResponse{Success: false}, fmt.Errorf(
			"queue settings are required",
		)
	}

	err := s.node.UpdateQueue(
		req.Name,
		entity.QueueSettings{
			Strategy:   strings.ToUpper(req.Settings.Strategy.String()),
			MaxUnacked: int(req.Settings.MaxUnacked),
			AckTimeout: req.Settings.AckTimeout,
		},
	)
	if err != nil {
		log.Error().Str("component", "grpc").Err(err).Msgf("Failed to update a queue %s", req.Name)
		return &pb.UpdateQueueResponse{Success: false}, fmt.Errorf(
			"failed to update a queue %s", req.Name,
		)
	}

	return &pb.UpdateQueueResponse{Success: true}, nil
}

// DeleteQueue deletes a queue with all messages
func (s *QueueServer) DeleteQueue(
	ctx context.Context,
	req *pb.DeleteQueueRequest,
) (*pb.DeleteQueueResponse, error) {
	err := s.node.DeleteQueue(req.Name)

	if err != nil {
		log.Error().Str("component", "grpc").Err(err).Msgf("Failed to delete a queue %s", req.Name)
		return &pb.DeleteQueueResponse{Success: false}, fmt.Errorf(
			"failed to delete a queue %s", req.Name,
		)
	}

	return &pb.DeleteQueueResponse{Success: true}, nil
}

// GetQueues returns a list of all queues
func (s *QueueServer) GetQueues(
	ctx context.Context,
	req *pb.GetQueuesRequest,
) (*pb.GetQueuesResponse, error) {

	queues := s.node.GetQueues()
	queueInfos := make([]*pb.GetQueueResponse, len(queues))

	for i, queue := range queues {
		queueInfos[i] = &pb.GetQueueResponse{
			Name: queue.Name,
			Type: queue.Type,
			Settings: &pb.QueueSettings{
				Strategy:   pb.QueueSettings_Strategy(pb.QueueSettings_Strategy_value[queue.Settings.Strategy]),
				MaxUnacked: uint32(queue.Settings.MaxUnacked),
				AckTimeout: queue.Settings.AckTimeout,
			},
			Stats: &pb.Stats{
				EnqueueRPS: queue.Stats.EnqueueRPS,
				DequeueRPS: queue.Stats.DequeueRPS,
				AckRPS:     queue.Stats.AckRPS,
				NackRPS:    queue.Stats.NackRPS,
			},
			Ready:   queue.Ready,
			Unacked: queue.Unacked,
			Total:   queue.Total,
		}
	}

	return &pb.GetQueuesResponse{Queues: queueInfos}, nil
}

// GetQueue returns a queue by name
func (s *QueueServer) GetQueue(
	ctx context.Context,
	req *pb.GetQueueRequest,
) (*pb.GetQueueResponse, error) {

	queue, err := s.node.GetQueueInfo(req.Name)
	if err != nil {
		log.Error().
			Str("component", "grpc").
			Err(err).
			Msgf("Failed to get a queue %s", req.Name)
		return nil, fmt.Errorf("failed to get a queue %s", req.Name)
	}

	return &pb.GetQueueResponse{
		Name: queue.Name,
		Type: queue.Type,
		Settings: &pb.QueueSettings{
			Strategy:   pb.QueueSettings_Strategy(pb.QueueSettings_Strategy_value[queue.Settings.Strategy]),
			MaxUnacked: uint32(queue.Settings.MaxUnacked),
			AckTimeout: queue.Settings.AckTimeout,
		},
		Stats: &pb.Stats{
			EnqueueRPS: queue.Stats.EnqueueRPS,
			DequeueRPS: queue.Stats.DequeueRPS,
			AckRPS:     queue.Stats.AckRPS,
			NackRPS:    queue.Stats.NackRPS,
		},
		Ready:   queue.Ready,
		Unacked: queue.Unacked,
		Total:   queue.Total,
	}, nil
}

// Enqueue enqueues a message to a queue
func (s *QueueServer) Enqueue(
	ctx context.Context,
	req *pb.EnqueueRequest,
) (*pb.EnqueueResponse, error) {
	message, err := s.node.Enqueue(
		req.QueueName, req.Id, req.Group, req.Priority, req.Content, req.Metadata,
	)

	if err != nil {
		log.Error().
			Str("component", "grpc").
			Err(err).
			Msgf("Failed to enqueue a message to queue %s", req.QueueName)
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
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			return nil
		}

		if err != nil {
			return err
		}

		message, err := s.node.Enqueue(
			req.QueueName, req.Id, req.Group, req.Priority, req.Content, req.Metadata,
		)
		if err != nil {
			log.Error().Str("component", "grpc").Err(err).Msgf(
				"Failed to enqueue a message to queue %s", req.QueueName,
			)
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
			log.Error().Str("component", "grpc").Err(err).Msgf(
				"Failed to send enqueue response for queue %s", req.QueueName,
			)
			return err
		}
	}
}

// Dequeue dequeues a message from a queue
func (s *QueueServer) Dequeue(
	ctx context.Context,
	req *pb.DequeueRequest,
) (*pb.DequeueResponse, error) {
	message, err := s.node.Dequeue(req.QueueName, req.Ack)
	if err != nil {
		// An empty queue is an expected, high-frequency outcome; only genuine
		// failures deserve an error log.
		if err == errors.ErrEmptyQueue {
			log.Debug().Str("component", "grpc").Err(err).Msgf(
				"Queue %s is empty", req.QueueName,
			)
		} else {
			log.Error().Str("component", "grpc").Err(err).Msgf(
				"Failed to dequeue a message from queue %s", req.QueueName,
			)
		}
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
	var queueName string
	var ack bool

	req, err := stream.Recv()
	if err != nil {
		log.Warn().Str("component", "grpc").Msg("Client closed the stream or encountered an error")
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
			log.Info().Str("component", "grpc").Msgf("Dequeue stream for consumer closed")
			return nil
		default:
			// Grab the wake channel before attempting the dequeue so an enqueue
			// that races in between still wakes us instead of being missed.
			notify := s.node.NotifyChan(queueName)

			message, err := s.node.Dequeue(queueName, ack)
			if err != nil {
				// An empty queue is the expected case here (we block and wait);
				// a real failure should still surface at error level.
				if err == errors.ErrEmptyQueue {
					log.Debug().Str("component", "grpc").Err(err).Msg("Queue empty, waiting for a message")
				} else {
					log.Error().Str("component", "grpc").Err(err).Msg("Failed to dequeue message")
				}

				// Queue is empty (or unavailable): block until a message is
				// enqueued, a safety timeout elapses, or the consumer
				// disconnects — rather than a fixed poll sleep. For a delayed
				// message maturing sooner than the poll interval, wake exactly
				// when it becomes ready; cap at emptyQueuePollInterval so a
				// stale follower clock still gets re-checked.
				wait := emptyQueuePollInterval
				if _, nextReadyIn := s.node.PeekReady(queueName); nextReadyIn > 0 && nextReadyIn < wait {
					wait = nextReadyIn
				}
				select {
				case <-notify:
				case <-time.After(wait):
				case <-stream.Context().Done():
					return nil
				}
				continue
			}

			log.Debug().
				Str("component", "grpc").
				Msgf("Dequeued message %d from queue %s", message.ID, queueName)

			err = stream.Send(&pb.DequeueResponse{
				Success:  true,
				Id:       message.ID,
				Group:    message.Group,
				Priority: message.Priority,
				Content:  message.Content,
				Metadata: message.Metadata,
			})
			if err != nil {
				log.Warn().
					Str("component", "grpc").
					Msgf("Failed to send dequeue response for consumer")
				return err
			}

			// receive the signal from the consumer that it is ready for the next message
			req, err = stream.Recv()
			if err != nil {
				log.Warn().
					Str("component", "grpc").
					Msg("Client closed the stream or encountered an error")
				return err
			}
		}
	}
}

// Get returns a message from a queue by ID
func (s *QueueServer) Get(ctx context.Context, req *pb.GetRequest) (*pb.GetResponse, error) {
	message, err := s.node.Get(req.QueueName, req.Id)
	if err != nil {
		log.Error().Str("component", "grpc").Err(err).Msgf(
			"Failed to get a message %d from queue %s", req.Id, req.QueueName,
		)
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
	err := s.node.Delete(req.QueueName, req.Id)
	if err != nil {
		log.Error().Str("component", "grpc").Err(err).Msgf(
			"Failed to delete a message %d from queue %s", req.Id, req.QueueName,
		)
		return &pb.DeleteResponse{Success: false}, fmt.Errorf("failed to delete a message")
	}

	return &pb.DeleteResponse{
		Success: true,
	}, nil
}

// Ack acknowledges a message (this could be used to confirm message processing)
func (s *QueueServer) Ack(ctx context.Context, req *pb.AckRequest) (*pb.AckResponse, error) {
	err := s.node.Ack(req.QueueName, req.Id)
	if err != nil {
		log.Error().Str("component", "grpc").Err(err).Msgf(
			"Failed to ack a message %d from queue %s", req.Id, req.QueueName,
		)
		return &pb.AckResponse{Success: false}, fmt.Errorf("failed to ack a message")
	}

	return &pb.AckResponse{Success: true}, nil
}

// Nack negatively acknowledges a message (this could be used to put a message back to the queue
// in case of processing failure)
func (s *QueueServer) Nack(ctx context.Context, req *pb.NackRequest) (*pb.NackResponse, error) {
	err := s.node.Nack(req.QueueName, req.Id, req.Priority, req.Metadata)
	if err != nil {
		log.Error().Str("component", "grpc").Err(err).Msgf(
			"Failed to nack a message %d from queue %s", req.Id, req.QueueName,
		)
		return &pb.NackResponse{Success: false}, fmt.Errorf("failed to ack a message")
	}

	return &pb.NackResponse{Success: true}, nil
}

// Touch touches a message (this could be used to notify the queue that the message is still being processed)
func (s *QueueServer) Touch(ctx context.Context, req *pb.TouchRequest) (*pb.TouchResponse, error) {
	err := s.node.Touch(req.QueueName, req.Id)
	if err != nil {
		log.Error().Str("component", "grpc").Err(err).Msgf(
			"Failed to touch a message %d from queue %s", req.Id, req.QueueName,
		)
		return &pb.TouchResponse{Success: false}, fmt.Errorf("failed to touch a message")
	}

	return &pb.TouchResponse{Success: true}, nil
}

// UpdatePriority updates priority of a message
func (s *QueueServer) UpdatePriority(
	ctx context.Context,
	req *pb.UpdatePriorityRequest,
) (*pb.UpdatePriorityResponse, error) {
	err := s.node.UpdatePriority(req.QueueName, req.Id, req.Priority)
	if err != nil {
		log.Error().Str("component", "grpc").Err(err).Msgf(
			"Failed to update priority of a message %d from queue %s", req.Id, req.QueueName,
		)
		return &pb.UpdatePriorityResponse{Success: false}, fmt.Errorf(
			"failed to update prioprity of a message",
		)
	}

	return &pb.UpdatePriorityResponse{Success: true}, nil
}
