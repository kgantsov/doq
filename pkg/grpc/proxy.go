package grpc

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	pb "github.com/kgantsov/doq/pkg/proto"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
)

type GRPCProxy struct {
	client pb.DOQClient
	leader string
	mu     sync.Mutex
}

func NewGRPCProxy() *GRPCProxy {
	return &GRPCProxy{}
}

// getClient returns the gRPC client for the given host, (re)creating the connection if the
// leader address changed. Safe for concurrent use.
func (p *GRPCProxy) getClient(host string) (pb.DOQClient, error) {
	if host == "" {
		return nil, fmt.Errorf("leader address is empty")
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.leader != host || p.client == nil {
		conn, err := grpc.Dial(host, grpc.WithInsecure())
		if err != nil {
			log.Fatal().Str("component", "proxy").Msgf("Failed to connect: %v", err)
		}
		p.leader = host
		p.client = pb.NewDOQClient(conn)
	}

	return p.client, nil
}

func (p *GRPCProxy) CreateQueue(ctx context.Context, host string, req *pb.CreateQueueRequest) (*pb.CreateQueueResponse, error) {
	log.Debug().
		Str("component", "proxy").
		Msgf("PROXY CreateQueue: %+v to the leader node: %s", req, host)

	client, err := p.getClient(host)
	if err != nil {
		log.Error().
			Str("component", "proxy").
			Err(err).
			Msgf("Failed to get gRPC client for host: %s", host)
		return &pb.CreateQueueResponse{Success: false}, err
	}
	return client.CreateQueue(ctx, req)
}

func (p *GRPCProxy) UpdateQueue(ctx context.Context, host string, req *pb.UpdateQueueRequest) (*pb.UpdateQueueResponse, error) {
	log.Debug().
		Str("component", "proxy").
		Msgf("PROXY UpdateQueue: %+v to the leader node: %s", req, host)

	client, err := p.getClient(host)
	if err != nil {
		log.Error().
			Str("component", "proxy").
			Err(err).
			Msgf("Failed to get gRPC client for host: %s", host)
		return &pb.UpdateQueueResponse{Success: false}, err
	}
	return client.UpdateQueue(ctx, req)
}

func (p *GRPCProxy) DeleteQueue(ctx context.Context, host string, req *pb.DeleteQueueRequest) (*pb.DeleteQueueResponse, error) {
	log.Debug().
		Str("component", "proxy").
		Msgf("PROXY DeleteQueue: %+v to the leader node: %s", req, host)

	client, err := p.getClient(host)
	if err != nil {
		log.Error().
			Str("component", "proxy").
			Err(err).
			Msgf("Failed to get gRPC client for host: %s", host)
		return &pb.DeleteQueueResponse{Success: false}, err
	}
	return client.DeleteQueue(ctx, req)
}

func (p *GRPCProxy) Enqueue(ctx context.Context, host string, req *pb.EnqueueRequest) (*pb.EnqueueResponse, error) {
	log.Debug().
		Str("component", "proxy").
		Msgf("PROXY Enqueue: %+v to the leader node: %s", req, host)

	client, err := p.getClient(host)
	if err != nil {
		log.Error().
			Str("component", "proxy").
			Err(err).
			Msgf("Failed to get gRPC client for host: %s", host)
		return &pb.EnqueueResponse{Success: false}, err
	}
	return client.Enqueue(ctx, req)
}

func (p *GRPCProxy) Dequeue(ctx context.Context, host string, req *pb.DequeueRequest) (*pb.DequeueResponse, error) {
	log.Debug().
		Str("component", "proxy").
		Msgf("PROXY Dequeue: %+v to the leader node: %s", req, host)

	client, err := p.getClient(host)
	if err != nil {
		log.Error().
			Str("component", "proxy").
			Err(err).
			Msgf("Failed to get gRPC client for host: %s", host)
		return &pb.DequeueResponse{Success: false}, err
	}
	return client.Dequeue(ctx, req)
}

func (p *GRPCProxy) Get(ctx context.Context, host string, req *pb.GetRequest) (*pb.GetResponse, error) {
	log.Debug().
		Str("component", "proxy").
		Msgf("PROXY Get: %+v to the leader node: %s", req, host)

	client, err := p.getClient(host)
	if err != nil {
		log.Error().
			Str("component", "proxy").
			Err(err).
			Msgf("Failed to get gRPC client for host: %s", host)
		return &pb.GetResponse{Success: false}, err
	}
	return client.Get(ctx, req)
}

func (p *GRPCProxy) Delete(ctx context.Context, host string, req *pb.DeleteRequest) (*pb.DeleteResponse, error) {
	log.Debug().Str("component", "proxy").Msgf("PROXY Delete: %+v to the leader node: %s", req, host)

	client, err := p.getClient(host)
	if err != nil {
		log.Error().
			Str("component", "proxy").
			Err(err).
			Msgf("Failed to get gRPC client for host: %s", host)
		return &pb.DeleteResponse{Success: false}, err
	}
	return client.Delete(ctx, req)
}

func (p *GRPCProxy) Ack(ctx context.Context, host string, req *pb.AckRequest) (*pb.AckResponse, error) {
	log.Debug().Str("component", "proxy").Msgf("PROXY Ack: %+v to the leader node: %s", req, host)

	client, err := p.getClient(host)
	if err != nil {
		log.Error().
			Str("component", "proxy").
			Err(err).
			Msgf("Failed to get gRPC client for host: %s", host)
		return &pb.AckResponse{Success: false}, err
	}
	return client.Ack(ctx, req)
}

func (p *GRPCProxy) Nack(ctx context.Context, host string, req *pb.NackRequest) (*pb.NackResponse, error) {
	log.Debug().Str("component", "proxy").Msgf("PROXY Nack: %+v to the leader node: %s", req, host)

	client, err := p.getClient(host)
	if err != nil {
		log.Error().
			Str("component", "proxy").
			Err(err).
			Msgf("Failed to get gRPC client for host: %s", host)
		return &pb.NackResponse{Success: false}, err
	}
	return client.Nack(ctx, req)
}

func (p *GRPCProxy) Touch(ctx context.Context, host string, req *pb.TouchRequest) (*pb.TouchResponse, error) {
	log.Debug().Str("component", "proxy").Msgf("PROXY Touch: %+v to the leader node: %s", req, host)

	client, err := p.getClient(host)
	if err != nil {
		log.Error().
			Str("component", "proxy").
			Err(err).
			Msgf("Failed to get gRPC client for host: %s", host)
		return &pb.TouchResponse{Success: false}, err
	}
	return client.Touch(ctx, req)
}

func (p *GRPCProxy) UpdatePriority(ctx context.Context, host string, req *pb.UpdatePriorityRequest) (*pb.UpdatePriorityResponse, error) {
	log.Debug().
		Str("component", "proxy").
		Msgf("PROXY UpdatePriority: %+v to the leader node: %s", req, host)

	client, err := p.getClient(host)
	if err != nil {
		log.Error().
			Str("component", "proxy").
			Err(err).
			Msgf("Failed to get gRPC client for host: %s", host)
		return &pb.UpdatePriorityResponse{Success: false}, err
	}
	return client.UpdatePriority(ctx, req)
}

func (p *GRPCProxy) EnqueueStream(inStream pb.DOQ_EnqueueStreamServer, host string) error {
	log.Debug().Str("component", "proxy").Msgf("PROXY EnqueueStream to the leader node: %s", host)

	client, err := p.getClient(host)
	if err != nil {
		log.Error().
			Str("component", "proxy").
			Err(err).
			Msgf("Failed to get gRPC client for host: %s", host)
		return err
	}

	outStream, err := client.EnqueueStream(inStream.Context())
	if err != nil {
		log.Error().
			Str("component", "proxy").
			Msgf("Failed to open stream: %v", err)
		return err
	}

	for {
		req, err := inStream.Recv()
		if err == io.EOF {
			return nil
		}

		if err != nil {
			return err
		}

		// Send the message to the queue
		if err := outStream.Send(req); err != nil {
			log.Error().Str("component", "proxy").Msgf("Failed to send message: %v", err)
			return err
		}

		// Receive the acknowledgment from the server
		message, err := outStream.Recv()
		if err != nil {
			log.Error().Str("component", "proxy").Msgf("Failed to receive acknowledgment: %v", err)
			return err
		}

		err = inStream.Send(&pb.EnqueueResponse{
			Success:  true,
			Id:       message.Id,
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

func (p *GRPCProxy) DequeueStream(outStream pb.DOQ_DequeueStreamServer, host string) error {
	log.Debug().Str("component", "proxy").Msgf("PROXY DequeueStream to the leader node: %s", host)

	client, err := p.getClient(host)
	if err != nil {
		log.Error().
			Str("component", "proxy").
			Err(err).
			Msgf("Failed to get gRPC client for host: %s", host)
		return err
	}

	inStream, err := client.DequeueStream(outStream.Context())
	if err != nil {
		log.Error().Str("component", "proxy").Msgf("Failed to open stream: %v", err)
		return err
	}

	msg, err := outStream.Recv()
	if err != nil {
		log.Error().Str("component", "proxy").Msgf("Failed to receive message: %v", err)
		return err
	}

	err = inStream.Send(msg)
	if err != nil {
		log.Error().Str("component", "proxy").Msgf("Failed to open stream: %v", err)
		return err
	}

	// Consume messages from the stream
	for {
		select {
		case <-outStream.Context().Done():
			log.Info().Str("component", "proxy").Msg("Client closed the connection")
			return nil
		case <-inStream.Context().Done():
			return nil
		default:
			msg, err := inStream.Recv()
			if err != nil {
				log.Warn().Str("component", "proxy").Msgf("Failed to receive message: %v", err)
				time.Sleep(50 * time.Millisecond)
				continue
			}

			if err := outStream.Send(msg); err != nil {
				log.Error().Str("component", "proxy").Msgf("Failed to send message: %v", err)
				return err
			}

			err = inStream.Send(&pb.DequeueRequest{})

			if err != nil {
				log.Error().Str("component", "proxy").Msgf("Failed to send message: %v", err)
				return err
			}
		}
	}
}
