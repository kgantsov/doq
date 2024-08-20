package raft

import (
	"fmt"
	"hash/crc32"
	"os"
	"sort"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
	"github.com/kgantsov/doq/pkg/queue"
	"github.com/rs/zerolog/log"
)

type Node struct {
	id       string
	address  string
	raftPort string

	Raft         *raft.Raft
	QueueManager *queue.QueueManager
	leader       string

	peers []string

	db *badger.DB
}

func NewNode(db *badger.DB, id, address, raftPort string, peers []string) *Node {
	return &Node{
		id:       id,
		address:  address,
		raftPort: raftPort,
		peers:    peers,
		db:       db,
	}
}

func (node *Node) Initialize() {
	nodes := make([]string, 0)
	nodes = append(nodes, node.id)
	for _, peer := range node.peers {
		nodes = append(nodes, peer)
	}
	nodes = sort.StringSlice(nodes)

	nodeID := node.id
	numNodes := len(nodes)

	nodeIndex := int(crc32.ChecksumIEEE([]byte(nodeID)) % uint32(numNodes))
	leader := nodes[nodeIndex]
	replica1 := nodes[(nodeIndex+1)%numNodes]
	replica2 := nodes[(nodeIndex+2)%numNodes]
	replicas := []string{replica1, replica2}

	queueManager := queue.NewQueueManager(node.db)
	raftDir := fmt.Sprintf("raft/%s", node.id)
	os.MkdirAll(raftDir, 0700)

	raftPort := fmt.Sprintf("%s:%s", node.id, node.raftPort)

	log.Debug().Msgf("=====> Node %s listens on: %s IS LEADER: %t", node.id, raftPort, leader == node.id)

	raftNode, err := createRaftNode(nodeID, raftDir, raftPort, queueManager, leader == node.id)
	if err != nil {
		log.Fatal().Msgf("failed to create raft node: %s", err.Error())
	}

	node.Raft = raftNode
	node.QueueManager = queueManager

	configFuture := node.Raft.GetConfiguration()
	if err := configFuture.Error(); err != nil {
		log.Info().Msgf("failed to get raft configuration: %v", err)
	}

	if len(configFuture.Configuration().Servers) == 1 {
		if leader == node.id {
			for _, replica := range replicas {
				peerID := replica
				peerAddr := fmt.Sprintf("%s:%s", replica, node.raftPort)
				// peerAddr := replica

				log.Debug().Msgf("=====> JOIN Node %s adding peer %s %s", node.id, raft.ServerID(peerID), raft.ServerAddress(peerAddr))

				err := node.Join(peerID, peerAddr)
				if err != nil {
					log.Warn().Msgf("Error adding peer %s: %s", peerAddr, err)
				} else {
					log.Info().Msgf("Successfully added peer %s", peerAddr)
				}
			}
		}
	}

	go node.monitorLeadership()

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

func (n *Node) Leader() string {
	return n.leader
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

func createRaftNode(nodeID, raftDir, raftPort string, queueManager *queue.QueueManager, leader bool) (*raft.Raft, error) {
	config := raft.DefaultConfig()
	config.LocalID = raft.ServerID(nodeID)

	bindAddr := raftPort
	transport, err := raft.NewTCPTransport(bindAddr, nil, 3, 10*time.Second, os.Stderr)
	if err != nil {
		return nil, err
	}

	snapshots, err := raft.NewFileSnapshotStore(raftDir, 1, os.Stderr)
	if err != nil {
		return nil, err
	}

	logStore, err := raftboltdb.NewBoltStore(fmt.Sprintf("%s/raft-log.bolt", raftDir))
	if err != nil {
		return nil, err
	}

	stableStore, err := raftboltdb.NewBoltStore(fmt.Sprintf("%s/raft-stable.bolt", raftDir))
	if err != nil {
		return nil, err
	}

	fsm := &FSM{queueManager: queueManager, NodeID: nodeID}

	r, err := raft.NewRaft(config, fsm, logStore, stableStore, snapshots, transport)
	if err != nil {
		return nil, err
	}

	// configuration := raft.Configuration{Servers: servers}
	// r.BootstrapCluster(configuration)
	// time.Sleep(10 * time.Second)

	// log.Debug().Msgf("NODE %s %s is a leader: %t", nodeID, bindAddr, leader)
	if leader {
		configFuture := r.GetConfiguration()
		if err := configFuture.Error(); err != nil {
			log.Info().Msgf("failed to get raft configuration: %v", err)
		}

		if len(configFuture.Configuration().Servers) == 0 {
			servers := make([]raft.Server, 0)

			servers = append(servers, raft.Server{
				ID: config.LocalID,
				// Address: transport.LocalAddr(),
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
			// configuration := raft.Configuration{Servers: configFuture.Configuration().Servers}
			// r.BootstrapCluster(configuration)
			// time.Sleep(10 * time.Second)
		}
	}

	return r, nil
}
