package raft

import (
	"net"
	"sync"

	"github.com/rs/zerolog/log"
)

type LeaderConfig struct {
	Id             string
	RaftAddr       string
	GrpcAddr       string
	leaderGrpcAddr string

	mu sync.RWMutex
}

func NewLeaderConfig(nodeID, raftAddr, grpcAddr string) *LeaderConfig {
	return &LeaderConfig{
		Id:       nodeID,
		RaftAddr: raftAddr,
		GrpcAddr: grpcAddr,

		mu: sync.RWMutex{},
	}
}

func (c *LeaderConfig) Set(nodeID, raftAddr, grpcAddr string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.Id = nodeID
	c.RaftAddr = raftAddr
	c.GrpcAddr = grpcAddr

	leaderHostname, _, err := net.SplitHostPort(raftAddr)
	if err != nil {
		return err
	}
	_, grpcPortOnly, err := net.SplitHostPort(grpcAddr)
	if err != nil {
		return err
	}

	c.leaderGrpcAddr = net.JoinHostPort(leaderHostname, grpcPortOnly)
	log.Info().
		Str("node_id", c.Id).
		Str("raft_addr", c.RaftAddr).
		Str("grpc_addr", c.GrpcAddr).
		Str("leader_grpc_addr", c.leaderGrpcAddr).
		Msg("Leader config updated")

	return nil
}

func (c *LeaderConfig) GetLeaderGrpcAddress() string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.leaderGrpcAddr
}
