package raft

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/kgantsov/doq/pkg/config"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
)

func TestNodeSingleNode(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "db*")
	defer os.RemoveAll(tmpDir)

	tmpRaftDir, _ := os.MkdirTemp("", "raft*")
	defer os.RemoveAll(tmpRaftDir)

	opts := badger.DefaultOptions(tmpDir)
	db, err := badger.Open(opts)
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
		},
	}

	n := NewNode(db, tmpRaftDir, cfg, []string{})
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

	err = n.CreateQueue("delayed", "test_queue")
	assert.Nil(t, err)

	queues = n.GetQueues()
	assert.Equal(t, 1, len(queues))

	queueInfo, err := n.GetQueueInfo("test_queue")
	assert.Nil(t, err)
	assert.Equal(t, "test_queue", queueInfo.Name)
	assert.Equal(t, "delayed", queueInfo.Type)
	assert.Equal(t, int64(0), queueInfo.Ready)
	assert.Equal(t, int64(0), queueInfo.Unacked)
	assert.Equal(t, int64(0), queueInfo.Total)

	m1, err := n.Enqueue("test_queue", "default", 10, "message 1", nil)
	assert.Nil(t, err)
	assert.NotNil(t, m1)
	assert.Equal(t, "message 1", m1.Content)
	assert.Equal(t, int64(10), m1.Priority)

	queueInfo, err = n.GetQueueInfo("test_queue")
	assert.Nil(t, err)
	assert.Equal(t, int64(1), queueInfo.Ready)
	assert.Equal(t, int64(0), queueInfo.Unacked)
	assert.Equal(t, int64(1), queueInfo.Total)

	m2, err := n.Enqueue("test_queue", "default", 5, "message 2", nil)
	assert.Nil(t, err)
	assert.NotNil(t, m2)
	assert.Equal(t, "message 2", m2.Content)
	assert.Equal(t, int64(5), m2.Priority)

	queueInfo, err = n.GetQueueInfo("test_queue")
	assert.Nil(t, err)
	assert.Equal(t, int64(2), queueInfo.Ready)
	assert.Equal(t, int64(0), queueInfo.Unacked)
	assert.Equal(t, int64(2), queueInfo.Total)

	m3, err := n.Enqueue("test_queue", "default", 10, "message 3", map[string]string{"key": "value"})
	assert.Nil(t, err)
	assert.NotNil(t, m3)
	assert.Equal(t, "message 3", m3.Content)
	assert.Equal(t, int64(10), m3.Priority)
	assert.Equal(t, "value", m3.Metadata["key"])

	queueInfo, err = n.GetQueueInfo("test_queue")
	assert.Nil(t, err)
	assert.Equal(t, int64(3), queueInfo.Ready)
	assert.Equal(t, int64(0), queueInfo.Unacked)
	assert.Equal(t, int64(3), queueInfo.Total)

	m, err := n.Get("test_queue", m1.ID)
	assert.Nil(t, err)
	assert.Equal(t, m1.ID, m.ID)
	assert.Equal(t, m1.Content, m.Content)
	assert.Equal(t, m1.Priority, m.Priority)

	m, err = n.Dequeue("test_queue", true)
	assert.Nil(t, err)
	assert.Equal(t, m2.ID, m.ID)
	assert.Equal(t, m2.Content, m.Content)
	assert.Equal(t, m2.Priority, m.Priority)

	queueInfo, err = n.GetQueueInfo("test_queue")
	assert.Nil(t, err)
	assert.Equal(t, int64(2), queueInfo.Ready)
	assert.Equal(t, int64(0), queueInfo.Unacked)
	assert.Equal(t, int64(2), queueInfo.Total)

	m, err = n.Dequeue("test_queue", true)
	assert.Nil(t, err)
	assert.Equal(t, m1.ID, m.ID)
	assert.Equal(t, m1.Content, m.Content)
	assert.Equal(t, m1.Priority, m.Priority)

	queueInfo, err = n.GetQueueInfo("test_queue")
	assert.Nil(t, err)
	assert.Equal(t, int64(1), queueInfo.Ready)
	assert.Equal(t, int64(0), queueInfo.Unacked)
	assert.Equal(t, int64(1), queueInfo.Total)

	err = n.Delete("test_queue", m3.ID)
	assert.Nil(t, err)

	m, err = n.Dequeue("test_queue", false)
	assert.Nil(t, m)
	assert.Error(t, err)

	err = n.DeleteQueue("test_queue")
	assert.Nil(t, err)

	queues = n.GetQueues()
	assert.Equal(t, 0, len(queues))
}

