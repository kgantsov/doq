package raft

import (
	"net"
	"sync"

	"github.com/rs/zerolog/log"
)

type LeaderConfig struct {
	Id             string
	HttpAddr       string
	RaftAddr       string
	GrpcAddr       string
	leaderHttpAddr string
	leaderGrpcAddr string
	leaderRaftAddr string

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

func (c *LeaderConfig) Set(nodeID, raftAddr, grpcAddr, httpAddr string) error {
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
	_, httpPortOnly, err := net.SplitHostPort(httpAddr)
	if err != nil {
		return err
	}
	_, raftPortOnly, err := net.SplitHostPort(raftAddr)
	if err != nil {
		return err
	}

	c.leaderGrpcAddr = net.JoinHostPort(leaderHostname, grpcPortOnly)
	c.leaderHttpAddr = net.JoinHostPort(leaderHostname, httpPortOnly)
	c.leaderRaftAddr = net.JoinHostPort(leaderHostname, raftPortOnly)
	log.Info().
		Str("node_id", c.Id).
		Str("raft_addr", c.RaftAddr).
		Str("grpc_addr", c.GrpcAddr).
		Str("leader_grpc_addr", c.leaderGrpcAddr).
		Str("leader_http_addr", c.leaderHttpAddr).
		Msg("Leader config updated")

	return nil
}

func (c *LeaderConfig) GetLeaderGrpcAddress() string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.leaderGrpcAddr
}

func (c *LeaderConfig) GetLeaderHttpAddress() string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.leaderHttpAddr
}

func (c *LeaderConfig) GetLeaderRaftAddress() string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.leaderRaftAddr
}
