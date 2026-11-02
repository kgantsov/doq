//go:build integration

package integration

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLeaderFailover kills the leader of a 3-node cluster and checks that:
//   - the remaining two nodes elect a new leader,
//   - the queue keeps accepting writes with one node down,
//   - every message is delivered exactly once across the leader change,
//   - the killed node catches up after it restarts.
func TestLeaderFailover(t *testing.T) {
	c := StartCluster(t, 3)
	leader := c.WaitForLeader(t)

	const queue = "failover"
	require.NoError(t, leader.Client().CreateQueue(queue, "delayed", map[string]any{"ack_timeout": 60}))
	d := newDeliveryLog(t, queue, 1)

	d.Enqueue(leader, 20)
	d.Dequeue(leader, 5)
	c.WaitForQueueCounts(t, queue, QueueCounts{Ready: 15, Total: 15})

	t.Logf("killing leader %s", leader.ID)
	old := leader
	old.Kill(t)
	leader = c.WaitForNewLeader(t, old)
	t.Logf("new leader %s", leader.ID)

	// Two of three nodes are a quorum, so writes keep working.
	d.Enqueue(leader, 10)
	d.Dequeue(leader, 25)
	d.Drain(leader)
	d.AssertExactlyOnce()

	// The killed node replays its log and catches up with what it missed.
	old.Start(t)
	c.WaitForQueueCounts(t, queue, QueueCounts{})
	assert.Len(t, c.Running(), 3)
}

// TestExactlyOnceAcrossLeaderChange checks that replicas dequeue the same
// messages: after leadership moves, the new leader must deliver exactly the
// messages the old one hadn't. Weighted fair queues used to pick a group with
// an unseeded RNG, so each replica dequeued different messages.
func TestExactlyOnceAcrossLeaderChange(t *testing.T) {
	queues := []struct {
		name      string
		queueType string
		settings  map[string]any
	}{
		{"delayed", "delayed", map[string]any{"ack_timeout": 600}},
		{"round_robin", "fair", map[string]any{"strategy": "round_robin", "max_unacked": 1000, "ack_timeout": 600}},
		{"weighted", "fair", map[string]any{"strategy": "weighted", "max_unacked": 1000, "ack_timeout": 600}},
	}

	for _, q := range queues {
		t.Run(q.name, func(t *testing.T) {
			c := StartCluster(t, 3)
			leader := c.WaitForLeader(t)
			require.NoError(t, leader.Client().CreateQueue(q.name, q.queueType, q.settings))
			d := newDeliveryLog(t, q.name, 10)

			d.Enqueue(leader, 200)
			d.Dequeue(leader, 100)
			c.WaitForQueueCounts(t, q.name, QueueCounts{Ready: 100, Total: 100})

			// A graceful hand-off: no entries are lost, so any mismatch comes
			// from the replicas' own state.
			old := leader
			require.NoError(t, old.Client().TransferLeadership())
			leader = c.WaitForNewLeader(t, old)
			t.Logf("leader %s -> %s", old.ID, leader.ID)

			d.Drain(leader)
			d.AssertExactlyOnce()
		})
	}
}
