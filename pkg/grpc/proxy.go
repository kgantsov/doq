package grpc

import (
	"context"
	"fmt"
	"io"

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

func (p *GRPCProxy) initClient(leader string) {
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
}

func (p *GRPCProxy) CreateQueue(ctx context.Context, host string, req *pb.CreateQueueRequest) (*pb.CreateQueueResponse, error) {
	log.Debug().Msgf("PROXY CreateQueue: %+v to the leader node: %s", req, host)

	if p.leader != host {
		p.initClient(host)
	}
	return p.client.CreateQueue(ctx, req)
}

func (p *GRPCProxy) DeleteQueue(ctx context.Context, host string, req *pb.DeleteQueueRequest) (*pb.DeleteQueueResponse, error) {
	log.Debug().Msgf("PROXY DeleteQueue: %+v to the leader node: %s", req, host)

	if p.leader != host {
		p.initClient(host)
	}
	return p.client.DeleteQueue(ctx, req)
}

func (p *GRPCProxy) Enqueue(ctx context.Context, host string, req *pb.EnqueueRequest) (*pb.EnqueueResponse, error) {
	log.Debug().Msgf("PROXY Enqueue: %+v to the leader node: %s", req, host)

	if p.leader != host {
		p.initClient(host)
	}
	return p.client.Enqueue(ctx, req)
}

func (p *GRPCProxy) Dequeue(ctx context.Context, host string, req *pb.DequeueRequest) (*pb.DequeueResponse, error) {
	log.Debug().Msgf("PROXY Dequeue: %+v to the leader node: %s", req, host)

	if p.leader != host {
		p.initClient(host)
	}
	return p.client.Dequeue(ctx, req)
}

func (p *GRPCProxy) Ack(ctx context.Context, host string, req *pb.AckRequest) (*pb.AckResponse, error) {
	log.Debug().Msgf("PROXY Ack: %+v to the leader node: %s", req, host)

	if p.leader != host {
		p.initClient(host)
	}
	return p.client.Ack(ctx, req)
}

func (p *GRPCProxy) Nack(ctx context.Context, host string, req *pb.NackRequest) (*pb.NackResponse, error) {
	log.Debug().Msgf("PROXY Nack: %+v to the leader node: %s", req, host)

	if p.leader != host {
		p.initClient(host)
	}
	return p.client.Nack(ctx, req)
}

func (p *GRPCProxy) UpdatePriority(ctx context.Context, host string, req *pb.UpdatePriorityRequest) (*pb.UpdatePriorityResponse, error) {
	log.Debug().Msgf("PROXY UpdatePriority: %+v to the leader node: %s", req, host)

	if p.leader != host {
		p.initClient(host)
	}
	return p.client.UpdatePriority(ctx, req)
}

func (p *GRPCProxy) EnqueueStream(inStream pb.DOQ_EnqueueStreamServer, host string) error {
	log.Debug().Msgf("PROXY EnqueueStream to the leader node: %s", host)

	if p.leader != host {
		p.initClient(host)
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

func (p *GRPCProxy) DequeueStream(req *pb.DequeueRequest, outStream pb.DOQ_DequeueStreamServer, host string) error {
	log.Debug().Msgf("PROXY DequeueStream to the leader node: %s", host)

	if p.leader != host {
		p.initClient(host)
	}

	// Open a stream to receive messages from the queue
	inStream, err := p.client.DequeueStream(outStream.Context(), &pb.DequeueRequest{
		QueueName: req.QueueName,
		Ack:       false,
	})
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
				log.Error().Msgf("Failed to receive message: %v", err)
				return err
			}

			if err := outStream.Send(msg); err != nil {
				log.Error().Msgf("Failed to send message: %v", err)
				return err
			}
		}
	}
}
