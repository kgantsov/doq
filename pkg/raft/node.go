package raft

import (
	"fmt"
	"net"
	"os"
	"sort"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"

	"github.com/dgraph-io/badger/v4"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/raft"
	"github.com/kgantsov/doq/pkg/config"
	"github.com/kgantsov/doq/pkg/entity"
	"github.com/kgantsov/doq/pkg/grpc"
	"github.com/kgantsov/doq/pkg/logger"
	"github.com/kgantsov/doq/pkg/metrics"
	"github.com/kgantsov/doq/pkg/queue"

	"github.com/kgantsov/doq/pkg/storage"
	raftstore "github.com/kgantsov/raft-badgerstore"
	"github.com/rs/zerolog/log"
)

type Node struct {
	cfg *config.Config

	idGenerator *snowflake.Node

	id       string
	address  string
	raftAddr string

	Raft         *raft.Raft
	QueueManager *queue.QueueManager
	leader       string

	leaderChangeFn func(bool)

	peers []string

	db      *badger.DB
	raftDB  *badger.DB
	raftDir string

	prometheusRegistry prometheus.Registerer

	leaderConfig *LeaderConfig

	proxy *grpc.GRPCProxy
}

func NewNode(db *badger.DB, raftDB *badger.DB, raftDir string, cfg *config.Config, peers []string) *Node {
	node := &Node{
		cfg:            cfg,
		id:             cfg.Cluster.NodeID,
		address:        cfg.Http.Port,
		raftAddr:       cfg.Raft.Address,
		peers:          peers,
		db:             db,
		raftDB:         raftDB,
		raftDir:        raftDir,
		leaderChangeFn: func(bool) {},

		leaderConfig: NewLeaderConfig(
			cfg.Cluster.NodeID,
			cfg.Raft.Address,
			cfg.Grpc.Address,
		),

		proxy: grpc.NewGRPCProxy(),
	}

	if cfg.Prometheus.Enabled {
		node.prometheusRegistry = prometheus.NewRegistry()
	}

	return node
}

func (n *Node) PrometheusRegistry() prometheus.Registerer {
	return n.prometheusRegistry
}

func (n *Node) SetLeaderChangeFunc(leaderChangeFn func(bool)) {
	n.leaderChangeFn = leaderChangeFn
}

func (n *Node) Initialize() {
	nodes := make([]string, 0)
	nodes = append(nodes, n.id)
	for _, peer := range n.peers {
		nodes = append(nodes, peer)
	}
	sort.Strings(nodes)

	nodeID := n.id

	log.Debug().Msgf("=====> TEST Initialize %+v", nodes)

	var prometheusMetrics *metrics.PrometheusMetrics
	if n.cfg.Prometheus.Enabled {
		promRegistry := n.PrometheusRegistry()
		prometheusMetrics = metrics.NewPrometheusMetrics(promRegistry, "doq", "queues")
		promRegistry.Register(collectors.NewGoCollector())
	}
	store := storage.NewBadgerStore(n.db)
	queueManager := queue.NewQueueManager(store, n.cfg, prometheusMetrics)

	os.MkdirAll(n.raftDir, 0700)

	idGenerator, err := snowflake.NewNode(1)
	if err != nil {
		log.Warn().Err(err).Msg("failed to create snowflake node")
	}

	n.idGenerator = idGenerator

	raftNode, err := n.createRaftNode(nodeID, n.raftDir, n.raftAddr, queueManager)
	if err != nil {
		log.Fatal().Msgf("failed to create raft node: '%s' %s", n.raftAddr, err.Error())
	}

	n.Raft = raftNode
	n.QueueManager = queueManager

	// go n.monitorLeadership()
	go n.ListenToLeaderChanges()
}

func (n *Node) GenerateID() uint64 {
	return uint64(n.idGenerator.Generate().Int64())
}

func (n *Node) InitIDGenerator() error {
	time.Sleep(2 * time.Second)
	configFuture := n.Raft.GetConfiguration()
	if err := configFuture.Error(); err != nil {
		log.Info().Msgf("failed to get raft configuration: %v", err)
		return err
	}

	servers := configFuture.Configuration().Servers
	sort.Slice(servers, func(i, j int) bool {
		return servers[i].ID < servers[j].ID
	})

	index := -1
	for i, srv := range servers {
		if srv.ID == raft.ServerID(n.id) {
			index = i
			break
		}
	}

	log.Info().Msgf("Server configuration: %v Node indes: %d", configFuture.Configuration().Servers, index)

	// Create a new snowflake Node with a Node number
	idGenerator, err := snowflake.NewNode(int64(index + 1))
	if err != nil {
		log.Warn().Err(err).Msg("failed to create snowflake node")
		return err
	}

	n.idGenerator = idGenerator

	return nil
}

func (n *Node) ListenToLeaderChanges() {
	for isLeader := range n.Raft.LeaderCh() {
		if isLeader {
			log.Info().Msgf("Node %s has become a leader", n.id)
			// Notify the leader configuration
			n.NotifyLeaderConfiguration()
		} else {
			log.Info().Msgf("Node %s lost leadership", n.id)
		}
		n.leaderChangeFn(isLeader)
	}
}

func (n *Node) IsLeader() bool {
	return n.Raft.State() == raft.Leader
}

