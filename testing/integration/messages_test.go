//go:build integration

package integration

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMessages covers the message API: ordering for each queue type, delayed
// delivery, acks, nacks, the fair queues' per-group unacked limit, priority
// updates and deletes. It started as a port of python/tests/test_messages.py.
//
// Every case runs twice on one shared cluster: through the leader, and
// through a follower, which proxies each request to the leader. Each case
// ends by checking that all three replicas report the same counts.
func TestMessages(t *testing.T) {
	c := StartCluster(t, 3)
	leader := c.WaitForLeader(t)
	var follower *Node
	for _, n := range c.Nodes {
		if n != leader {
			follower = n
			break
		}
	}

	cases := []struct {
		name string
		run  func(q *testQueue)
	}{
		{"delayed_fifo", testDelayedFIFO},
		{"delayed_priority", testDelayedPriority},
		{"delayed_until_due", testDelayedUntilDue},
		{"fair_round_robin", testFairRoundRobin},
		{"fair_max_unacked_round_robin", func(q *testQueue) { testFairMaxUnacked(q, "round_robin") }},
		{"fair_max_unacked_weighted", func(q *testQueue) { testFairMaxUnacked(q, "weighted") }},
		{"ack", testAck},
		{"nack", testNack},
		{"update_priority_and_delete", testUpdatePriorityAndDelete},
	}

	for _, via := range []struct {
		name string
		node *Node
	}{{"via_leader", leader}, {"via_follower", follower}} {
		t.Run(via.name, func(t *testing.T) {
			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					tc.run(&testQueue{t: t, c: c, node: via.node, name: via.name + "-" + tc.name})
				})
			}
		})
	}
}

// testDelayedFIFO: messages with the same priority come out in the order they
// went in. Delayed queues ignore groups (see enqueue).
func testDelayedFIFO(q *testQueue) {
	q.create("delayed", nil)
	var sent []Message
	for i := 1; i <= 8; i++ {
		sent = append(sent, q.enqueue(Message{
			Group:    fmt.Sprintf("customer-%d", (i+1)/2),
			Priority: 10,
			Content:  fmt.Sprintf(`{"id": %d}`, i),
			Metadata: map[string]string{"retry": strconv.Itoa(i)},
		}))
	}
	q.expectOrder(true, sent)
	q.expectEmpty()
	q.expectCounts(QueueCounts{})
}

// testDelayedPriority: lower priorities come out first.
func testDelayedPriority(q *testQueue) {
	q.create("delayed", nil)
	priorities := []int64{123, 523, 35, 7863, 12, 4, 1, 4555}
	var sent []Message
	for i, p := range priorities {
		sent = append(sent, q.enqueue(Message{Priority: p, Content: fmt.Sprintf(`{"id": %d}`, i+1)}))
	}

	q.expectOrder(true, pick(sent, 6, 5, 4, 2, 0, 1, 7, 3))
	q.expectEmpty()
	q.expectCounts(QueueCounts{})
}

// testDelayedUntilDue: a priority in a delayed queue is the Unix time a
// message becomes due. Messages are held until then, and no longer.
func testDelayedUntilDue(q *testQueue) {
	q.create("delayed", nil)
	now := time.Now().Unix()
	later := q.enqueue(Message{Priority: now + 2, Content: "in 2s"})
	soon := q.enqueue(Message{Priority: now + 1, Content: "in 1s"})
	due := q.enqueue(Message{Priority: 10, Content: "due"})

	q.expectOrder(true, []Message{due})
	for _, want := range []Message{soon, later} {
		q.expectEmpty()
		got := q.eventuallyDequeue(true, 4*time.Second)
		q.expectMessage(want, got)
		assert.GreaterOrEqual(q.t, time.Now().Unix(), want.Priority, "%s delivered before it was due", want.Content)
	}
	q.expectEmpty()
	q.expectCounts(QueueCounts{})
}

// testFairRoundRobin: a round-robin fair queue serves its groups in turn, in
// the order they were first seen, whatever their sizes.
func testFairRoundRobin(q *testQueue) {
	q.create("fair", map[string]any{"strategy": "round_robin"})
	groups := []string{"customer-1", "customer-1", "customer-1", "customer-1", "customer-2", "customer-2", "customer-3", "customer-4"}
	var sent []Message
	for i, g := range groups {
		sent = append(sent, q.enqueue(Message{
			Group:    g,
			Priority: 10,
			Content:  fmt.Sprintf(`{"id": %d}`, i+1),
			Metadata: map[string]string{"retry": strconv.Itoa(i % 3)},
		}))
	}

	q.expectOrder(true, pick(sent, 0, 4, 6, 7, 1, 5, 2, 3))
	q.expectEmpty()
	q.expectCounts(QueueCounts{})
}

