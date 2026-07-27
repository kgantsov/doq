package raft

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/kgantsov/doq/pkg/config"
	"github.com/kgantsov/doq/pkg/entity"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
)

func TestNodeSingleNode(t *testing.T) {
	tmpStoreDir, _ := os.MkdirTemp("", "db*")
	defer os.RemoveAll(tmpStoreDir)

	tmpRaftDir, _ := os.MkdirTemp("", "raft*")
	defer os.RemoveAll(tmpRaftDir)

	tmpStableStoreDir, _ := os.MkdirTemp("", "stable_store*")
	defer os.RemoveAll(tmpStableStoreDir)

	db, err := badger.Open(badger.DefaultOptions(tmpStoreDir))
	if err != nil {
		log.Fatal().Msg(err.Error())
	}
	raftDB, err := badger.Open(badger.DefaultOptions(tmpStableStoreDir))
	if err != nil {
		log.Fatal().Msg(err.Error())
	}

	cfg := &config.Config{
		Cluster: config.ClusterConfig{
			NodeID: "localhost",
		},
		Http: config.HttpConfig{
			Port: "9110",
		},
		Raft: config.RaftConfig{
			Address: "localhost:9111",
		},
		Queue: config.QueueConfig{
			AcknowledgementCheckInterval: 1,
			QueueStats:                   config.QueueStatsConfig{WindowSide: 10},
		},
	}

	n := NewNode(db, raftDB, tmpRaftDir, cfg, []string{})
	n.Initialize()

	// Simple way to ensure there is a leader.
	time.Sleep(1 * time.Second)
	err = n.InitIDGenerator()
	assert.Nil(t, err)

	id := n.GenerateID()
	assert.NotEqual(t, int64(0), id)

	assert.True(t, n.IsLeader())

	queues := n.GetQueues()
	assert.Equal(t, 0, len(queues))

	queueName := "test_queue_0"

	err = n.CreateQueue("delayed", queueName, entity.QueueSettings{})
	assert.Nil(t, err)

	queues = n.GetQueues()
	assert.Equal(t, 1, len(queues))

	queueInfo, err := n.GetQueueInfo(queueName)
	assert.Nil(t, err)
	assert.Equal(t, queueName, queueInfo.Name)
	assert.Equal(t, "delayed", queueInfo.Type)
	assert.Equal(t, int64(0), queueInfo.Ready)
	assert.Equal(t, int64(0), queueInfo.Unacked)
	assert.Equal(t, int64(0), queueInfo.Total)

	m1, err := n.Enqueue(queueName, 0, "default", 10, "message 1", nil)
	assert.Nil(t, err)
	assert.NotNil(t, m1)
	assert.Equal(t, "message 1", m1.Content)
	assert.Equal(t, int64(10), m1.Priority)

	queueInfo, err = n.GetQueueInfo(queueName)
	assert.Nil(t, err)
	assert.Equal(t, int64(1), queueInfo.Ready)
	assert.Equal(t, int64(0), queueInfo.Unacked)
	assert.Equal(t, int64(1), queueInfo.Total)

	m2, err := n.Enqueue(queueName, 0, "default", 5, "message 2", nil)
	assert.Nil(t, err)
	assert.NotNil(t, m2)
	assert.Equal(t, "message 2", m2.Content)
	assert.Equal(t, int64(5), m2.Priority)

	queueInfo, err = n.GetQueueInfo(queueName)
	assert.Nil(t, err)
	assert.Equal(t, int64(2), queueInfo.Ready)
	assert.Equal(t, int64(0), queueInfo.Unacked)
	assert.Equal(t, int64(2), queueInfo.Total)

	m3, err := n.Enqueue(queueName, 0, "default", 10, "message 3", map[string]string{"key": "value"})
	assert.Nil(t, err)
	assert.NotNil(t, m3)
	assert.Equal(t, "message 3", m3.Content)
	assert.Equal(t, int64(10), m3.Priority)
	assert.Equal(t, "value", m3.Metadata["key"])

	queueInfo, err = n.GetQueueInfo(queueName)
	assert.Nil(t, err)
	assert.Equal(t, int64(3), queueInfo.Ready)
	assert.Equal(t, int64(0), queueInfo.Unacked)
	assert.Equal(t, int64(3), queueInfo.Total)

	m, err := n.Get(queueName, m1.ID)
	assert.Nil(t, err)
	assert.Equal(t, m1.ID, m.ID)
	assert.Equal(t, m1.Content, m.Content)
	assert.Equal(t, m1.Priority, m.Priority)

	m, err = n.Dequeue(queueName, true)
	assert.Nil(t, err)
	assert.Equal(t, m2.ID, m.ID)
	assert.Equal(t, m2.Content, m.Content)
	assert.Equal(t, m2.Priority, m.Priority)

	queueInfo, err = n.GetQueueInfo(queueName)
	assert.Nil(t, err)
	assert.Equal(t, int64(2), queueInfo.Ready)
	assert.Equal(t, int64(0), queueInfo.Unacked)
	assert.Equal(t, int64(2), queueInfo.Total)

	m, err = n.Dequeue(queueName, true)
	assert.Nil(t, err)
	assert.Equal(t, m1.ID, m.ID)
	assert.Equal(t, m1.Content, m.Content)
	assert.Equal(t, m1.Priority, m.Priority)

	queueInfo, err = n.GetQueueInfo(queueName)
	assert.Nil(t, err)
	assert.Equal(t, int64(1), queueInfo.Ready)
	assert.Equal(t, int64(0), queueInfo.Unacked)
	assert.Equal(t, int64(1), queueInfo.Total)

	err = n.Delete(queueName, m3.ID)
	assert.Nil(t, err)

	m, err = n.Dequeue(queueName, false)
	assert.Nil(t, m)
	assert.Error(t, err)

	err = n.DeleteQueue(queueName)
	assert.Nil(t, err)

	queues = n.GetQueues()
	assert.Equal(t, 0, len(queues))
}

