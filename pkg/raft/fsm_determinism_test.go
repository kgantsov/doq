package raft

import (
	"fmt"
	"io"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/kgantsov/doq/pkg/config"
	pb "github.com/kgantsov/doq/pkg/proto"
	"github.com/kgantsov/doq/pkg/storage"
	"github.com/stretchr/testify/require"
)

// TestFSMDeterminism is a property test for the FSM's core requirement: replicas
// that apply the same log must stay identical. Each run generates a random log
// of queue operations and applies every entry to two FSMs, requiring the same
// response from both. In "snapshot" mode the second FSM is replaced partway
// through by a fresh FSM restored from its snapshot, as when a follower
// installs a snapshot or a node restarts from one. At the end both FSMs must
// hold the same messages and in-flight state, and draining them must deliver
// the same messages in the same order.
//
// A failure reports the seed and the last log entries; rerun a single seed with
// DOQ_DETERMINISM_SEED=<seed>.
func TestFSMDeterminism(t *testing.T) {
	queueTypes := []struct {
		name     string
		typ      string
		settings *pb.QueueSettings
		groups   []string
		delayed  bool
	}{
		{
			name:     "delayed",
			typ:      "delayed",
			settings: &pb.QueueSettings{AckTimeout: 60},
			groups:   []string{"default"},
			delayed:  true,
		},
		{
			name: "round_robin",
			typ:  "fair",
			settings: &pb.QueueSettings{
				Strategy: pb.QueueSettings_ROUND_ROBIN, MaxUnacked: 3, AckTimeout: 60,
			},
			// Dotted groups exercise the round-robin group tree.
			groups: []string{"a", "b", "c", "t.x", "t.y", "t.z.1"},
		},
		{
			name: "weighted",
			typ:  "fair",
			settings: &pb.QueueSettings{
				Strategy: pb.QueueSettings_WEIGHTED, MaxUnacked: 3, AckTimeout: 60,
			},
			groups: []string{"a", "b", "c", "d", "e"},
		},
	}

	const stepsPerRun = 300
	seeds := make([]int64, 25)
	for i := range seeds {
		seeds[i] = int64(i + 1)
	}
	if env := os.Getenv("DOQ_DETERMINISM_SEED"); env != "" {
		seed, err := strconv.ParseInt(env, 10, 64)
		require.NoError(t, err, "DOQ_DETERMINISM_SEED")
		seeds = []int64{seed}
	}

	for _, qt := range queueTypes {
		for _, mode := range []string{"replicas", "snapshot"} {
			t.Run(qt.name+"/"+mode, func(t *testing.T) {
				for _, seed := range seeds {
					h := newDeterminismHarness(t, seed, qt.typ, qt.settings, qt.groups, qt.delayed)
					restoreAt := -1
					if mode == "snapshot" {
						restoreAt = h.rng.Intn(stepsPerRun)
					}
					for step := 0; step < stepsPerRun; step++ {
						if step == restoreAt {
							h.restoreB()
						}
						h.randomOp()
					}
					h.requireSameState()
					h.drain()
				}
			})
		}
	}
}

type determinismHarness struct {
	t       *testing.T
	seed    int64
	rng     *rand.Rand
	cfg     *config.Config
	queue   string
	groups  []string
	delayed bool

	a, b  *FSM
	index uint64
	now   time.Time

	nextID uint64
	ids    []uint64 // every ID ever enqueued, including ones gone since
	trace  []string
}