// testFairMaxUnacked: a fair queue lets each group have at most max_unacked
// messages in flight, so one busy group can't take every consumer.
func testFairMaxUnacked(q *testQueue, strategy string) {
	q.create("fair", map[string]any{"strategy": strategy, "max_unacked": 1})
	a1 := q.enqueue(Message{Group: "a", Priority: 1, Content: "a1"})
	a2 := q.enqueue(Message{Group: "a", Priority: 2, Content: "a2"})
	b1 := q.enqueue(Message{Group: "b", Priority: 1, Content: "b1"})
	q.enqueue(Message{Group: "b", Priority: 2, Content: "b2"})

	first, second := q.dequeue(false), q.dequeue(false)
	assert.ElementsMatch(q.t, []string{a1.Content, b1.Content}, []string{first.Content, second.Content},
		"one message from each group")
	q.expectEmpty() // both groups are at their limit
	q.expectCounts(QueueCounts{Ready: 2, Unacked: 2, Total: 4})

	var inFlightA Message
	if first.Group == "a" {
		inFlightA = first
	} else {
		inFlightA = second
	}
	require.NoError(q.t, q.client().Ack(q.name, inFlightA.ID))
	q.expectMessage(a2, q.dequeue(false))
	q.expectEmpty()
	q.expectCounts(QueueCounts{Ready: 1, Unacked: 2, Total: 3})
}

// testAck: an acked message is gone for good, and can't be acked twice.
func testAck(q *testQueue) {
	q.create("delayed", nil)
	var sent []Message
	for i := 1; i <= 8; i++ {
		sent = append(sent, q.enqueue(Message{Priority: 10, Content: fmt.Sprintf(`{"id": %d}`, i)}))
	}

	for i, want := range sent {
		got := q.dequeue(false)
		q.expectMessage(want, got)
		q.expectCounts(QueueCounts{Ready: int64(len(sent) - i - 1), Unacked: 1, Total: int64(len(sent) - i)})
		require.NoError(q.t, q.client().Ack(q.name, got.ID))
		q.expectStatus(http.StatusNotFound, q.client().Ack(q.name, got.ID), "acking %s twice", got.Content)
	}
	q.expectEmpty()
	q.expectCounts(QueueCounts{})
}

// testNack: a nacked message goes back to the queue with the priority and
// metadata given in the nack.
func testNack(q *testQueue) {
	q.create("delayed", nil)
	var sent []Message
	for i := 1; i <= 3; i++ {
		sent = append(sent, q.enqueue(Message{Priority: 10, Content: fmt.Sprintf(`{"id": %d}`, i)}))
	}
	for _, want := range sent {
		q.expectMessage(want, q.dequeue(false))
	}
	q.expectEmpty()
	q.expectCounts(QueueCounts{Unacked: 3, Total: 3})

	// Nack in reverse priority, so the redelivery order shows the new
	// priorities were applied.
	retry := map[string]string{"retry": "1"}
	for i, msg := range sent {
		require.NoError(q.t, q.client().Nack(q.name, msg.ID, int64(30-i*10), retry))
	}
	q.expectCounts(QueueCounts{Ready: 3, Total: 3})

	for i := len(sent) - 1; i >= 0; i-- {
		want := sent[i]
		want.Priority = int64(30 - i*10)
		want.Metadata = retry
		q.expectMessage(want, q.dequeue(true))
	}
	q.expectEmpty()
	q.expectCounts(QueueCounts{})
}

// testUpdatePriorityAndDelete: a priority update reorders a waiting message,
// and a deleted message is never delivered.
func testUpdatePriorityAndDelete(q *testQueue) {
	q.create("delayed", nil)
	first := q.enqueue(Message{Priority: 10, Content: "first"})
	deleted := q.enqueue(Message{Priority: 20, Content: "deleted"})
	moved := q.enqueue(Message{Priority: 30, Content: "moved"})

	require.NoError(q.t, q.client().UpdatePriority(q.name, moved.ID, 5))
	moved.Priority = 5
	require.NoError(q.t, q.client().Delete(q.name, deleted.ID))
	_, err := q.client().Get(q.name, deleted.ID)
	q.expectStatus(http.StatusNotFound, err, "getting a deleted message")
	q.expectCounts(QueueCounts{Ready: 2, Total: 2})

	q.expectOrder(true, []Message{moved, first})
	q.expectEmpty()
	q.expectCounts(QueueCounts{})
}

