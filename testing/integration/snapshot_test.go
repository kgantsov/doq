//go:build integration

package integration

import (
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const (
	// snapshotThreshold is the snapshot threshold of the node under test: low
	// enough that a few batches of enqueues trigger a snapshot.
	snapshotThreshold = 100
	// snapshotNode is the index of the node under test. Only it takes
	// snapshots; the others keep the defaults, so they never snapshot during
	// a test and always restart by replaying the whole log. That keeps them
	// a reference for what the snapshot-restored node should have.
	snapshotNode = 2
	// maxLoadRounds bounds how many batches of enqueues snapshotAfter sends
	// while waiting for a snapshot.
	maxLoadRounds = 20
)

// withSnapshots makes node snapshotNode take a snapshot every
// snapshotThreshold log entries, checking every 1-2 seconds.
func withSnapshots() ClusterOption {
	return WithNodeArgs(snapshotNode,
		"--raft.snapshot_threshold", strconv.Itoa(snapshotThreshold),
		"--raft.snapshot_interval", "1",
	)
}

// snapshotAfter waits until node has a snapshot that captured its state at
// or after mark, calling load to append more log entries until it does. A
// snapshot's directory name records when it was created, which is shortly
// after the state was captured; the one-second margin covers that gap.
func snapshotAfter(t *testing.T, node *Node, mark time.Time, load func()) {
	t.Helper()
	for round := 0; round < maxLoadRounds; round++ {
		load()
		deadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(deadline) {
			if created, ok := node.LatestSnapshot(t); ok && created.After(mark.Add(time.Second)) {
				t.Logf("%s took a snapshot at %s", node.ID, created.Format(time.TimeOnly))
				return
			}
			time.Sleep(200 * time.Millisecond)
		}
	}
	t.Fatalf("%s took no snapshot after %s", node.ID, mark.Format(time.TimeOnly))
}

// restart stops node gracefully and starts it again on the same data.
func restart(t *testing.T, node *Node) {
	t.Helper()
	node.Stop(t)
	node.Start(t)
}

// makeLeader moves leadership to target by killing the leader until target
// wins the election. LeadershipTransfer can't be aimed at a node: it always
// picks the same follower when they're equally up to date. Each killed node
// is restarted before the next round, so the cluster keeps a quorum.
func makeLeader(t *testing.T, c *Cluster, target *Node) {
	t.Helper()
	for round := 0; round < 10; round++ {
		leader := c.WaitForLeader(t)
		if leader == target {
			return
		}
		t.Logf("killing leader %s to get %s elected", leader.ID, target.ID)
		leader.Kill(t)
		c.WaitForNewLeader(t, leader)
		leader.Start(t)
	}
	t.Fatalf("%s did not become leader", target.ID)
}

// TestRoundRobinSurvivesSnapshotRestore checks that a node restored from a
// snapshot keeps the round-robin queue's group order and cursors. They used
// to be left out of snapshots, so a restored node rebuilt its group tree from
// the messages it had, picked different messages than the other replicas,
// and as leader delivered some messages twice and never delivered others.
func TestRoundRobinSurvivesSnapshotRestore(t *testing.T) {
	c := StartCluster(t, 3, withSnapshots())
	leader := c.WaitForLeader(t)
	restored := c.Nodes[snapshotNode]

	const queue = "round_robin"
	require.NoError(t, leader.Client().CreateQueue(queue, "fair",
		map[string]any{"strategy": "round_robin", "max_unacked": 1000, "ack_timeout": 600}))
	d := newDeliveryLog(t, queue, 10)
	waitForReplicas := func() {
		ready := int64(d.enqueued - d.deliveries())
		c.WaitForQueueCounts(t, queue, QueueCounts{Ready: ready, Total: ready})
	}

	// The cursors must move before the snapshot. Otherwise rebuilding the
	// tree in message-ID order recreates the same order and cursors.
	d.Enqueue(leader, 200)
	d.Dequeue(leader, 37)
	waitForReplicas()
	snapshotAfter(t, restored, time.Now(), func() { d.Enqueue(leader, snapshotThreshold) })

	// Entries after the snapshot are replayed on top of the restored state.
	d.Dequeue(leader, 20)
	waitForReplicas()
	restart(t, restored)
	waitForReplicas()

	// The restored node applies these as a follower, from its restored state.
	d.Dequeue(c.WaitForLeader(t), 50)
	waitForReplicas()

	makeLeader(t, c, restored)
	waitForReplicas()
	d.Drain(restored)
	d.AssertExactlyOnce()
}

// TestUnackedSurvivesSnapshotRestore checks that a node restored from a
// snapshot still knows which messages are in flight. Snapshots used to store
// only the messages, so a restored node saw its unacked messages as ready.
func TestUnackedSurvivesSnapshotRestore(t *testing.T) {
	c := StartCluster(t, 3, withSnapshots())
	leader := c.WaitForLeader(t)
	restored := c.Nodes[snapshotNode]

	const queue = "unacked"
	require.NoError(t, leader.Client().CreateQueue(queue, "delayed", map[string]any{"ack_timeout": 600}))
	d := newDeliveryLog(t, queue, 1)

	const inFlight = 30
	d.Enqueue(leader, 100)
	for i := 0; i < inFlight; i++ {
		_, ok, err := leader.Client().Dequeue(queue, false)
		require.NoError(t, err)
		require.True(t, ok)
	}
	want := func() QueueCounts {
		total := int64(d.enqueued)
		return QueueCounts{Ready: total - inFlight, Unacked: inFlight, Total: total}
	}
	c.WaitForQueueCounts(t, queue, want())
	snapshotAfter(t, restored, time.Now(), func() { d.Enqueue(leader, snapshotThreshold) })
	c.WaitForQueueCounts(t, queue, want())

	restart(t, restored)
	c.WaitForQueueCounts(t, queue, want())
}
