//go:build integration

package integration

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// retryTimeout bounds how long a request is retried while the cluster has no
// leader, e.g. right after the leader was killed.
const retryTimeout = 15 * time.Second

// deliveryLog enqueues messages with unique contents and records every
// delivery, so a test can check that each message was delivered exactly once.
// If replicas diverge (they dequeue different messages for the same Raft
// entry), a new leader delivers some messages twice and never delivers others.
type deliveryLog struct {
	t         *testing.T
	queue     string
	groups    int
	enqueued  int
	delivered map[string]int
}

func newDeliveryLog(t *testing.T, queue string, groups int) *deliveryLog {
	return &deliveryLog{t: t, queue: queue, groups: groups, delivered: map[string]int{}}
}

// Enqueue adds count messages through n. Messages are spread over the groups;
// every third group is nested, to exercise the round-robin group tree.
// Priorities vary within each group, so a group's messages are not consumed
// in the order they were enqueued in. For delayed queues 0-9 are timestamps
// long past, so every message is due.
func (d *deliveryLog) Enqueue(n *Node, count int) {
	d.t.Helper()
	for i := 0; i < count; i++ {
		d.enqueued++
		g := d.enqueued % d.groups
		group := fmt.Sprintf("g%d", g)
		if g%3 == 0 {
			group = fmt.Sprintf("t.g%d", g)
		}
		priority := int64((d.enqueued / d.groups * 7) % 10)
		content := fmt.Sprintf("m%d", d.enqueued)
		retry(d.t, func() error {
			_, err := n.Client().Enqueue(d.queue, group, priority, content)
			return err
		})
	}
}

// Dequeue takes count messages through n, acking them on delivery. It fails
// the test if the queue runs out first.
func (d *deliveryLog) Dequeue(n *Node, count int) {
	d.t.Helper()
	for i := 0; i < count; i++ {
		require.True(d.t, d.dequeueOne(n), "queue is empty after %d deliveries", d.deliveries())
	}
}

// Drain takes messages through n until the queue is empty.
func (d *deliveryLog) Drain(n *Node) {
	d.t.Helper()
	for d.dequeueOne(n) {
	}
}

// AssertExactlyOnce checks that every enqueued message was delivered once.
func (d *deliveryLog) AssertExactlyOnce() {
	d.t.Helper()
	var duplicates, missing, unknown []string
	expected := map[string]bool{}
	for i := 1; i <= d.enqueued; i++ {
		content := fmt.Sprintf("m%d", i)
		expected[content] = true
		switch got := d.delivered[content]; {
		case got == 0:
			missing = append(missing, content)
		case got > 1:
			duplicates = append(duplicates, content)
		}
	}
	for content := range d.delivered {
		if !expected[content] {
			unknown = append(unknown, content)
		}
	}
	assert.Empty(d.t, duplicates, "messages delivered more than once")
	assert.Empty(d.t, missing, "messages never delivered")
	assert.Empty(d.t, unknown, "delivered messages that were never enqueued")
}

func (d *deliveryLog) dequeueOne(n *Node) bool {
	d.t.Helper()
	var msg Message
	var ok bool
	retry(d.t, func() (err error) {
		msg, ok, err = n.Client().Dequeue(d.queue, true)
		return err
	})
	if ok {
		d.delivered[msg.Content]++
	}
	return ok
}

func (d *deliveryLog) deliveries() int {
	total := 0
	for _, c := range d.delivered {
		total += c
	}
	return total
}

// retry calls f until it succeeds or retryTimeout passes. Use it for requests
// that may hit an election. Enqueue and dequeue aren't idempotent: if a
// request was applied but its response was lost, the retry applies it again.
// The tests send requests to the leader, where a failure almost always means
// the request never reached Raft.
func retry(t *testing.T, f func() error) {
	t.Helper()
	var err error
	deadline := time.Now().Add(retryTimeout)
	for {
		if err = f(); err == nil {
			return
		}
		if time.Now().After(deadline) {
			require.NoError(t, err, "request kept failing for %s", retryTimeout)
		}
		time.Sleep(200 * time.Millisecond)
	}
}