// testQueue is one test's queue, used through one node.
type testQueue struct {
	t         *testing.T
	c         *Cluster
	node      *Node
	name      string
	queueType string
}

func (q *testQueue) client() *Client { return q.node.Client() }

func (q *testQueue) create(queueType string, settings map[string]any) {
	q.t.Helper()
	if settings == nil {
		settings = map[string]any{}
	}
	q.queueType = queueType
	require.NoError(q.t, q.client().CreateQueue(q.name, queueType, settings))
}

// enqueue sends msg, checks the response and what Get returns for it, and
// returns the message as stored, with its ID.
//
// Delayed queues store every message in group "default", whatever group the
// client sent. The enqueue response still echoes the client's group, so the
// group is checked on the stored message only.
func (q *testQueue) enqueue(msg Message) Message {
	q.t.Helper()
	got, err := q.client().EnqueueMessage(q.name, msg)
	require.NoError(q.t, err)
	assert.Equal(q.t, "ENQUEUED", got.Status)
	require.NotEmpty(q.t, got.ID)
	assert.Equal(q.t, msg.Priority, got.Priority)
	assert.Equal(q.t, msg.Content, got.Content)
	assert.Equal(q.t, msg.Metadata, got.Metadata)

	stored, err := q.client().Get(q.name, got.ID)
	require.NoError(q.t, err)
	assert.Equal(q.t, "GOT", stored.Status)
	want := msg
	want.ID = got.ID
	if q.queueType == "delayed" {
		want.Group = "default"
	}
	stored.Status = ""
	assert.Equal(q.t, want, stored, "Get returns what was enqueued")
	return stored
}

// dequeue takes the next message, failing the test if none comes.
//
// It retries briefly while the queue looks empty. A follower checks its own
// replica before proxying a dequeue to the leader, so right after a write
// (an enqueue, or an ack that frees a fair queue group's slot) it can answer
// "empty" until it has applied that write too. On the leader the first try
// succeeds.
func (q *testQueue) dequeue(ack bool) Message {
	q.t.Helper()
	return q.eventuallyDequeue(ack, 2*time.Second)
}

// eventuallyDequeue polls until a message is available or timeout passes.
func (q *testQueue) eventuallyDequeue(ack bool, timeout time.Duration) Message {
	q.t.Helper()
	var msg Message
	require.Eventually(q.t, func() bool {
		var ok bool
		var err error
		msg, ok, err = q.client().Dequeue(q.name, ack)
		require.NoError(q.t, err)
		return ok
	}, timeout, 50*time.Millisecond, "no message in %s within %s", q.name, timeout)
	return msg
}

func (q *testQueue) expectEmpty() {
	q.t.Helper()
	msg, ok, err := q.client().Dequeue(q.name, false)
	require.NoError(q.t, err)
	assert.False(q.t, ok, "expected %s to be empty, got %+v", q.name, msg)
}

// expectOrder dequeues len(want) messages and checks they come in that order.
func (q *testQueue) expectOrder(ack bool, want []Message) {
	q.t.Helper()
	for i, w := range want {
		got := q.dequeue(ack)
		if !q.expectMessage(w, got) {
			q.t.Fatalf("dequeue %d of %d: got %s, want %s", i+1, len(want), got.Content, w.Content)
		}
	}
}

// expectMessage checks that a dequeued message is want.
func (q *testQueue) expectMessage(want, got Message) bool {
	q.t.Helper()
	ok := assert.Equal(q.t, "DEQUEUED", got.Status)
	got.Status, want.Status = "", ""
	return assert.Equal(q.t, want, got) && ok
}

// expectCounts checks that every replica reports these counts.
func (q *testQueue) expectCounts(want QueueCounts) {
	q.t.Helper()
	q.c.WaitForQueueCounts(q.t, q.name, want)
}

func (q *testQueue) expectStatus(code int, err error, msgAndArgs ...any) {
	q.t.Helper()
	var se *StatusError
	if assert.True(q.t, errors.As(err, &se), "expected HTTP %d, got %v", code, err) {
		assert.Equal(q.t, code, se.Code, msgAndArgs...)
	}
}

// pick returns msgs at the given indexes, in that order.
func pick(msgs []Message, indexes ...int) []Message {
	out := make([]Message, len(indexes))
	for i, idx := range indexes {
		out[i] = msgs[idx]
	}
	return out
}
