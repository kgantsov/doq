package raft

import (
	"fmt"
	"io"
	"os"
	"testing"
	"time"

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

	m, err := queue1.Dequeue(true, 0, time.Now())
	assert.Nil(t, err)
	assert.Equal(t, q1m2.ID, m.ID)
	assert.Equal(t, q1m2.Content, m.Content)

	m, err = queue1.Dequeue(true, 0, time.Now())
	assert.Nil(t, err)
	assert.Equal(t, q1m1.ID, m.ID)
	assert.Equal(t, q1m1.Content, m.Content)

	m, err = queue1.Dequeue(true, 0, time.Now())
	assert.Nil(t, err)
	assert.Equal(t, q1m3.ID, m.ID)
	assert.Equal(t, q1m3.Content, m.Content)

	m, err = queue1.Dequeue(true, 0, time.Now())
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

	m, err := queue1.Dequeue(true, 0, time.Now())
	assert.Nil(t, err)
	assert.Equal(t, q1m2.ID, m.ID)
	assert.Equal(t, q1m2.Content, m.Content)

	m, err = queue1.Dequeue(true, 0, time.Now())
	assert.Nil(t, err)
	assert.Equal(t, q1m5.ID, m.ID)
	assert.Equal(t, q1m5.Content, m.Content)

	m, err = queue1.Dequeue(true, 0, time.Now())
	assert.Nil(t, err)
	assert.Equal(t, q1m6.ID, m.ID)
	assert.Equal(t, q1m6.Content, m.Content)

	m, err = queue1.Dequeue(true, 0, time.Now())
	assert.Nil(t, err)
	assert.Equal(t, q1m1.ID, m.ID)
	assert.Equal(t, q1m1.Content, m.Content)

	m, err = queue1.Dequeue(true, 0, time.Now())
	assert.Nil(t, err)
	assert.Equal(t, q1m4.ID, m.ID)
	assert.Equal(t, q1m4.Content, m.Content)

	m, err = queue1.Dequeue(true, 0, time.Now())
	assert.Nil(t, err)
	assert.Equal(t, q1m3.ID, m.ID)
	assert.Equal(t, q1m3.Content, m.Content)

	m, err = queue1.Dequeue(true, 0, time.Now())
	assert.Error(t, err)
	assert.Nil(t, m)
}

func raftLog(t *testing.T, index uint64, cmd *pb.RaftCommand) *raft.Log {
	return raftLogAt(t, index, time.Now(), cmd)
}