func TestNodeUpdateDeleteQueue(t *testing.T) {
	tmpStoreDir, _ := os.MkdirTemp("", "db*")
	defer os.RemoveAll(tmpStoreDir)

	tmpRaftDir, _ := os.MkdirTemp("", "raft*")
	defer os.RemoveAll(tmpRaftDir)

	tmpStableStoreDir, _ := os.MkdirTemp("", "stable_store*")
	defer os.RemoveAll(tmpStableStoreDir)

	db, err := badger.Open(badger.DefaultOptions(tmpStoreDir))
	if err != nil {
		log.Fatal().Msg(err.Error())
	}
	raftDB, err := badger.Open(badger.DefaultOptions(tmpStableStoreDir))
	if err != nil {
		log.Fatal().Msg(err.Error())
	}

	cfg := &config.Config{
		Cluster: config.ClusterConfig{
			NodeID: "localhost",
		},
		Http: config.HttpConfig{
			Port: "9120",
		},
		Raft: config.RaftConfig{
			Address: "localhost:9121",
		},
		Queue: config.QueueConfig{
			AcknowledgementCheckInterval: 1,
			QueueStats:                   config.QueueStatsConfig{WindowSide: 10},
		},
	}

	n := NewNode(db, raftDB, tmpRaftDir, cfg, []string{})
	n.Initialize()

	// Simple way to ensure there is a leader.
	time.Sleep(3 * time.Second)

	queueName := "test_queue_1"

	err = n.CreateQueue("fair", queueName, entity.QueueSettings{
		Strategy:   "WEIGHTED",
		MaxUnacked: 75,
		AckTimeout: 300,
	})
	assert.Nil(t, err)

	queue, err := n.GetQueueInfo(queueName)
	assert.Nil(t, err)
	assert.Equal(t, "WEIGHTED", queue.Settings.Strategy)
	assert.Equal(t, int(75), queue.Settings.MaxUnacked)
	assert.Equal(t, uint32(300), queue.Settings.AckTimeout)

	err = n.UpdateQueue(queueName, entity.QueueSettings{
		Strategy:   "WEIGHTED",
		MaxUnacked: 100,
		AckTimeout: 600,
	})
	assert.Nil(t, err)

	queue, err = n.GetQueueInfo(queueName)
	assert.Nil(t, err)
	assert.Equal(t, "WEIGHTED", queue.Settings.Strategy)
	assert.Equal(t, int(100), queue.Settings.MaxUnacked)
	assert.Equal(t, uint32(600), queue.Settings.AckTimeout)

	err = n.DeleteQueue(queueName)
	assert.Nil(t, err)

	queue, err = n.GetQueueInfo(queueName)
	assert.NotNil(t, err)
	assert.Nil(t, queue)

	err = n.DeleteQueue("non_existent_queue")
	assert.Error(t, err)
}

