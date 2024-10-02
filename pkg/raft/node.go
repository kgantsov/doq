package raft

import (
	"fmt"
	"net/url"
	"os"
	"sort"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/dgraph-io/badger/v4"
	"github.com/hashicorp/raft"
	badgerstore "github.com/kgantsov/doq/pkg/badger-store"
	"github.com/kgantsov/doq/pkg/config"
	"github.com/kgantsov/doq/pkg/queue"
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
	raftDir string

	prometheusRegistry prometheus.Registerer
}

func NewNode(db *badger.DB, raftDir string, cfg *config.Config, peers []string) *Node {
	node := &Node{
		cfg:            cfg,
		id:             cfg.Cluster.NodeID,
		address:        cfg.Http.Port,
		raftAddr:       cfg.Raft.Address,
		peers:          peers,
		db:             db,
		raftDir:        raftDir,
		leaderChangeFn: func(bool) {},
	}

	if cfg.Prometheus.Enabled {
		node.prometheusRegistry = prometheus.NewRegistry()
	}

	return node
}

func (node *Node) PrometheusRegistry() prometheus.Registerer {
	return node.prometheusRegistry
}

func (node *Node) SetLeaderChangeFunc(leaderChangeFn func(bool)) {
	node.leaderChangeFn = leaderChangeFn
}

func (node *Node) Initialize() {
	nodes := make([]string, 0)
	nodes = append(nodes, node.id)
	for _, peer := range node.peers {
		nodes = append(nodes, peer)
	}
	nodes = sort.StringSlice(nodes)

	nodeID := node.id

	log.Debug().Msgf("=====> TEST Initialize %+v", nodes)

	var prometheusMetrics *queue.PrometheusMetrics
	if node.cfg.Prometheus.Enabled {
		prometheusMetrics = queue.NewPrometheusMetrics(node.PrometheusRegistry(), "queues")
	}
	queueManager := queue.NewQueueManager(node.db, node.cfg, prometheusMetrics)

	os.MkdirAll(node.raftDir, 0700)

	raftNode, err := node.createRaftNode(nodeID, node.raftDir, node.raftAddr, queueManager)
	if err != nil {
		log.Fatal().Msgf("failed to create raft node: '%s' %s", node.raftAddr, err.Error())
	}

	node.Raft = raftNode
	node.QueueManager = queueManager

	idGenerator, err := snowflake.NewNode(1)
	if err != nil {
		log.Warn().Err(err).Msg("failed to create snowflake node")
	}

	node.idGenerator = idGenerator

	go node.monitorLeadership()
	go node.ListenToLeaderChanges()

}

func (node *Node) GenerateID() uint64 {
	return uint64(node.idGenerator.Generate().Int64())
}

func (node *Node) InitIDGenerator() error {
	time.Sleep(2 * time.Second)
	configFuture := node.Raft.GetConfiguration()
	if err := configFuture.Error(); err != nil {
		log.Info().Msgf("failed to get raft configuration: %v", err)
		return err
	}

	index := -1
	for i, srv := range configFuture.Configuration().Servers {
		if srv.ID == raft.ServerID(node.id) {
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

	node.idGenerator = idGenerator

	return nil
}

func (node *Node) monitorLeadership() {
	log.Debug().Msgf("Node %s Monitoring leadership for node %s", node.id, node.id)

	for {
		leader := string(node.Raft.Leader())
		if leader != node.leader {
			node.leader = leader
			log.Printf("Node %s leader is now %s", node.id, leader)
		}
		time.Sleep(1 * time.Second)
	}
}

func (node *Node) ListenToLeaderChanges() {
	for isLeader := range node.Raft.LeaderCh() {
		node.leaderChangeFn(isLeader)
	}
}

func (n *Node) Leader() string {
	u, _ := url.ParseRequestURI(fmt.Sprintf("http://%s", n.leader))

	return fmt.Sprintf("http://%s:8000", u.Hostname())
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

func (n *Node) createRaftNode(nodeID, raftDir, raftPort string, queueManager *queue.QueueManager) (*raft.Raft, error) {
	config := raft.DefaultConfig()
	config.SnapshotInterval = 120 * time.Second
	config.SnapshotThreshold = 8192
	config.LocalID = raft.ServerID(nodeID)
	config.LogLevel = "DEBUG"

	bindAddr := raftPort
	transport, err := raft.NewTCPTransport(bindAddr, nil, 3, 10*time.Second, os.Stderr)
	if err != nil {
		log.Warn().Msgf("failed to create transport: %s", err)
		return nil, err
	}

	var logStore raft.LogStore
	var stableStore raft.StableStore

	badgerDB, err := badgerstore.New(
		n.db,
		badgerstore.Options{},
	)
	if err != nil {
		log.Warn().Msgf("failed to create store: %s", err)
		return nil, fmt.Errorf("new store: %s", err)
	}
	logStore = badgerDB
	stableStore = badgerDB

	snapshots, err := raft.NewFileSnapshotStore(raftDir, 1, os.Stderr)
	if err != nil {
		log.Warn().Msgf("failed to create snapshot store: %s", err)
		return nil, err
	}

	fsm := &FSM{queueManager: queueManager, NodeID: nodeID, db: n.db, config: n.cfg}

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

		servers = append(servers, raft.Server{
			ID:      config.LocalID,
			Address: raft.ServerAddress(bindAddr),
		})

		log.Info().Msgf("BootstrapCluster %s joining peers: %v", nodeID, servers)

		configuration := raft.Configuration{Servers: servers}
		r.BootstrapCluster(configuration)
		time.Sleep(2 * time.Second)

		for isLeader := range r.LeaderCh() {
			if isLeader {
				log.Info().Msgf("Node %s has become a leader", nodeID)
			} else {
				log.Info().Msgf("Node %s lost leadership", nodeID)
			}
			break
		}
	} else {
		log.Info().Msgf("Already bootstraped %s %v", nodeID, configFuture.Configuration().Servers)
	}

	return r, nil
}