func newDeterminismHarness(
	t *testing.T, seed int64, typ string, settings *pb.QueueSettings, groups []string, delayed bool,
) *determinismHarness {
	h := &determinismHarness{
		t:    t,
		seed: seed,
		rng:  rand.New(rand.NewSource(seed)),
		cfg: &config.Config{
			Queue: config.QueueConfig{
				AcknowledgementCheckInterval: 1,
				QueueStats:                   config.QueueStatsConfig{WindowSide: 10},
			},
		},
		queue:   "q",
		groups:  groups,
		delayed: delayed,
		now:     time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	h.a = h.openFSM()
	h.b = h.openFSM()

	h.apply("create", &pb.RaftCommand{Cmd: &pb.RaftCommand_CreateQueue{CreateQueue: &pb.CreateQueueRequest{
		Type: typ, Name: h.queue, Settings: settings,
	}}})
	return h
}

func (h *determinismHarness) openFSM() *FSM {
	// Badger's default 64MB memtable is allocated up front even in memory, and
	// a run opens hundreds of these.
	opts := badger.DefaultOptions("").WithInMemory(true).WithLogger(nil).
		WithMemTableSize(1 << 20).WithValueThreshold(1 << 10).WithBlockCacheSize(1 << 20)
	db, err := badger.Open(opts)
	require.NoError(h.t, err)
	f := newTestFSM(db, h.cfg)
	h.t.Cleanup(func() {
		// Deleting the queue stops its background goroutines before the
		// database closes under them.
		f.queueManager.DeleteQueue(h.queue)
		db.Close()
	})
	return f
}

// apply appends one entry to the log and applies it to both FSMs. The leader's
// clock moves forward by up to 5s per entry.
func (h *determinismHarness) apply(desc string, cmd *pb.RaftCommand) {
	h.t.Helper()

	h.index++
	h.now = h.now.Add(time.Duration(h.rng.Intn(5000)) * time.Millisecond)
	l := raftLogAt(h.t, h.index, h.now, cmd)

	ra, rb := fsmResult(h.a.Apply(l)), fsmResult(h.b.Apply(l))
	h.trace = append(h.trace, fmt.Sprintf("#%d %s -> %s", h.index, desc, ra))
	if ra != rb {
		h.fail(fmt.Sprintf("replicas diverged at entry #%d (%s): %s vs %s", h.index, desc, ra, rb))
	}
}

func fsmResult(resp interface{}) string {
	r, ok := resp.(*FSMResponse)
	if !ok || r == nil {
		return fmt.Sprintf("%v", resp)
	}
	if r.error != nil {
		return "error: " + r.error.Error()
	}
	return fmt.Sprintf("id=%d group=%s priority=%d", r.ID, r.Group, r.Priority)
}

func (h *determinismHarness) fail(msg string) {
	h.t.Helper()

	start := len(h.trace) - 15
	if start < 0 {
		start = 0
	}
	h.t.Fatalf("seed %d: %s\nlast log entries:\n  %s", h.seed, msg, strings.Join(h.trace[start:], "\n  "))
}

// restoreB replaces the second FSM with a fresh one restored from its snapshot.
func (h *determinismHarness) restoreB() {
	h.t.Helper()

	snap, err := h.b.Snapshot()
	require.NoError(h.t, err)
	sink := &MockSink{}
	require.NoError(h.t, snap.Persist(sink))
	snap.Release()

	restored := h.openFSM()
	require.NoError(h.t, restored.Restore(sink))
	h.b = restored
	h.trace = append(h.trace, fmt.Sprintf("-- follower restored from a snapshot after #%d --", h.index))
}

func (h *determinismHarness) priority() int64 {
	if h.delayed {
		// Delivery times from 20s ago to 40s ahead, so some messages are due and
		// some are still waiting.
		return h.now.Unix() + int64(h.rng.Intn(60)) - 20
	}
	return int64(h.rng.Intn(10))
}

// inFlightID returns a random unacked message, or a random known ID (possibly
// long gone, which exercises the error paths) when none are in flight.
func (h *determinismHarness) inFlightID() uint64 {
	q, err := h.a.queueManager.GetQueue(h.queue)
	require.NoError(h.t, err)
	unacked := q.UnackedSnapshot()
	if len(unacked) > 0 && h.rng.Intn(10) > 0 {
		return unacked[h.rng.Intn(len(unacked))].Id
	}
	return h.anyID()
}

func (h *determinismHarness) anyID() uint64 {
	if len(h.ids) == 0 {
		return 1
	}
	return h.ids[h.rng.Intn(len(h.ids))]
}

func (h *determinismHarness) randomOp() {
	h.t.Helper()

	switch r := h.rng.Intn(100); {
	case r < 30:
		h.nextID++
		h.ids = append(h.ids, h.nextID)
		group := h.groups[h.rng.Intn(len(h.groups))]
		priority := h.priority()
		h.apply(fmt.Sprintf("enqueue id=%d group=%s priority=%d", h.nextID, group, priority),
			&pb.RaftCommand{Cmd: &pb.RaftCommand_Enqueue{Enqueue: &pb.EnqueueRequest{
				QueueName: h.queue, Id: h.nextID, Group: group, Priority: priority,
				Content: fmt.Sprintf("message %d", h.nextID),
			}}})
	case r < 55:
		ack := h.rng.Intn(3) == 0
		h.apply(fmt.Sprintf("dequeue ack=%t", ack),
			&pb.RaftCommand{Cmd: &pb.RaftCommand_Dequeue{Dequeue: &pb.DequeueRequest{
				QueueName: h.queue, Ack: ack,
			}}})
	case r < 65:
		id := h.inFlightID()
		h.apply(fmt.Sprintf("ack id=%d", id),
			&pb.RaftCommand{Cmd: &pb.RaftCommand_Ack{Ack: &pb.AckRequest{QueueName: h.queue, Id: id}}})
	case r < 72:
		id := h.inFlightID()
		var priority int64
		var metadata map[string]string
		if h.rng.Intn(2) == 0 {
			priority = h.priority()
		}
		if h.rng.Intn(2) == 0 {
			metadata = map[string]string{"retry": fmt.Sprint(h.rng.Intn(5))}
		}
		h.apply(fmt.Sprintf("nack id=%d priority=%d metadata=%v", id, priority, metadata),
			&pb.RaftCommand{Cmd: &pb.RaftCommand_Nack{Nack: &pb.NackRequest{
				QueueName: h.queue, Id: id, Priority: priority, Metadata: metadata,
			}}})
	case r < 78:
		id := h.inFlightID()
		h.apply(fmt.Sprintf("touch id=%d", id),
			&pb.RaftCommand{Cmd: &pb.RaftCommand_Touch{Touch: &pb.TouchRequest{QueueName: h.queue, Id: id}}})
	case r < 85:
		id := h.inFlightID()
		h.apply(fmt.Sprintf("ack timeout id=%d", id),
			&pb.RaftCommand{Cmd: &pb.RaftCommand_AckTimeout{AckTimeout: &pb.AckTimeoutRequest{
				QueueName: h.queue, Id: id,
			}}})
	case r < 92:
		id := h.anyID()
		priority := h.priority()
		h.apply(fmt.Sprintf("update priority id=%d priority=%d", id, priority),
			&pb.RaftCommand{Cmd: &pb.RaftCommand_UpdatePriority{UpdatePriority: &pb.UpdatePriorityRequest{
				QueueName: h.queue, Id: id, Priority: priority,
			}}})
	default:
		id := h.anyID()
		h.apply(fmt.Sprintf("delete id=%d", id),
			&pb.RaftCommand{Cmd: &pb.RaftCommand_Delete{Delete: &pb.DeleteRequest{QueueName: h.queue, Id: id}}})
	}
}

// requireSameState compares what both FSMs would persist in a snapshot (stored
// messages and in-flight deadlines) and their ready/unacked counts.
func (h *determinismHarness) requireSameState() {
	h.t.Helper()

	stateA, stateB := h.state(h.a), h.state(h.b)
	if stateA != stateB {
		h.fail(fmt.Sprintf("replica state differs after #%d:\n%s\nvs\n%s", h.index, stateA, stateB))
	}
}

func (h *determinismHarness) state(f *FSM) string {
	h.t.Helper()

	q, err := f.queueManager.GetQueue(h.queue)
	require.NoError(h.t, err)
	stats := q.GetStats()

	snap, err := f.Snapshot()
	require.NoError(h.t, err)
	sink := &MockSink{}
	require.NoError(h.t, snap.Persist(sink))
	snap.Release()

	var lines []string
	lines = append(lines, fmt.Sprintf("ready=%d unacked=%d", stats.Ready, stats.Unacked))
	for {
		item, err := storage.ReadSnapshotItem(sink)
		if err == io.EOF {
			break
		}
		require.NoError(h.t, err)
		switch v := item.Item.(type) {
		case *pb.SnapshotItem_UnackedMessage:
			lines = append(lines, fmt.Sprintf("unacked id=%d deadline=%d", v.UnackedMessage.Id, v.UnackedMessage.AckDeadline))
		case *pb.SnapshotItem_Message:
			m := v.Message
			lines = append(lines, fmt.Sprintf("message id=%d group=%s priority=%d content=%q metadata=%v",
				m.Id, m.Group, m.Priority, m.Content, m.Metadata))
		}
	}
	return strings.Join(lines, "\n")
}

// drain times out every in-flight message and then dequeues with ack until the
// queue is empty, far enough in the future that every delayed message is due.
// Both replicas must hand out the same messages in the same order.
func (h *determinismHarness) drain() {
	h.t.Helper()

	q, err := h.a.queueManager.GetQueue(h.queue)
	require.NoError(h.t, err)
	for _, u := range q.UnackedSnapshot() {
		h.apply(fmt.Sprintf("drain: ack timeout id=%d", u.Id),
			&pb.RaftCommand{Cmd: &pb.RaftCommand_AckTimeout{AckTimeout: &pb.AckTimeoutRequest{
				QueueName: h.queue, Id: u.Id,
			}}})
	}

	h.now = h.now.Add(24 * time.Hour)
	for i := 0; i <= len(h.ids); i++ {
		h.apply("drain: dequeue ack=true",
			&pb.RaftCommand{Cmd: &pb.RaftCommand_Dequeue{Dequeue: &pb.DequeueRequest{
				QueueName: h.queue, Ack: true,
			}}})
		if q.GetStats().Total == 0 {
			break
		}
	}
	h.requireSameState()
}