func TestNodeSingleNodeAck(t *testing.T) {
	tmpStoreDir, _ := os.MkdirTemp("", "db*")
	defer os.RemoveAll(tmpStoreDir)

	tmpRaftDir, _ := os.MkdirTemp("", "raft*")
	defer os.RemoveAll(tmpRaftDir)

	tmpStableStoreDir, _ := os.MkdirTemp("", "stable_store*")
	defer os.RemoveAll(tmpStableStoreDir)

	db, err := badger.Open(badger.DefaultOptions(tmpStoreDir))
	if err != nil {
		log.Fatal().Msg(err.Error())
	}
	raftDB, err := badger.Open(badger.DefaultOptions(tmpStableStoreDir))
	if err != nil {
		log.Fatal().Msg(err.Error())
	}

	cfg := &config.Config{
		Cluster: config.ClusterConfig{
			NodeID: "localhost",
		},
		Http: config.HttpConfig{
			Port: "9130",
		},
		Raft: config.RaftConfig{
			Address: "localhost:9131",
		},
		Queue: config.QueueConfig{
			AcknowledgementCheckInterval: 1,
			QueueStats:                   config.QueueStatsConfig{WindowSide: 10},
		},
	}

	n := NewNode(db, raftDB, tmpRaftDir, cfg, []string{})
	n.Initialize()

	queueName := "test_queue_2"

	// Simple way to ensure there is a leader.
	time.Sleep(3 * time.Second)
	err = n.CreateQueue("delayed", queueName, entity.QueueSettings{})
	assert.Nil(t, err)

	assert.True(t, n.IsLeader())

	m1, err := n.Enqueue(queueName, 12312, "default", 10, "message 1", nil)
	assert.Nil(t, err)
	assert.NotNil(t, m1)
	assert.Equal(t, uint64(12312), m1.ID)
	assert.Equal(t, "message 1", m1.Content)
	assert.Equal(t, int64(10), m1.Priority)

	m, err := n.Dequeue(queueName, false)
	assert.Nil(t, err)
	assert.Equal(t, m1.ID, m.ID)
	assert.Equal(t, uint64(12312), m.ID)
	assert.Equal(t, m1.Content, m.Content)
	assert.Equal(t, m1.Priority, m.Priority)

	err = n.Ack(queueName, m.ID)
	assert.Nil(t, err)

	err = n.DeleteQueue(queueName)
	assert.Nil(t, err)
}