func TestNodeDeleteQueue(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "db*")
	defer os.RemoveAll(tmpDir)

	tmpRaftDir, _ := os.MkdirTemp("", "raft*")
	defer os.RemoveAll(tmpRaftDir)

	opts := badger.DefaultOptions(tmpDir)
	db, err := badger.Open(opts)
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
		},
	}

	n := NewNode(db, tmpRaftDir, cfg, []string{})
	n.Initialize()

	// Simple way to ensure there is a leader.
	time.Sleep(3 * time.Second)

	err = n.DeleteQueue("non_existent_queue")
	assert.Error(t, err)
}

func TestNodeSingleNodeAck(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "db*")
	defer os.RemoveAll(tmpDir)

	tmpRaftDir, _ := os.MkdirTemp("", "raft*")
	defer os.RemoveAll(tmpRaftDir)

	opts := badger.DefaultOptions(tmpDir)
	db, err := badger.Open(opts)
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
		},
	}

	n := NewNode(db, tmpRaftDir, cfg, []string{})
	n.Initialize()

	// Simple way to ensure there is a leader.
	time.Sleep(3 * time.Second)
	err = n.CreateQueue("delayed", "test_queue")
	assert.Nil(t, err)

	assert.True(t, n.IsLeader())

	m1, err := n.Enqueue("test_queue", "default", 10, "message 1", nil)
	assert.Nil(t, err)
	assert.NotNil(t, m1)
	assert.Equal(t, "message 1", m1.Content)
	assert.Equal(t, int64(10), m1.Priority)

	m, err := n.Dequeue("test_queue", false)
	assert.Nil(t, err)
	assert.Equal(t, m1.ID, m.ID)
	assert.Equal(t, m1.Content, m.Content)
	assert.Equal(t, m1.Priority, m.Priority)

	err = n.Ack("test_queue", m.ID)
	assert.Nil(t, err)

	err = n.DeleteQueue("test_queue")
	assert.Nil(t, err)
}

func TestNodeSingleNodeNack(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "db*")
	defer os.RemoveAll(tmpDir)

	tmpRaftDir, _ := os.MkdirTemp("", "raft*")
	defer os.RemoveAll(tmpRaftDir)

	opts := badger.DefaultOptions(tmpDir)
	db, err := badger.Open(opts)
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
		},
	}

	n := NewNode(db, tmpRaftDir, cfg, []string{})
	n.Initialize()

	// Simple way to ensure there is a leader.
	time.Sleep(3 * time.Second)
	err = n.CreateQueue("delayed", "test_queue")
	assert.Nil(t, err)

	assert.True(t, n.IsLeader())

	m1, err := n.Enqueue("test_queue", "default", 10, "message 1", nil)
	assert.Nil(t, err)
	assert.NotNil(t, m1)
	assert.Equal(t, "message 1", m1.Content)
	assert.Equal(t, int64(10), m1.Priority)

	m, err := n.Dequeue("test_queue", false)
	assert.Nil(t, err)
	assert.Equal(t, m1.ID, m.ID)
	assert.Equal(t, m1.Content, m.Content)
	assert.Equal(t, m1.Priority, m.Priority)

	err = n.Nack("test_queue", m.ID, 5, map[string]string{})
	assert.Nil(t, err)

	m, err = n.Dequeue("test_queue", false)
	assert.Nil(t, err)
	assert.Equal(t, m1.ID, m.ID)
	assert.Equal(t, m1.Content, m.Content)
	assert.Equal(t, int64(5), m.Priority)

	err = n.DeleteQueue("test_queue")
	assert.Nil(t, err)
}

