package grpc

import (
	"context"
	"fmt"
	"io"
	"time"

	pb "github.com/kgantsov/doq/pkg/proto"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
)

type GRPCProxy struct {
	client pb.DOQClient
	leader string
	port   int
}

func NewGRPCProxy(client pb.DOQClient, port int) *GRPCProxy {
	return &GRPCProxy{
		client: client,
		port:   port,
	}
}

func (p *GRPCProxy) initClient(leader string) error {
	if leader == "" {
		return fmt.Errorf("leader address is empty")
	}

	p.leader = leader

	host := leader

	if p.port != 0 {
		host = fmt.Sprintf("%s:%d", host, p.port)
	}

	conn, err := grpc.Dial(host, grpc.WithInsecure())
	if err != nil {
		log.Fatal().Msgf("Failed to connect: %v", err)
	}
	// defer conn.Close()

	p.client = pb.NewDOQClient(conn)

	return nil
}

func (p *GRPCProxy) CreateQueue(ctx context.Context, host string, req *pb.CreateQueueRequest) (*pb.CreateQueueResponse, error) {
	log.Debug().Msgf("PROXY CreateQueue: %+v to the leader node: %s", req, host)

	if p.leader != host || p.client == nil {
		if err := p.initClient(host); err != nil {
			return &pb.CreateQueueResponse{Success: false}, err
		}
	}
	return p.client.CreateQueue(ctx, req)
}

func (p *GRPCProxy) DeleteQueue(ctx context.Context, host string, req *pb.DeleteQueueRequest) (*pb.DeleteQueueResponse, error) {
	log.Debug().Msgf("PROXY DeleteQueue: %+v to the leader node: %s", req, host)

	if p.leader != host || p.client == nil {
		if err := p.initClient(host); err != nil {
			return &pb.DeleteQueueResponse{Success: false}, err
		}
	}
	return p.client.DeleteQueue(ctx, req)
}

func (p *GRPCProxy) Enqueue(ctx context.Context, host string, req *pb.EnqueueRequest) (*pb.EnqueueResponse, error) {
	log.Debug().Msgf("PROXY Enqueue: %+v to the leader node: %s", req, host)

	if p.leader != host || p.client == nil {
		if err := p.initClient(host); err != nil {
			return &pb.EnqueueResponse{Success: false}, err
		}
	}
	return p.client.Enqueue(ctx, req)
}

func (p *GRPCProxy) Dequeue(ctx context.Context, host string, req *pb.DequeueRequest) (*pb.DequeueResponse, error) {
	log.Debug().Msgf("PROXY Dequeue: %+v to the leader node: %s", req, host)

	if p.leader != host || p.client == nil {
		if err := p.initClient(host); err != nil {
			return &pb.DequeueResponse{Success: false}, err
		}
	}
	return p.client.Dequeue(ctx, req)
}

func (p *GRPCProxy) Get(ctx context.Context, host string, req *pb.GetRequest) (*pb.GetResponse, error) {
	log.Debug().Msgf("PROXY Get: %+v to the leader node: %s", req, host)

	if p.leader != host || p.client == nil {
		if err := p.initClient(host); err != nil {
			return &pb.GetResponse{Success: false}, err
		}
	}
	return p.client.Get(ctx, req)
}

func (p *GRPCProxy) Delete(ctx context.Context, host string, req *pb.DeleteRequest) (*pb.DeleteResponse, error) {
	log.Debug().Msgf("PROXY Delete: %+v to the leader node: %s", req, host)

	if p.leader != host || p.client == nil {
		if err := p.initClient(host); err != nil {
			return &pb.DeleteResponse{Success: false}, err
		}
	}
	return p.client.Delete(ctx, req)
}

func (p *GRPCProxy) Ack(ctx context.Context, host string, req *pb.AckRequest) (*pb.AckResponse, error) {
	log.Debug().Msgf("PROXY Ack: %+v to the leader node: %s", req, host)

	if p.leader != host || p.client == nil {
		if err := p.initClient(host); err != nil {
			return &pb.AckResponse{Success: false}, err
		}
	}
	return p.client.Ack(ctx, req)
}

func (p *GRPCProxy) Nack(ctx context.Context, host string, req *pb.NackRequest) (*pb.NackResponse, error) {
	log.Debug().Msgf("PROXY Nack: %+v to the leader node: %s", req, host)

	if p.leader != host || p.client == nil {
		if err := p.initClient(host); err != nil {
			return &pb.NackResponse{Success: false}, err
		}
	}
	return p.client.Nack(ctx, req)
}

func (p *GRPCProxy) UpdatePriority(ctx context.Context, host string, req *pb.UpdatePriorityRequest) (*pb.UpdatePriorityResponse, error) {
	log.Debug().Msgf("PROXY UpdatePriority: %+v to the leader node: %s", req, host)

	if p.leader != host || p.client == nil {
		if err := p.initClient(host); err != nil {
			return &pb.UpdatePriorityResponse{Success: false}, err
		}
	}
	return p.client.UpdatePriority(ctx, req)
}

func (p *GRPCProxy) EnqueueStream(inStream pb.DOQ_EnqueueStreamServer, host string) error {
	log.Debug().Msgf("PROXY EnqueueStream to the leader node: %s", host)

	if p.leader != host || p.client == nil {
		if err := p.initClient(host); err != nil {
			return err
		}
	}

	outStream, err := p.client.EnqueueStream(inStream.Context())
	if err != nil {
		log.Error().Msgf("Failed to open stream: %v", err)
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
			log.Error().Msgf("Failed to send message: %v", err)
			return err
		}

		// Receive the acknowledgment from the server
		message, err := outStream.Recv()
		if err != nil {
			log.Error().Msgf("Failed to receive acknowledgment: %v", err)
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
	log.Debug().Msgf("PROXY DequeueStream to the leader node: %s", host)

	if p.leader != host || p.client == nil {
		if err := p.initClient(host); err != nil {
			return err
		}
	}

	inStream, err := p.client.DequeueStream(outStream.Context())
	if err != nil {
		log.Error().Msgf("Failed to open stream: %v", err)
		return err
	}

	msg, err := outStream.Recv()
	if err != nil {
		log.Error().Msgf("Failed to receive message: %v", err)
		return err
	}

	err = inStream.Send(msg)
	if err != nil {
		log.Error().Msgf("Failed to open stream: %v", err)
		return err
	}

	// Consume messages from the stream
	for {
		select {
		case <-outStream.Context().Done():
			log.Info().Msg("Client closed the connection")
			return nil
		case <-inStream.Context().Done():
			return nil
		default:
			msg, err := inStream.Recv()
			if err != nil {
				log.Warn().Msgf("Failed to receive message: %v", err)
				time.Sleep(50 * time.Millisecond)
				continue
			}

			if err := outStream.Send(msg); err != nil {
				log.Error().Msgf("Failed to send message: %v", err)
				return err
			}

			err = inStream.Send(&pb.DequeueRequest{})

			if err != nil {
				log.Error().Msgf("Failed to send message: %v", err)
				return err
			}
		}
	}
}