func TestNodeSingleNodeNack(t *testing.T) {
	tmpStoreDir, _ := os.MkdirTemp("", "db*")
	defer os.RemoveAll(tmpStoreDir)

	tmpRaftDir, _ := os.MkdirTemp("", "raft*")
	defer os.RemoveAll(tmpRaftDir)

	tmpStableStoreDir, _ := os.MkdirTemp("", "stable_store*")
	defer os.RemoveAll(tmpStableStoreDir)

	db, err := badger.Open(badger.DefaultOptions(tmpStoreDir))
	if err != nil {
		log.Fatal().Msg(err.Error())
	}
	raftDB, err := badger.Open(badger.DefaultOptions(tmpStableStoreDir))
	if err != nil {
		log.Fatal().Msg(err.Error())
	}

	cfg := &config.Config{
		Cluster: config.ClusterConfig{
			NodeID: "localhost",
		},
		Http: config.HttpConfig{
			Port: "9150",
		},
		Raft: config.RaftConfig{
			Address: "localhost:9151",
		},
		Queue: config.QueueConfig{
			AcknowledgementCheckInterval: 1,
			QueueStats:                   config.QueueStatsConfig{WindowSide: 10},
		},
	}

	n := NewNode(db, raftDB, tmpRaftDir, cfg, []string{})
	n.Initialize()

	queueName := "test_queue_3"
	// Simple way to ensure there is a leader.
	time.Sleep(3 * time.Second)
	err = n.CreateQueue("delayed", queueName, entity.QueueSettings{})
	assert.Nil(t, err)

	assert.True(t, n.IsLeader())

	m1, err := n.Enqueue(queueName, 0, "default", 10, "message 1", nil)
	assert.Nil(t, err)
	assert.NotNil(t, m1)
	assert.Equal(t, "message 1", m1.Content)
	assert.Equal(t, int64(10), m1.Priority)

	m, err := n.Dequeue(queueName, false)
	assert.Nil(t, err)
	assert.Equal(t, m1.ID, m.ID)
	assert.Equal(t, m1.Content, m.Content)
	assert.Equal(t, m1.Priority, m.Priority)

	err = n.Nack(queueName, m.ID, 5, map[string]string{})
	assert.Nil(t, err)

	m, err = n.Dequeue(queueName, false)
	assert.Nil(t, err)
	assert.Equal(t, m1.ID, m.ID)
	assert.Equal(t, m1.Content, m.Content)
	assert.Equal(t, int64(5), m.Priority)

	err = n.DeleteQueue(queueName)
	assert.Nil(t, err)
}

func TestNodeSingleNodeUpdatePriority(t *testing.T) {
	tmpStoreDir, _ := os.MkdirTemp("", "db*")
	defer os.RemoveAll(tmpStoreDir)

	tmpRaftDir, _ := os.MkdirTemp("", "raft*")
	defer os.RemoveAll(tmpRaftDir)

	tmpStableStoreDir, _ := os.MkdirTemp("", "stable_store*")
	defer os.RemoveAll(tmpStableStoreDir)

	db, err := badger.Open(badger.DefaultOptions(tmpStoreDir))
	if err != nil {
		log.Fatal().Msg(err.Error())
	}
	raftDB, err := badger.Open(badger.DefaultOptions(tmpStableStoreDir))
	if err != nil {
		log.Fatal().Msg(err.Error())
	}

	cfg := &config.Config{
		Cluster: config.ClusterConfig{
			NodeID: "localhost",
		},
		Http: config.HttpConfig{
			Port: "9140",
		},
		Raft: config.RaftConfig{
			Address: "localhost:9141",
		},
		Queue: config.QueueConfig{
			AcknowledgementCheckInterval: 1,
			QueueStats:                   config.QueueStatsConfig{WindowSide: 10},
		},
	}

	n := NewNode(db, raftDB, tmpRaftDir, cfg, []string{})
	n.Initialize()

	queueName := "test_queue_4"

	// Simple way to ensure there is a leader.
	time.Sleep(3 * time.Second)
	err = n.CreateQueue("delayed", queueName, entity.QueueSettings{})
	assert.Nil(t, err)

	assert.True(t, n.IsLeader())

	m1, err := n.Enqueue(queueName, 0, "default", 10, "message 1", nil)
	assert.Nil(t, err)
	assert.NotNil(t, m1)
	assert.Equal(t, "message 1", m1.Content)
	assert.Equal(t, int64(10), m1.Priority)

	m2, err := n.Enqueue(queueName, 0, "default", 5, "message 2", nil)
	assert.Nil(t, err)
	assert.NotNil(t, m2)
	assert.Equal(t, "message 2", m2.Content)
	assert.Equal(t, int64(5), m2.Priority)

	m3, err := n.Enqueue(queueName, 0, "default", 10, "message 3", nil)
	assert.Nil(t, err)
	assert.NotNil(t, m3)
	assert.Equal(t, "message 3", m3.Content)
	assert.Equal(t, int64(10), m3.Priority)

	err = n.UpdatePriority(queueName, m1.ID, 20)
	assert.Nil(t, err)

	err = n.UpdatePriority(queueName, m3.ID, 2)
	assert.Nil(t, err)

	m, err := n.Dequeue(queueName, true)
	assert.Nil(t, err)
	assert.Equal(t, m3.ID, m.ID)
	assert.Equal(t, m3.Content, m.Content)
	assert.Equal(t, int64(2), m.Priority)

	m, err = n.Dequeue(queueName, true)
	assert.Nil(t, err)
	assert.Equal(t, m2.ID, m.ID)
	assert.Equal(t, m2.Content, m.Content)
	assert.Equal(t, m2.Priority, m.Priority)

	m, err = n.Dequeue(queueName, false)
	assert.Nil(t, err)
	assert.Equal(t, m1.ID, m.ID)
	assert.Equal(t, m1.Content, m.Content)
	assert.Equal(t, int64(20), m.Priority)

	err = n.DeleteQueue(queueName)
	assert.Nil(t, err)
}

