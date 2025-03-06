package raft

import (
	"io"
	"os"
	"testing"

	"github.com/bwmarrin/snowflake"
	"github.com/dgraph-io/badger/v4"
	"github.com/kgantsov/doq/pkg/config"
	"github.com/kgantsov/doq/pkg/queue"
	"github.com/kgantsov/doq/pkg/storage"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
)

type MockSink struct {
	data      []byte
	readIndex int64
}

func (ms *MockSink) ID() string {
	return "sink-id"
}

func (ms *MockSink) Cancel() error {
	return nil
}

func (ms *MockSink) Write(p []byte) (n int, err error) {
	ms.data = append(ms.data, p...)
	return len(p), nil
}

func (ms *MockSink) Read(p []byte) (n int, err error) {
	if ms.readIndex >= int64(len(ms.data)) {
		err = io.EOF
		return
	}

	n = copy(p, ms.data[ms.readIndex:])
	ms.readIndex += int64(n)
	return
}

func (ms *MockSink) Close() error {
	return nil
}

func TestFSMPersistRestore(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "db*")
	defer os.RemoveAll(tmpDir)

	opts := badger.DefaultOptions(tmpDir)
	db, err := badger.Open(opts)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}

	cfg := &config.Config{
		Queue: config.QueueConfig{
			AcknowledgementCheckInterval: 1,
			QueueStats:                   config.QueueStatsConfig{WindowSide: 10},
		},
	}

	queueManager := queue.NewQueueManager(
		storage.NewBadgerStore(db),
		cfg,
		nil,
	)

	idGenerator, _ := snowflake.NewNode(1)

	queue1, err := queueManager.CreateQueue("delayed", "queue_1")
	assert.Nil(t, err)

	q1m1, err := queue1.Enqueue(
		uint64(idGenerator.Generate().Int64()), "default", 10, "queue 1 message 1", nil,
	)
	assert.Nil(t, err)
	assert.NotNil(t, q1m1)

	q1m2, err := queue1.Enqueue(
		uint64(idGenerator.Generate().Int64()), "default", 5, "queue 1 message 2", nil,
	)
	assert.Nil(t, err)
	assert.NotNil(t, q1m2)

	q1m3, err := queue1.Enqueue(
		uint64(idGenerator.Generate().Int64()), "default", 15, "queue 1 message 3", nil,
	)
	assert.Nil(t, err)
	assert.NotNil(t, q1m3)

	fsm := &FSM{
		queueManager: queueManager,
		NodeID:       "node-1",
		db:           db,
		config:       cfg,
	}

	snap, err := fsm.Snapshot()
	assert.Nil(t, err)

	sink := &MockSink{}
	snap.Persist(sink)

	// Emulating the node restart by closing the DB and reopening it
	db.Close()

	db, err = badger.Open(opts)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}
	defer db.Close()

	queueManager = queue.NewQueueManager(
		storage.NewBadgerStore(db),
		cfg,
		nil,
	)

	fsm = &FSM{queueManager: queueManager, NodeID: "node-1", db: db, config: cfg}

	err = fsm.Restore(sink)
	assert.Nil(t, err)

	queue1, err = queueManager.GetQueue("queue_1")
	assert.Nil(t, err)

	m, err := queue1.Dequeue(true)
	assert.Nil(t, err)
	assert.Equal(t, q1m2.ID, m.ID)
	assert.Equal(t, q1m2.Content, m.Content)

	m, err = queue1.Dequeue(true)
	assert.Nil(t, err)
	assert.Equal(t, q1m1.ID, m.ID)
	assert.Equal(t, q1m1.Content, m.Content)

	m, err = queue1.Dequeue(true)
	assert.Nil(t, err)
	assert.Equal(t, q1m3.ID, m.ID)
	assert.Equal(t, q1m3.Content, m.Content)

	m, err = queue1.Dequeue(true)
	assert.Nil(t, m)
	assert.Error(t, err)
}