func (n *Node) Join(nodeID, addr string) error {
	log.Info().Msgf("received join request for remote node %s at %s", nodeID, addr)

	configFuture := n.Raft.GetConfiguration()
	if err := configFuture.Error(); err != nil {
		log.Info().Msgf("failed to get raft configuration: %v", err)
		return err
	}

	for _, srv := range configFuture.Configuration().Servers {
		// If a node already exists with either the joining node's ID or address,
		// that node may need to be removed from the config first.
		// log.Debug().Msgf("=====> @@@@@ checking existing node %s at %s == %t %t", srv.ID, srv.Address, srv.ID == raft.ServerID(nodeID), srv.Address == raft.ServerAddress(addr))
		if srv.ID == raft.ServerID(nodeID) || srv.Address == raft.ServerAddress(addr) {
			// However if *both* the ID and the address are the same, then nothing -- not even
			// a join operation -- is needed.
			if srv.Address == raft.ServerAddress(addr) && srv.ID == raft.ServerID(nodeID) {
				log.Info().Msgf("node %s at %s already member of cluster, ignoring join request", nodeID, addr)
				return nil
			}

			future := n.Raft.RemoveServer(srv.ID, 0, 0)
			// log.Debug().Msgf("=====> !!!! removing existing node %s at %s", srv.ID, srv.Address)
			if err := future.Error(); err != nil {
				// log.Warn().Msgf("error removing existing node %s at %s: %s", nodeID, addr, err)
				return fmt.Errorf("error removing existing node %s at %s: %s", nodeID, addr, err)
			}
		}
	}

	f := n.Raft.AddVoter(raft.ServerID(nodeID), raft.ServerAddress(addr), 0, 0)
	if f.Error() != nil {
		return f.Error()
	}
	log.Info().Msgf("node %s at %s joined successfully", nodeID, addr)
	return nil
}

func (n *Node) Leave(nodeID string) error {
	removeFuture := n.Raft.RemoveServer(raft.ServerID(nodeID), 0, 0)
	return removeFuture.Error()
}

func (n *Node) GetServers() ([]*entity.Server, error) {
	future := n.Raft.GetConfiguration()
	if err := future.Error(); err != nil {
		return nil, err
	}

	var servers []*entity.Server

	leaderAddr := n.leaderConfig.GetLeaderRaftAddress()

	for _, server := range future.Configuration().Servers {
		servers = append(servers, &entity.Server{
			Id:         string(server.ID),
			Addr:       string(server.Address),
			LeaderAddr: string(leaderAddr),
			IsLeader:   leaderAddr == string(server.Address),
			Suffrage:   server.Suffrage.String(),
		})
	}

	return servers, nil
}

func (n *Node) createRaftNode(nodeID, raftDir, raftPort string, queueManager *queue.QueueManager) (*raft.Raft, error) {
	config := raft.DefaultConfig()
	config.SnapshotInterval = 120 * time.Second
	config.SnapshotThreshold = 8192
	config.LocalID = raft.ServerID(nodeID)
	config.LogLevel = n.cfg.Logging.Level
	config.Logger = logger.NewZeroHCLLogger("raft", hclog.LevelFromString(n.cfg.Logging.Level))

	bindAddr := raftPort
	transport, err := raft.NewTCPTransportWithLogger(bindAddr, nil, 3, 10*time.Second, config.Logger)
	if err != nil {
		log.Warn().Msgf("failed to create transport: %s", err)
		return nil, err
	}

	var logStore raft.LogStore
	var stableStore raft.StableStore

	badgerDB, err := raftstore.New(
		n.db,
		raftstore.Options{},
	)
	if err != nil {
		log.Warn().Msgf("failed to create store: %s", err)
		return nil, fmt.Errorf("new store: %s", err)
	}
	logStore = badgerDB
	stableStore = badgerDB

	snapshots, err := raft.NewFileSnapshotStoreWithLogger(raftDir, 1, config.Logger)
	if err != nil {
		log.Warn().Msgf("failed to create snapshot store: %s", err)
		return nil, err
	}

	fsm := &FSM{
		queueManager: queueManager,
		NodeID:       nodeID,
		db:           n.db,
		config:       n.cfg,
		leaderConfig: n.leaderConfig,
	}

	r, err := raft.NewRaft(config, fsm, logStore, stableStore, snapshots, transport)
	if err != nil {
		log.Warn().Msgf("failed to create raft: %s", err)
		return nil, err
	}

	configFuture := r.GetConfiguration()
	if err := configFuture.Error(); err != nil {
		log.Info().Msgf("failed to get raft configuration: %v", err)
	}

	if len(configFuture.Configuration().Servers) == 0 {
		servers := make([]raft.Server, 0)

		host, _, err := net.SplitHostPort(string(config.LocalID))
		if err != nil {
			log.Warn().Msgf(
				"Error splitting host and port for config.LocalID: %s %v\n", config.LocalID, err,
			)
			host = "localhost"
		}

		_, raftPort, err := net.SplitHostPort(string(transport.LocalAddr()))
		if err != nil {
			log.Warn().Msgf(
				"Error splitting host and port for raftAddr: %s %v\n", transport.LocalAddr(), err,
			)
		}

		servers = append(servers, raft.Server{
			ID:      config.LocalID,
			Address: raft.ServerAddress(fmt.Sprintf("%s:%s", host, raftPort)),
		})

		log.Info().Msgf("BootstrapCluster %s joining peers: %v", nodeID, servers)

		configuration := raft.Configuration{Servers: servers}
		r.BootstrapCluster(configuration)
		// time.Sleep(2 * time.Second)
	} else {
		log.Info().Msgf("Already bootstraped %s %v", nodeID, configFuture.Configuration().Servers)
	}

	return r, nil
}