func TestBackupRestore(t *testing.T) {
	tmpStoreDir, _ := os.MkdirTemp("", "db*")
	defer os.RemoveAll(tmpStoreDir)

	tmpRaftDir, _ := os.MkdirTemp("", "raft*")
	defer os.RemoveAll(tmpRaftDir)

	tmpStableStoreDir, _ := os.MkdirTemp("", "stable_store*")
	defer os.RemoveAll(tmpStableStoreDir)

	db, err := badger.Open(badger.DefaultOptions(tmpStoreDir))
	if err != nil {
		log.Fatal().Msg(err.Error())
	}
	raftDB, err := badger.Open(badger.DefaultOptions(tmpStableStoreDir))
	if err != nil {
		log.Fatal().Msg(err.Error())
	}

	cfg := &config.Config{
		Cluster: config.ClusterConfig{
			NodeID: "localhost",
		},
		Http: config.HttpConfig{
			Port: "9100",
		},
		Raft: config.RaftConfig{
			Address: "localhost:9101",
		},
		Queue: config.QueueConfig{
			AcknowledgementCheckInterval: 1,
			QueueStats:                   config.QueueStatsConfig{WindowSide: 10},
		},
	}

	n := NewNode(db, raftDB, tmpRaftDir, cfg, []string{})
	n.Initialize()

	// Simple way to ensure there is a leader.
	err = n.InitIDGenerator()
	assert.Nil(t, err)

	id := n.GenerateID()
	assert.NotEqual(t, int64(0), id)

	assert.True(t, n.IsLeader())

	queues := n.GetQueues()
	assert.Equal(t, 0, len(queues))

	queueName := "test_queue_5"

	err = n.CreateQueue("delayed", queueName, entity.QueueSettings{})
	assert.Nil(t, err)

	queues = n.GetQueues()
	assert.Equal(t, 1, len(queues))

	for i := 0; i < 10; i++ {
		msg := fmt.Sprintf("message %d", i)
		m, err := n.Enqueue(queueName, 0, "default", 10, msg, nil)
		assert.Nil(t, err)
		assert.NotNil(t, m)
		assert.Equal(t, msg, m.Content)
		assert.Equal(t, int64(10), m.Priority)
	}

	sink := &MockSink{}
	_, err = n.Backup(sink, 0)
	assert.Nil(t, err)

	db.Close()

	tmpStoreDir1, _ := os.MkdirTemp("", "db*")
	defer os.RemoveAll(tmpStoreDir1)

	tmpRaftDir1, _ := os.MkdirTemp("", "raft*")
	defer os.RemoveAll(tmpRaftDir1)

	tmpStableStoreDir1, _ := os.MkdirTemp("", "stable_store*")
	defer os.RemoveAll(tmpStableStoreDir1)

	db1, err := badger.Open(badger.DefaultOptions(tmpStoreDir1))
	if err != nil {
		log.Fatal().Msg(err.Error())
	}
	raftDB1, err := badger.Open(badger.DefaultOptions(tmpStableStoreDir1))
	if err != nil {
		log.Fatal().Msg(err.Error())
	}

	cfg1 := &config.Config{
		Cluster: config.ClusterConfig{
			NodeID: "localhost",
		},
		Http: config.HttpConfig{
			Port: "9000",
		},
		Raft: config.RaftConfig{
			Address: "localhost:9001",
		},
		Queue: config.QueueConfig{
			AcknowledgementCheckInterval: 1,
			QueueStats:                   config.QueueStatsConfig{WindowSide: 10},
		},
	}

	n1 := NewNode(db1, raftDB1, tmpRaftDir1, cfg1, []string{})

	err = n1.Restore(sink, 10)
	assert.Nil(t, err)

	n1.Initialize()

	// Simple way to ensure there is a leader.
	err = n1.InitIDGenerator()
	assert.Nil(t, err)

	assert.True(t, n1.IsLeader())

	queues = n1.GetQueues()
	assert.Equal(t, 1, len(queues))

	queueInfo, err := n1.GetQueueInfo(queueName)
	assert.Nil(t, err)
	assert.Equal(t, queueName, queueInfo.Name)

	for i := 0; i < 10; i++ {
		msg := fmt.Sprintf("message %d", i)
		m, err := n1.Dequeue(queueName, false)
		assert.Nil(t, err)
		assert.NotNil(t, m)
		assert.Equal(t, msg, m.Content)
		assert.Equal(t, int64(10), m.Priority)
	}
}

