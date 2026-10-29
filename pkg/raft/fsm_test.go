package raft

import (
	"io"
	"os"
	"testing"

	"github.com/bwmarrin/snowflake"
	"github.com/dgraph-io/badger/v4"
	"github.com/hashicorp/raft"
	"github.com/kgantsov/doq/pkg/config"
	"github.com/kgantsov/doq/pkg/entity"
	"github.com/kgantsov/doq/pkg/errors"
	pb "github.com/kgantsov/doq/pkg/proto"
	"github.com/kgantsov/doq/pkg/queue"
	"github.com/kgantsov/doq/pkg/storage"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
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
		nil,
	)

	idGenerator, _ := snowflake.NewNode(1)

	queue1, err := queueManager.CreateQueue("delayed", "queue_1", entity.QueueSettings{})
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

	leaderConfig := NewLeaderConfig(
		"node-1",
		"localhost:9000",
		"localhost:10000",
	)
	leaderConfig.Set("node-1", "localhost:9000", "localhost:10000", "localhost:8000")

	fsm := &FSM{
		queueManager: queueManager,
		NodeID:       "node-1",
		db:           db,
		config:       cfg,
		leaderConfig: leaderConfig,
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
		nil,
	)

	fsm = &FSM{
		queueManager: queueManager,
		NodeID:       "node-1",
		db:           db,
		config:       cfg,
		leaderConfig: &LeaderConfig{},
	}

	err = fsm.Restore(sink)
	assert.Nil(t, err)

	assert.Equal(t, "node-1", fsm.leaderConfig.Id)
	assert.Equal(t, "localhost:9000", fsm.leaderConfig.RaftAddr)
	assert.Equal(t, "localhost:10000", fsm.leaderConfig.GrpcAddr)
	assert.Equal(t, "localhost:8000", fsm.leaderConfig.HttpAddr)
	assert.Equal(t, "localhost:10000", fsm.leaderConfig.GetLeaderGrpcAddress())
	assert.Equal(t, "localhost:8000", fsm.leaderConfig.GetLeaderHttpAddress())

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
		nil,
	)

	idGenerator, _ := snowflake.NewNode(1)

	queue1, err := queueManager.CreateQueue("fair", "queue_1", entity.QueueSettings{
		Strategy:   "ROUND_ROBIN",
		MaxUnacked: 40,
		AckTimeout: 3600,
	})
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

	leaderConfig := NewLeaderConfig(
		"node-2", "localhost:9001", "localhost:10001",
	)
	leaderConfig.Set("node-2", "localhost:9001", "localhost:10001", "localhost:8001")

	fsm := &FSM{
		queueManager: queueManager,
		NodeID:       "node-1",
		db:           db,
		config:       cfg,
		leaderConfig: leaderConfig,
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
		nil,
	)

	fsm = &FSM{
		queueManager: queueManager,
		NodeID:       "node-1",
		db:           db,
		config:       cfg,
		leaderConfig: &LeaderConfig{},
	}

	err = fsm.Restore(sink)
	assert.Nil(t, err)

	assert.Equal(t, "node-2", fsm.leaderConfig.Id)
	assert.Equal(t, "localhost:9001", fsm.leaderConfig.RaftAddr)
	assert.Equal(t, "localhost:10001", fsm.leaderConfig.GrpcAddr)
	assert.Equal(t, "localhost:8001", fsm.leaderConfig.HttpAddr)
	assert.Equal(t, "localhost:10001", fsm.leaderConfig.GetLeaderGrpcAddress())
	assert.Equal(t, "localhost:8001", fsm.leaderConfig.GetLeaderHttpAddress())

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

func raftLog(t *testing.T, index uint64, cmd *pb.RaftCommand) *raft.Log {
	data, err := proto.Marshal(cmd)
	require.NoError(t, err)
	return &raft.Log{Index: index, Term: 1, Type: raft.LogCommand, Data: data}
}

func newTestFSM(db *badger.DB, cfg *config.Config) *FSM {
	return &FSM{
		queueManager: queue.NewQueueManager(storage.NewBadgerStore(db), cfg, nil, nil),
		NodeID:       "node-1",
		db:           db,
		config:       cfg,
		leaderConfig: NewLeaderConfig("node-1", "localhost:9000", "localhost:10000"),
	}
}

// TestFSMReplayAfterAckTimeout replays the log of "dequeue without ack, ack
// deadline passes, dequeue with ack" on top of the same data dir, as Raft does
// on restart when there is no snapshot. The ack timeout is in the log, so the
// replay must end with an empty queue instead of resurrecting the message.
func TestFSMReplayAfterAckTimeout(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Queue: config.QueueConfig{
			AcknowledgementCheckInterval: 1,
			QueueStats:                   config.QueueStatsConfig{WindowSide: 10},
		},
	}

	const queueName = "queue_1"
	const msgID = uint64(42)
	logs := []*raft.Log{
		raftLog(t, 1, &pb.RaftCommand{Cmd: &pb.RaftCommand_CreateQueue{CreateQueue: &pb.CreateQueueRequest{
			Type: "delayed", Name: queueName, Settings: &pb.QueueSettings{AckTimeout: 60},
		}}}),
		raftLog(t, 2, &pb.RaftCommand{Cmd: &pb.RaftCommand_Enqueue{Enqueue: &pb.EnqueueRequest{
			QueueName: queueName, Id: msgID, Group: "default", Priority: 10, Content: "message",
		}}}),
		raftLog(t, 3, &pb.RaftCommand{Cmd: &pb.RaftCommand_Dequeue{Dequeue: &pb.DequeueRequest{
			QueueName: queueName, Ack: false,
		}}}),
		raftLog(t, 4, &pb.RaftCommand{Cmd: &pb.RaftCommand_AckTimeout{AckTimeout: &pb.AckTimeoutRequest{
			QueueName: queueName, Id: msgID,
		}}}),
		raftLog(t, 5, &pb.RaftCommand{Cmd: &pb.RaftCommand_Dequeue{Dequeue: &pb.DequeueRequest{
			QueueName: queueName, Ack: true,
		}}}),
	}

	assertEmpty := func(t *testing.T, fsm *FSM) {
		q, err := fsm.queueManager.GetQueue(queueName)
		require.NoError(t, err)
		stats := q.GetStats()
		assert.Equal(t, int64(0), stats.Ready)
		assert.Equal(t, int64(0), stats.Unacked)
		_, err = q.Get(msgID)
		assert.ErrorIs(t, err, errors.ErrMessageNotFound)
	}

	db, err := badger.Open(badger.DefaultOptions(tmpDir))
	require.NoError(t, err)
	fsm := newTestFSM(db, cfg)
	for _, l := range logs {
		resp := fsm.Apply(l).(*FSMResponse)
		require.NoError(t, resp.error, "log index %d", l.Index)
	}
	assertEmpty(t, fsm)
	require.NoError(t, db.Close())

	// Restart: same data dir, fresh in-memory state, replay the whole log.
	db, err = badger.Open(badger.DefaultOptions(tmpDir))
	require.NoError(t, err)
	defer db.Close()
	fsm = newTestFSM(db, cfg)
	for _, l := range logs {
		resp := fsm.Apply(l).(*FSMResponse)
		require.NoError(t, resp.error, "replayed log index %d", l.Index)
	}
	assertEmpty(t, fsm)
}