func TestFSMPersistRestoreFairQueue(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "db*")
	defer os.RemoveAll(tmpDir)

	opts := badger.DefaultOptions(tmpDir)
	db, err := badger.Open(opts)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}

	cfg := &config.Config{
		Queue: config.QueueConfig{
			AcknowledgementCheckInterval: 1,
			QueueStats:                   config.QueueStatsConfig{WindowSide: 10},
		},
	}

	queueManager := queue.NewQueueManager(
		storage.NewBadgerStore(db),
		cfg,
		nil,
	)

	idGenerator, _ := snowflake.NewNode(1)

	queue1, err := queueManager.CreateQueue("fair", "queue_1")
	assert.Nil(t, err)

	q1m1, err := queue1.Enqueue(
		uint64(idGenerator.Generate().Int64()), "client-1", 10, "queue 1 message 1", nil,
	)
	assert.Nil(t, err)
	assert.NotNil(t, q1m1)

	q1m2, err := queue1.Enqueue(
		uint64(idGenerator.Generate().Int64()), "client-1", 5, "queue 1 message 2", nil,
	)
	assert.Nil(t, err)
	assert.NotNil(t, q1m2)

	q1m3, err := queue1.Enqueue(
		uint64(idGenerator.Generate().Int64()), "client-1", 15, "queue 1 message 3", nil,
	)
	assert.Nil(t, err)
	assert.NotNil(t, q1m3)

	q1m4, err := queue1.Enqueue(
		uint64(idGenerator.Generate().Int64()), "client-2", 15, "queue 1 message 4", nil,
	)
	assert.Nil(t, err)
	assert.NotNil(t, q1m4)

	q1m5, err := queue1.Enqueue(
		uint64(idGenerator.Generate().Int64()), "client-2", 1, "queue 1 message 5", nil,
	)
	assert.Nil(t, err)
	assert.NotNil(t, q1m5)

	q1m6, err := queue1.Enqueue(
		uint64(idGenerator.Generate().Int64()), "client-3", 1, "queue 1 message 6", nil,
	)
	assert.Nil(t, err)
	assert.NotNil(t, q1m6)

	fsm := &FSM{
		queueManager: queueManager,
		NodeID:       "node-1",
		db:           db,
		config:       cfg,
	}

	snap, err := fsm.Snapshot()
	assert.Nil(t, err)

	sink := &MockSink{}
	snap.Persist(sink)

	db.Close()

	db, err = badger.Open(opts)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}
	defer db.Close()

	queueManager = queue.NewQueueManager(
		storage.NewBadgerStore(db),
		cfg,
		nil,
	)

	fsm = &FSM{queueManager: queueManager, NodeID: "node-1", db: db, config: cfg}

	err = fsm.Restore(sink)
	assert.Nil(t, err)

	queue1, err = queueManager.GetQueue("queue_1")
	assert.Nil(t, err)

	m, err := queue1.Dequeue(true)
	assert.Nil(t, err)
	assert.Equal(t, q1m2.ID, m.ID)
	assert.Equal(t, q1m2.Content, m.Content)

	m, err = queue1.Dequeue(true)
	assert.Nil(t, err)
	assert.Equal(t, q1m5.ID, m.ID)
	assert.Equal(t, q1m5.Content, m.Content)

	m, err = queue1.Dequeue(true)
	assert.Nil(t, err)
	assert.Equal(t, q1m6.ID, m.ID)
	assert.Equal(t, q1m6.Content, m.Content)

	m, err = queue1.Dequeue(true)
	assert.Nil(t, err)
	assert.Equal(t, q1m1.ID, m.ID)
	assert.Equal(t, q1m1.Content, m.Content)

	m, err = queue1.Dequeue(true)
	assert.Nil(t, err)
	assert.Equal(t, q1m4.ID, m.ID)
	assert.Equal(t, q1m4.Content, m.Content)

	m, err = queue1.Dequeue(true)
	assert.Nil(t, err)
	assert.Equal(t, q1m3.ID, m.ID)
	assert.Equal(t, q1m3.Content, m.Content)

	m, err = queue1.Dequeue(true)
	assert.Error(t, err)
	assert.Nil(t, m)
}