// TestNodeNonSeedDoesNotBootstrap verifies that a fresh node with bootstrap
// disabled does NOT form its own single-node cluster. It must start with an
// empty Raft configuration, never elect itself leader, report itself as a new
// node that still needs to Join, and report not-ready so Kubernetes keeps it
// out of the service endpoints until it is admitted to the real cluster. This
// guards against the split-brain regression where every pod self-bootstrapped.
func TestNodeNonSeedDoesNotBootstrap(t *testing.T) {
	tmpStoreDir, _ := os.MkdirTemp("", "db*")
	defer os.RemoveAll(tmpStoreDir)

	tmpRaftDir, _ := os.MkdirTemp("", "raft*")
	defer os.RemoveAll(tmpRaftDir)

	tmpStableStoreDir, _ := os.MkdirTemp("", "stable_store*")
	defer os.RemoveAll(tmpStableStoreDir)

	db, err := badger.Open(badger.DefaultOptions(tmpStoreDir))
	if err != nil {
		log.Fatal().Msg(err.Error())
	}
	defer db.Close()
	raftDB, err := badger.Open(badger.DefaultOptions(tmpStableStoreDir))
	if err != nil {
		log.Fatal().Msg(err.Error())
	}
	defer raftDB.Close()

	cfg := &config.Config{
		Cluster: config.ClusterConfig{
			NodeID: "localhost",
		},
		Http: config.HttpConfig{
			Port: "9160",
		},
		Raft: config.RaftConfig{
			Address: "localhost:9161",
		},
		Queue: config.QueueConfig{
			AcknowledgementCheckInterval: 1,
			QueueStats:                   config.QueueStatsConfig{WindowSide: 10},
		},
	}

	n := NewNode(db, raftDB, tmpRaftDir, cfg, []string{})
	n.SetBootstrap(false)
	n.Initialize()

	// Give Raft ample time to (wrongly) elect itself if the gating is broken.
	time.Sleep(2 * time.Second)

	// A non-seed node with no peers must remain a follower with no known leader.
	assert.False(t, n.IsLeader(), "non-seed node must not become leader on its own")
	assert.False(t, n.Ready(), "non-seed node without an elected leader must be not-ready")

	// It had no prior state, so it must still Join the real cluster.
	assert.True(t, n.IsNewNode(), "non-seed fresh node must be flagged as needing to Join")

	// It must not have bootstrapped a configuration containing itself.
	future := n.Raft.GetConfiguration()
	assert.Nil(t, future.Error())
	assert.Equal(
		t, 0, len(future.Configuration().Servers),
		"non-seed node must start with an empty Raft configuration",
	)
}