func TestNodeSingleNodeUpdatePriority(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "db*")
	defer os.RemoveAll(tmpDir)

	tmpRaftDir, _ := os.MkdirTemp("", "raft*")
	defer os.RemoveAll(tmpRaftDir)

	opts := badger.DefaultOptions(tmpDir)
	db, err := badger.Open(opts)
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
		},
	}

	n := NewNode(db, tmpRaftDir, cfg, []string{})
	n.Initialize()

	// Simple way to ensure there is a leader.
	time.Sleep(3 * time.Second)
	err = n.CreateQueue("delayed", "test_queue")
	assert.Nil(t, err)

	assert.True(t, n.IsLeader())

	m1, err := n.Enqueue("test_queue", "default", 10, "message 1", nil)
	assert.Nil(t, err)
	assert.NotNil(t, m1)
	assert.Equal(t, "message 1", m1.Content)
	assert.Equal(t, int64(10), m1.Priority)

	m2, err := n.Enqueue("test_queue", "default", 5, "message 2", nil)
	assert.Nil(t, err)
	assert.NotNil(t, m2)
	assert.Equal(t, "message 2", m2.Content)
	assert.Equal(t, int64(5), m2.Priority)

	m3, err := n.Enqueue("test_queue", "default", 10, "message 3", nil)
	assert.Nil(t, err)
	assert.NotNil(t, m3)
	assert.Equal(t, "message 3", m3.Content)
	assert.Equal(t, int64(10), m3.Priority)

	err = n.UpdatePriority("test_queue", m1.ID, 20)
	assert.Nil(t, err)

	err = n.UpdatePriority("test_queue", m3.ID, 2)
	assert.Nil(t, err)

	m, err := n.Dequeue("test_queue", true)
	assert.Nil(t, err)
	assert.Equal(t, m3.ID, m.ID)
	assert.Equal(t, m3.Content, m.Content)
	assert.Equal(t, int64(2), m.Priority)

	m, err = n.Dequeue("test_queue", true)
	assert.Nil(t, err)
	assert.Equal(t, m2.ID, m.ID)
	assert.Equal(t, m2.Content, m.Content)
	assert.Equal(t, m2.Priority, m.Priority)

	m, err = n.Dequeue("test_queue", false)
	assert.Nil(t, err)
	assert.Equal(t, m1.ID, m.ID)
	assert.Equal(t, m1.Content, m.Content)
	assert.Equal(t, int64(20), m.Priority)

	err = n.DeleteQueue("test_queue")
	assert.Nil(t, err)
}

func TestBackupRestore(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "db*")
	defer os.RemoveAll(tmpDir)

	tmpRaftDir, _ := os.MkdirTemp("", "raft*")
	defer os.RemoveAll(tmpRaftDir)

	opts := badger.DefaultOptions(tmpDir)
	db, err := badger.Open(opts)
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

	n := NewNode(db, tmpRaftDir, cfg, []string{})
	n.Initialize()

	// Simple way to ensure there is a leader.
	err = n.InitIDGenerator()
	assert.Nil(t, err)

	id := n.GenerateID()
	assert.NotEqual(t, int64(0), id)

	assert.True(t, n.IsLeader())

	queues := n.GetQueues()
	assert.Equal(t, 0, len(queues))

	err = n.CreateQueue("delayed", "test_queue")
	assert.Nil(t, err)

	queues = n.GetQueues()
	assert.Equal(t, 1, len(queues))

	for i := 0; i < 10; i++ {
		msg := fmt.Sprintf("message %d", i)
		m, err := n.Enqueue("test_queue", "default", 10, msg, nil)
		assert.Nil(t, err)
		assert.NotNil(t, m)
		assert.Equal(t, msg, m.Content)
		assert.Equal(t, int64(10), m.Priority)
	}

	sink := &MockSink{}
	_, err = n.Backup(sink, 0)
	assert.Nil(t, err)

	db.Close()

	tmpDir1, _ := os.MkdirTemp("", "db*")
	defer os.RemoveAll(tmpDir1)

	tmpRaftDir1, _ := os.MkdirTemp("", "raft*")
	defer os.RemoveAll(tmpRaftDir1)

	opts1 := badger.DefaultOptions(tmpDir1)
	db1, err := badger.Open(opts1)
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

	n1 := NewNode(db1, tmpRaftDir1, cfg1, []string{})

	err = n1.Restore(sink, 10)
	assert.Nil(t, err)

	n1.Initialize()

	// Simple way to ensure there is a leader.
	err = n1.InitIDGenerator()
	assert.Nil(t, err)

	assert.True(t, n1.IsLeader())

	queues = n1.GetQueues()
	assert.Equal(t, 1, len(queues))

	queueInfo, err := n1.GetQueueInfo("test_queue")
	assert.Nil(t, err)
	assert.Equal(t, "test_queue", queueInfo.Name)

	for i := 0; i < 10; i++ {
		msg := fmt.Sprintf("message %d", i)
		m, err := n1.Dequeue("test_queue", false)
		assert.Nil(t, err)
		assert.NotNil(t, m)
		assert.Equal(t, msg, m.Content)
		assert.Equal(t, int64(10), m.Priority)
	}
}
