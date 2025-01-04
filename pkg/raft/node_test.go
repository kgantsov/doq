package raft

import (
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

	m3, err := n.Enqueue("test_queue", "default", 10, "message 3", map[string]string{"key": "value"})
	assert.Nil(t, err)
	assert.NotNil(t, m3)
	assert.Equal(t, "message 3", m3.Content)
	assert.Equal(t, int64(10), m3.Priority)
	assert.Equal(t, "value", m3.Metadata["key"])

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

	m, err = n.Dequeue("test_queue", true)
	assert.Nil(t, err)
	assert.Equal(t, m1.ID, m.ID)
	assert.Equal(t, m1.Content, m.Content)
	assert.Equal(t, m1.Priority, m.Priority)

	err = n.Delete("test_queue", m3.ID)
	assert.Nil(t, err)

	m, err = n.Dequeue("test_queue", false)
	assert.Nil(t, m)
	assert.Error(t, err)

	err = n.DeleteQueue("test_queue")
	assert.Nil(t, err)
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
