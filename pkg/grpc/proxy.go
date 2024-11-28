package grpc

import (
	"context"
	"fmt"

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