// raftLogAt builds a log entry as the leader would have appended it at time at.
func raftLogAt(t *testing.T, index uint64, at time.Time, cmd *pb.RaftCommand) *raft.Log {
	data, err := proto.Marshal(cmd)
	require.NoError(t, err)
	return &raft.Log{Index: index, Term: 1, Type: raft.LogCommand, Data: data, AppendedAt: at}
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

// TestFSMWeightedQueueIsDeterministic applies the same log to two independent
// FSMs, as happens on a leader and a follower, or on the original run and a
// replay after restart. Weighted fair queues sample a group at random on every
// dequeue, so both replicas must draw the same sample from the log entry, not
// from their own RNG, or they hand out different messages.
func TestFSMWeightedQueueIsDeterministic(t *testing.T) {
	cfg := &config.Config{
		Queue: config.QueueConfig{
			AcknowledgementCheckInterval: 1,
			QueueStats:                   config.QueueStatsConfig{WindowSide: 10},
		},
	}

	const queueName = "weighted"
	var logs []*raft.Log
	index := uint64(0)
	appendLog := func(cmd *pb.RaftCommand) {
		index++
		logs = append(logs, raftLog(t, index, cmd))
	}

	appendLog(&pb.RaftCommand{Cmd: &pb.RaftCommand_CreateQueue{CreateQueue: &pb.CreateQueueRequest{
		Type: "fair",
		Name: queueName,
		Settings: &pb.QueueSettings{
			Strategy:   pb.QueueSettings_WEIGHTED,
			MaxUnacked: 3,
			AckTimeout: 60,
		},
	}}})
	for i := 0; i < 200; i++ {
		appendLog(&pb.RaftCommand{Cmd: &pb.RaftCommand_Enqueue{Enqueue: &pb.EnqueueRequest{
			QueueName: queueName,
			Id:        uint64(i + 1),
			Group:     fmt.Sprintf("customer-%d", i%10),
			Priority:  10,
			Content:   fmt.Sprintf("message %d", i),
		}}})
	}
	for i := 0; i < 150; i++ {
		appendLog(&pb.RaftCommand{Cmd: &pb.RaftCommand_Dequeue{Dequeue: &pb.DequeueRequest{
			QueueName: queueName,
			Ack:       i%3 == 0,
		}}})
	}

	type result struct {
		ID  uint64
		Err string
	}
	run := func() ([]result, *queue.QueueInfo) {
		db, err := badger.Open(badger.DefaultOptions("").WithInMemory(true))
		require.NoError(t, err)
		t.Cleanup(func() { db.Close() })

		fsm := newTestFSM(db, cfg)
		var results []result
		for _, l := range logs {
			resp := fsm.Apply(l).(*FSMResponse)
			r := result{ID: resp.ID}
			if resp.error != nil {
				r.Err = resp.error.Error()
			}
			results = append(results, r)
		}

		q, err := fsm.queueManager.GetQueue(queueName)
		require.NoError(t, err)
		return results, q.GetStats()
	}

	leaderResults, leaderStats := run()
	followerResults, followerStats := run()

	assert.Equal(t, leaderResults, followerResults)
	assert.Equal(t, leaderStats.Ready, followerStats.Ready)
	assert.Equal(t, leaderStats.Unacked, followerStats.Unacked)
}

// TestFSMSnapshotRestoresUnacked takes a snapshot while messages are in flight
// and restores it into a fresh FSM, as a follower installing a snapshot or a
// node restarting from one does. In-flight messages must come back unacked with
// their deadlines, not ready, and the restored FSM must then behave exactly like
// the original for the rest of the log.
func TestFSMSnapshotRestoresUnacked(t *testing.T) {
	cfg := &config.Config{
		Queue: config.QueueConfig{
			AcknowledgementCheckInterval: 1,
			QueueStats:                   config.QueueStatsConfig{WindowSide: 10},
		},
	}

	queues := []struct {
		name       string
		typ        string
		strategy   pb.QueueSettings_Strategy
		maxUnacked uint32
	}{
		{"delayed", "delayed", pb.QueueSettings_STRATEGY_UNSPECIFIED, 0},
		{"weighted", "fair", pb.QueueSettings_WEIGHTED, 4},
		{"round_robin", "fair", pb.QueueSettings_ROUND_ROBIN, 4},
	}

	index := uint64(0)
	nextLog := func(cmd *pb.RaftCommand) *raft.Log {
		index++
		return raftLog(t, index, cmd)
	}
	dequeue := func(queueName string, ack bool) *raft.Log {
		return nextLog(&pb.RaftCommand{Cmd: &pb.RaftCommand_Dequeue{Dequeue: &pb.DequeueRequest{
			QueueName: queueName, Ack: ack,
		}}})
	}

	var before []*raft.Log
	id := uint64(0)
	for _, qd := range queues {
		before = append(before, nextLog(&pb.RaftCommand{Cmd: &pb.RaftCommand_CreateQueue{CreateQueue: &pb.CreateQueueRequest{
			Type: qd.typ,
			Name: qd.name,
			Settings: &pb.QueueSettings{
				Strategy:   qd.strategy,
				MaxUnacked: qd.maxUnacked,
				AckTimeout: 60,
			},
		}}}))
		for i := 0; i < 60; i++ {
			id++
			before = append(before, nextLog(&pb.RaftCommand{Cmd: &pb.RaftCommand_Enqueue{Enqueue: &pb.EnqueueRequest{
				QueueName: qd.name,
				Id:        id,
				Group:     fmt.Sprintf("customer-%d", i%6),
				Priority:  int64(i % 7),
				Content:   fmt.Sprintf("message %d", id),
			}}}))
		}
		for i := 0; i < 30; i++ {
			before = append(before, dequeue(qd.name, i%3 == 0))
		}
	}

	openFSM := func() *FSM {
		db, err := badger.Open(badger.DefaultOptions("").WithInMemory(true))
		require.NoError(t, err)
		t.Cleanup(func() { db.Close() })
		return newTestFSM(db, cfg)
	}

	original := openFSM()
	// Dequeues may legitimately fail once every group is at its unacked limit,
	// so only check that each queue was created.
	for _, l := range before {
		original.Apply(l)
	}
	for _, qd := range queues {
		_, err := original.queueManager.GetQueue(qd.name)
		require.NoError(t, err, qd.name)
	}

	snap, err := original.Snapshot()
	require.NoError(t, err)
	sink := &MockSink{}
	require.NoError(t, snap.Persist(sink))
	snap.Release()

	restored := openFSM()
	require.NoError(t, restored.Restore(sink))

	unackedIDs := map[string][]uint64{}
	for _, qd := range queues {
		oq, err := original.queueManager.GetQueue(qd.name)
		require.NoError(t, err)
		rq, err := restored.queueManager.GetQueue(qd.name)
		require.NoError(t, err)

		os, rs := oq.GetStats(), rq.GetStats()
		require.Positive(t, os.Unacked, qd.name)
		assert.Equal(t, os.Ready, rs.Ready, qd.name)
		assert.Equal(t, os.Unacked, rs.Unacked, qd.name)
		assert.Equal(t, oq.UnackedSnapshot(), rq.UnackedSnapshot(), qd.name)

		for _, u := range oq.UnackedSnapshot() {
			unackedIDs[qd.name] = append(unackedIDs[qd.name], u.Id)
		}
	}

	// The rest of the log: settle some in-flight messages and keep dequeuing.
	var after []*raft.Log
	for _, qd := range queues {
		ids := unackedIDs[qd.name]
		after = append(after,
			nextLog(&pb.RaftCommand{Cmd: &pb.RaftCommand_Ack{Ack: &pb.AckRequest{
				QueueName: qd.name, Id: ids[0],
			}}}),
			nextLog(&pb.RaftCommand{Cmd: &pb.RaftCommand_AckTimeout{AckTimeout: &pb.AckTimeoutRequest{
				QueueName: qd.name, Id: ids[1],
			}}}),
		)
		for i := 0; i < 30; i++ {
			after = append(after, dequeue(qd.name, i%2 == 0))
		}
	}

	type result struct {
		ID  uint64
		Err string
	}
	apply := func(fsm *FSM) []result {
		var results []result
		for _, l := range after {
			resp := fsm.Apply(l).(*FSMResponse)
			r := result{ID: resp.ID}
			if resp.error != nil {
				r.Err = resp.error.Error()
			}
			results = append(results, r)
		}
		return results
	}

	assert.Equal(t, apply(original), apply(restored))
	for _, qd := range queues {
		oq, _ := original.queueManager.GetQueue(qd.name)
		rq, _ := restored.queueManager.GetQueue(qd.name)
		assert.Equal(t, oq.GetStats().Ready, rq.GetStats().Ready, qd.name)
		assert.Equal(t, oq.UnackedSnapshot(), rq.UnackedSnapshot(), qd.name)
	}
}

// TestFSMUsesLeaderAppendTime checks that the FSM takes "now" from the entry's
// AppendedAt, never the local clock: it decides whether a delayed message is due
// and sets ack deadlines. The entries are stamped in 2020, so the wall clock
// would give different answers, and so would any replica or replay applying
// them at a different time.
func TestFSMUsesLeaderAppendTime(t *testing.T) {
	cfg := &config.Config{
		Queue: config.QueueConfig{
			AcknowledgementCheckInterval: 1,
			QueueStats:                   config.QueueStatsConfig{WindowSide: 10},
		},
	}

	const queueName = "delayed"
	const msgID = uint64(7)
	const ackTimeout = 60
	base := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	// Due at base+100s: long past by the wall clock, still ahead of the leader's
	// clock for the first dequeue.
	deliverAt := base.Add(100 * time.Second).Unix()

	logs := []*raft.Log{
		raftLogAt(t, 1, base, &pb.RaftCommand{Cmd: &pb.RaftCommand_CreateQueue{CreateQueue: &pb.CreateQueueRequest{
			Type: "delayed", Name: queueName, Settings: &pb.QueueSettings{AckTimeout: ackTimeout},
		}}}),
		raftLogAt(t, 2, base, &pb.RaftCommand{Cmd: &pb.RaftCommand_Enqueue{Enqueue: &pb.EnqueueRequest{
			QueueName: queueName, Id: msgID, Group: "default", Priority: deliverAt, Content: "message",
		}}}),
		// Not due yet at base+50s.
		raftLogAt(t, 3, base.Add(50*time.Second), &pb.RaftCommand{Cmd: &pb.RaftCommand_Dequeue{Dequeue: &pb.DequeueRequest{
			QueueName: queueName, Ack: false,
		}}}),
		// Due at base+200s; the ack deadline is base+200s+ackTimeout.
		raftLogAt(t, 4, base.Add(200*time.Second), &pb.RaftCommand{Cmd: &pb.RaftCommand_Dequeue{Dequeue: &pb.DequeueRequest{
			QueueName: queueName, Ack: false,
		}}}),
		// Touch at base+300s moves the deadline to base+300s+ackTimeout.
		raftLogAt(t, 5, base.Add(300*time.Second), &pb.RaftCommand{Cmd: &pb.RaftCommand_Touch{Touch: &pb.TouchRequest{
			QueueName: queueName, Id: msgID,
		}}}),
	}

	db, err := badger.Open(badger.DefaultOptions("").WithInMemory(true))
	require.NoError(t, err)
	defer db.Close()
	fsm := newTestFSM(db, cfg)

	for _, l := range logs[:3] {
		fsm.Apply(l)
	}
	q, err := fsm.queueManager.GetQueue(queueName)
	require.NoError(t, err)
	assert.Equal(t, int64(1), q.GetStats().Ready, "dequeue at base+50s must not deliver a message due at base+100s")

	resp := fsm.Apply(logs[3]).(*FSMResponse)
	require.NoError(t, resp.error)
	assert.Equal(t, msgID, resp.ID)
	assert.Equal(t,
		[]*pb.UnackedMessage{{Id: msgID, AckDeadline: base.Add((200 + ackTimeout) * time.Second).Unix()}},
		q.UnackedSnapshot(),
	)

	resp = fsm.Apply(logs[4]).(*FSMResponse)
	require.NoError(t, resp.error)
	assert.Equal(t,
		[]*pb.UnackedMessage{{Id: msgID, AckDeadline: base.Add((300 + ackTimeout) * time.Second).Unix()}},
		q.UnackedSnapshot(),
	)
}

// TestFSMRestoreReplacesExistingState installs a snapshot on an FSM that
// already has state, as a running follower does when it has fallen too far
// behind for the leader to send it the log. The follower's own state must be
// replaced, not merged: queues and messages that are gone from the snapshot
// must be gone after the restore, from memory and from the store, or they
// come back in the follower's next snapshot and on its next restart.
func TestFSMRestoreReplacesExistingState(t *testing.T) {
	cfg := &config.Config{
		Queue: config.QueueConfig{
			AcknowledgementCheckInterval: 1,
			QueueStats:                   config.QueueStatsConfig{WindowSide: 10},
		},
	}

	index := uint64(0)
	nextLog := func(cmd *pb.RaftCommand) *raft.Log {
		index++
		return raftLog(t, index, cmd)
	}
	createQueue := func(name string) *raft.Log {
		return nextLog(&pb.RaftCommand{Cmd: &pb.RaftCommand_CreateQueue{CreateQueue: &pb.CreateQueueRequest{
			Type: "delayed", Name: name, Settings: &pb.QueueSettings{AckTimeout: 60},
		}}})
	}
	enqueue := func(queueName string, id uint64) *raft.Log {
		return nextLog(&pb.RaftCommand{Cmd: &pb.RaftCommand_Enqueue{Enqueue: &pb.EnqueueRequest{
			QueueName: queueName, Id: id, Priority: int64(id), Content: fmt.Sprintf("message %d", id),
		}}})
	}

	// What the follower applied before it fell behind.
	shared := []*raft.Log{
		createQueue("kept"),
		enqueue("kept", 1),
		enqueue("kept", 2),
		enqueue("kept", 3),
		createQueue("deleted"),
		enqueue("deleted", 10),
	}
	// What the leader applied after that, up to its snapshot.
	missed := []*raft.Log{
		nextLog(&pb.RaftCommand{Cmd: &pb.RaftCommand_Delete{Delete: &pb.DeleteRequest{QueueName: "kept", Id: 1}}}),
		nextLog(&pb.RaftCommand{Cmd: &pb.RaftCommand_DeleteQueue{DeleteQueue: &pb.DeleteQueueRequest{Name: "deleted"}}}),
		enqueue("kept", 4),
		nextLog(&pb.RaftCommand{Cmd: &pb.RaftCommand_Dequeue{Dequeue: &pb.DequeueRequest{QueueName: "kept", Ack: false}}}),
	}

	openFSM := func() (*FSM, storage.Store) {
		db, err := badger.Open(badger.DefaultOptions("").WithInMemory(true))
		require.NoError(t, err)
		t.Cleanup(func() { db.Close() })
		return newTestFSM(db, cfg), storage.NewBadgerStore(db)
	}
	applyAll := func(fsm *FSM, logs []*raft.Log) {
		for _, l := range logs {
			resp := fsm.Apply(l).(*FSMResponse)
			require.NoError(t, resp.error, "log index %d", l.Index)
		}
	}

	leader, _ := openFSM()
	applyAll(leader, shared)
	applyAll(leader, missed)

	follower, followerStore := openFSM()
	applyAll(follower, shared)

	leaderSnapshot := snapshotItems(t, leader)
	require.NoError(t, follower.Restore(itemsSink(t, leaderSnapshot)))

	_, err := follower.queueManager.GetQueue("deleted")
	assert.ErrorIs(t, err, errors.ErrQueueNotFound, "a queue deleted on the leader survived the restore")
	_, err = followerStore.Get("kept", 1)
	assert.Error(t, err, "a message deleted on the leader survived the restore in the store")

	kept, err := follower.queueManager.GetQueue("kept")
	require.NoError(t, err)
	stats := kept.GetStats()
	assert.Equal(t, int64(2), stats.Ready)
	assert.Equal(t, int64(1), stats.Unacked)

	// The follower's next snapshot must match the leader's.
	assert.ElementsMatch(t, itemStrings(leaderSnapshot), itemStrings(snapshotItems(t, follower)))
}

// snapshotItems takes a snapshot of fsm and returns its items.
func snapshotItems(t *testing.T, fsm *FSM) []*pb.SnapshotItem {
	t.Helper()
	snap, err := fsm.Snapshot()
	require.NoError(t, err)
	sink := &MockSink{}
	require.NoError(t, snap.Persist(sink))
	snap.Release()

	var items []*pb.SnapshotItem
	for {
		item, err := storage.ReadSnapshotItem(sink)
		if err == io.EOF {
			return items
		}
		require.NoError(t, err)
		items = append(items, item)
	}
}

// itemsSink returns a sink that reads back as a snapshot of items.
func itemsSink(t *testing.T, items []*pb.SnapshotItem) *MockSink {
	t.Helper()
	sink := &MockSink{}
	for _, item := range items {
		require.NoError(t, storage.WriteSnapshotItem(sink, item))
	}
	return sink
}

// itemStrings renders snapshot items for comparing snapshots whose queues may
// be written in any order.
func itemStrings(items []*pb.SnapshotItem) []string {
	out := make([]string, len(items))
	for i, item := range items {
		out[i] = item.String()
	}
	return out
}

// TestFSMRestorePreservesDeliveryCount guards the restore path. The snapshot
// carries delivery_count (it is written field-agnostically from the stored
// bytes), but Queue.Enqueue has no parameter for it — restoring through Enqueue
// silently resets every counter and hands poison messages a fresh delivery
// budget. The queue's delivery_limit must survive too.
func TestFSMRestorePreservesDeliveryCount(t *testing.T) {
	for _, queueType := range []string{"delayed", "fair"} {
		t.Run(queueType, func(t *testing.T) {
			tmpDir, _ := os.MkdirTemp("", "db*")
			defer os.RemoveAll(tmpDir)

			opts := badger.DefaultOptions(tmpDir)
			db, err := badger.Open(opts)
			require.NoError(t, err)

			cfg := &config.Config{
				Queue: config.QueueConfig{
					AcknowledgementCheckInterval: 1,
					QueueStats:                   config.QueueStatsConfig{WindowSide: 10},
				},
			}
			settings := entity.QueueSettings{
				AckTimeout:    3600,
				DeliveryLimit: 5,
			}
			if queueType == "fair" {
				settings.Strategy = "WEIGHTED"
				settings.MaxUnacked = 10
			}

			queueManager := queue.NewQueueManager(storage.NewBadgerStore(db), cfg, nil, nil)
			q, err := queueManager.CreateQueue(queueType, "queue_1", settings)
			require.NoError(t, err)

			// readyID stays on the ready queue; unackedID is left in flight, so
			// the two restore paths (EnqueueRestored and RestoreUnacked) are both
			// exercised.
			const readyID, unackedID = uint64(1), uint64(2)
			_, err = q.Enqueue(readyID, "g1", 10, "ready", nil)
			require.NoError(t, err)
			_, err = q.Enqueue(unackedID, "g1", 20, "unacked", nil)
			require.NoError(t, err)

			// Deliver each message a few times so both carry a non-zero count.
			for i := 0; i < 3; i++ {
				m, err := q.Dequeue(false, 0, time.Now())
				require.NoError(t, err)
				if i < 2 {
					require.NoError(t, q.Nack(m.ID, 0, nil))
				}
			}
			// One more delivery, left unacked, for the in-flight message.
			inFlight, err := q.Dequeue(false, 0, time.Now())
			require.NoError(t, err)

			before := map[uint64]uint32{}
			for _, id := range []uint64{readyID, unackedID} {
				m, err := q.Get(id)
				require.NoError(t, err)
				require.NotZero(t, m.DeliveryCount, "message %d should have been delivered", id)
				before[id] = m.DeliveryCount
			}
			require.NotNil(t, inFlight)

			leaderConfig := NewLeaderConfig("node-1", "localhost:9000", "localhost:10000")
			leaderConfig.Set("node-1", "localhost:9000", "localhost:10000", "localhost:8000")

			fsm := &FSM{
				queueManager: queueManager,
				NodeID:       "node-1",
				db:           db,
				config:       cfg,
				leaderConfig: leaderConfig,
			}

			snap, err := fsm.Snapshot()
			require.NoError(t, err)

			sink := &MockSink{}
			require.NoError(t, snap.Persist(sink))

			// Emulate a node restart: fresh DB, fresh queue manager, restore.
			db.Close()
			db, err = badger.Open(opts)
			require.NoError(t, err)
			defer db.Close()

			queueManager = queue.NewQueueManager(storage.NewBadgerStore(db), cfg, nil, nil)
			fsm = &FSM{
				queueManager: queueManager,
				NodeID:       "node-1",
				db:           db,
				config:       cfg,
				leaderConfig: leaderConfig,
			}
			require.NoError(t, fsm.Restore(sink))

			restored, err := queueManager.GetQueue("queue_1")
			require.NoError(t, err)
			assert.Equal(
				t, uint32(5), restored.GetStats().Settings.DeliveryLimit,
				"the queue's delivery limit must survive a restore",
			)

			for _, id := range []uint64{readyID, unackedID} {
				m, err := restored.Get(id)
				require.NoError(t, err)
				assert.Equalf(
					t, before[id], m.DeliveryCount,
					"message %d lost its delivery count across snapshot/restore", id,
				)
			}
		})
	}
}

// TestFSMDeliveryCountSurvivesLogReplay covers what moving the delivery counter
// into Store.Dequeue implies for restarts: applyDequeue now writes to BadgerDB,
// which is persistent, so a replayed dequeue could inflate the count and drop a
// message early.
//
// It does not, and the reason is worth pinning:
//
//   - Replay with no snapshot starts at index 1, and the message's own Enqueue
//     entry rewrites it with delivery_count 0, so every later dequeue of it
//     replays onto a reset counter and the final value is reproduced exactly.
//   - Replay after a snapshot is safe because FSM.Restore clears all state
//     first (DeleteAll) and seeds from the snapshot bytes, which carry
//     delivery_count — so only post-snapshot entries apply, once.
//
// Both properties are incidental to code elsewhere (Store.Enqueue not
// preserving an existing count; Restore clearing before it restores), so this
// test guards against a change to either silently making dequeue non-idempotent.
func TestFSMDeliveryCountSurvivesLogReplay(t *testing.T) {
	cfg := &config.Config{Queue: config.QueueConfig{
		AcknowledgementCheckInterval: 1,
		QueueStats:                   config.QueueStatsConfig{WindowSide: 10},
	}}

	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	// Two deliveries, each nacked, so the message ends on delivery_count 2.
	raftLog := []*pb.RaftCommand{
		{Cmd: &pb.RaftCommand_CreateQueue{CreateQueue: &pb.CreateQueueRequest{
			Type: "delayed", Name: "q", Settings: &pb.QueueSettings{AckTimeout: 60},
		}}},
		{Cmd: &pb.RaftCommand_Enqueue{Enqueue: &pb.EnqueueRequest{
			QueueName: "q", Id: 1, Group: "default", Priority: 10, Content: "poison",
		}}},
		{Cmd: &pb.RaftCommand_Dequeue{Dequeue: &pb.DequeueRequest{QueueName: "q"}}},
		{Cmd: &pb.RaftCommand_Nack{Nack: &pb.NackRequest{QueueName: "q", Id: 1}}},
		{Cmd: &pb.RaftCommand_Dequeue{Dequeue: &pb.DequeueRequest{QueueName: "q"}}},
		{Cmd: &pb.RaftCommand_Nack{Nack: &pb.NackRequest{QueueName: "q", Id: 1}}},
	}

	applyAll := func(fsm *FSM) {
		for i, cmd := range raftLog {
			fsm.Apply(raftLogAt(t, uint64(i+1), now.Add(time.Duration(i)*time.Second), cmd))
		}
	}
	deliveryCount := func(fsm *FSM) uint32 {
		q, err := fsm.queueManager.GetQueue("q")
		require.NoError(t, err)
		msg, err := q.Get(1)
		require.NoError(t, err)
		return msg.DeliveryCount
	}

	t.Run("replay without a snapshot", func(t *testing.T) {
		dir, err := os.MkdirTemp("", "replay*")
		require.NoError(t, err)
		defer os.RemoveAll(dir)

		opts := badger.DefaultOptions(dir).WithLogger(nil)
		db, err := badger.Open(opts)
		require.NoError(t, err)

		fsm := newTestFSM(db, cfg)
		applyAll(fsm)
		require.Equal(t, uint32(2), deliveryCount(fsm))

		// Restart with no snapshot: same Badger state, fresh FSM, whole log
		// replayed.
		require.NoError(t, db.Close())
		db, err = badger.Open(opts)
		require.NoError(t, err)
		defer db.Close()

		replayed := newTestFSM(db, cfg)
		applyAll(replayed)
		assert.Equal(
			t, uint32(2), deliveryCount(replayed),
			"log replay must not inflate the delivery count",
		)
	})

	t.Run("replay after a snapshot", func(t *testing.T) {
		db, err := badger.Open(badger.DefaultOptions("").WithInMemory(true).WithLogger(nil))
		require.NoError(t, err)
		defer db.Close()

		fsm := newTestFSM(db, cfg)
		applyAll(fsm)
		require.Equal(t, uint32(2), deliveryCount(fsm))

		snap, err := fsm.Snapshot()
		require.NoError(t, err)
		sink := &MockSink{}
		require.NoError(t, snap.Persist(sink))
		snap.Release()

		// A follower installing this snapshot, then replaying two more entries
		// that were already applied before the snapshot was taken.
		restoreDB, err := badger.Open(badger.DefaultOptions("").WithInMemory(true).WithLogger(nil))
		require.NoError(t, err)
		defer restoreDB.Close()

		restored := newTestFSM(restoreDB, cfg)
		require.NoError(t, restored.Restore(sink))
		assert.Equal(
			t, uint32(2), deliveryCount(restored),
			"the snapshot must carry the delivery count",
		)

		restored.Apply(raftLogAt(t, 7, now.Add(7*time.Second), raftLog[4]))
		restored.Apply(raftLogAt(t, 8, now.Add(8*time.Second), raftLog[5]))
		assert.Equal(
			t, uint32(3), deliveryCount(restored),
			"entries after the snapshot apply exactly once",
		)
	})
}
