package queue

import (
	"testing"
	"time"

	"github.com/kgantsov/doq/pkg/entity"
	"github.com/kgantsov/doq/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestQueueEnqueueUnackedIsIgnored enqueues an id that is in flight. Before the
// fix the message became ready again, a second consumer got it, and the ack
// queue held two entries for one id. Now the enqueue is a no-op.
func TestQueueEnqueueUnackedIsIgnored(t *testing.T) {
	tests := []struct {
		name      string
		queueType string
		settings  entity.QueueSettings
	}{
		{name: "delayed", queueType: "delayed", settings: entity.QueueSettings{AckTimeout: 60}},
		{
			name:      "fair round_robin",
			queueType: "fair",
			settings:  entity.QueueSettings{Strategy: "round_robin", MaxUnacked: 1, AckTimeout: 60},
		},
		{
			name:      "fair weighted",
			queueType: "fair",
			settings:  entity.QueueSettings{Strategy: "weighted", MaxUnacked: 1, AckTimeout: 60},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pq := newDeliveryLimitQueue(t, tt.queueType, tt.settings)

			_, err := pq.Enqueue(1, "g1", 10, "original", map[string]string{"k": "v"})
			require.NoError(t, err)

			m, err := pq.Dequeue(false, 0, time.Now())
			require.NoError(t, err)
			require.Equal(t, uint64(1), m.ID)

			// The re-enqueue returns the stored message untouched.
			m, err = pq.Enqueue(1, "g1", 20, "copy", nil)
			require.NoError(t, err)
			assert.True(t, m.AlreadyExists)
			assert.Equal(t, uint64(1), m.ID)
			assert.Equal(t, "original", m.Content)
			assert.Equal(t, map[string]string{"k": "v"}, m.Metadata)
			assert.Equal(t, int64(10), m.Priority)
			assert.Equal(t, uint32(1), m.DeliveryCount)

			// It is not delivered again.
			_, err = pq.Dequeue(false, 0, time.Now())
			require.ErrorIs(t, err, errors.ErrEmptyQueue)

			stats := pq.GetStats()
			assert.Equal(t, int64(0), stats.Ready)
			assert.Equal(t, int64(1), stats.Unacked)

			require.NoError(t, pq.Touch(1, time.Now()))
			require.NoError(t, pq.Ack(1))

			_, err = pq.Get(1)
			require.ErrorIs(t, err, errors.ErrMessageNotFound)
			assert.Equal(t, int64(0), pq.GetStats().Unacked)

			// One delivery took one unacked slot, and the ack released it: the
			// group, at MaxUnacked 1 for fair queues, is still usable.
			_, err = pq.Enqueue(2, "g1", 10, "next", nil)
			require.NoError(t, err)

			m, err = pq.Dequeue(false, 0, time.Now())
			require.NoError(t, err, "group starved after re-enqueue of an unacked id")
			assert.Equal(t, uint64(2), m.ID)
		})
	}
}

// TestQueueEnqueueUnackedKeepsAckTimeouts re-enqueues an in-flight id and then
// lets every ack deadline pass. Each message must be redelivered exactly once,
// and the re-enqueued one must not hold up the others.
func TestQueueEnqueueUnackedKeepsAckTimeouts(t *testing.T) {
	pq := newDeliveryLimitQueue(t, "delayed", entity.QueueSettings{AckTimeout: 60})
	pq.ackTimeoutHandler = func(queueName string, id uint64) error {
		// Apply the expiry the way the FSM does once Raft commits it.
		pq.AckTimeout(id)
		return nil
	}

	for _, id := range []uint64{1, 2, 3} {
		_, err := pq.Enqueue(id, "default", 10, "msg", nil)
		require.NoError(t, err)
		_, err = pq.Dequeue(false, 0, time.Now())
		require.NoError(t, err)
	}

	_, err := pq.Enqueue(1, "default", 10, "msg", nil)
	require.NoError(t, err)

	for _, id := range []uint64{1, 2, 3} {
		pq.ackQueue.UpdatePriority("default", id, time.Now().UTC().Unix()-1)
	}
	pq.expireAcks()

	stats := pq.GetStats()
	assert.Equal(t, int64(3), stats.Ready)
	assert.Equal(t, int64(0), stats.Unacked)

	var ids []uint64
	for range 3 {
		m, err := pq.Dequeue(true, 0, time.Now())
		require.NoError(t, err)
		ids = append(ids, m.ID)
	}
	assert.ElementsMatch(t, []uint64{1, 2, 3}, ids)

	_, err = pq.Dequeue(true, 0, time.Now())
	require.ErrorIs(t, err, errors.ErrEmptyQueue)
}

// TestQueueEnqueueReadyIsIgnored re-enqueues an id that is on the ready queue.
// Nothing changes. Before the fix the store took the new content and priority
// while the ready queue kept the old priority.
func TestQueueEnqueueReadyIsIgnored(t *testing.T) {
	tests := []struct {
		name      string
		queueType string
		settings  entity.QueueSettings
	}{
		{name: "delayed", queueType: "delayed", settings: entity.QueueSettings{AckTimeout: 60}},
		{
			name:      "fair round_robin",
			queueType: "fair",
			settings:  entity.QueueSettings{Strategy: "round_robin", AckTimeout: 60},
		},
		{
			name:      "fair weighted",
			queueType: "fair",
			settings:  entity.QueueSettings{Strategy: "weighted", AckTimeout: 60},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pq := newDeliveryLimitQueue(t, tt.queueType, tt.settings)

			m, err := pq.Enqueue(1, "g1", 10, "original", map[string]string{"k": "v"})
			require.NoError(t, err)
			assert.False(t, m.AlreadyExists)
			_, err = pq.Enqueue(2, "g1", 20, "other", nil)
			require.NoError(t, err)

			m, err = pq.Enqueue(1, "g1", 30, "copy", nil)
			require.NoError(t, err)
			assert.True(t, m.AlreadyExists)
			assert.Equal(t, "original", m.Content)
			assert.Equal(t, map[string]string{"k": "v"}, m.Metadata)
			assert.Equal(t, int64(10), m.Priority)

			assert.Equal(t, int64(2), pq.GetStats().Ready)

			m, err = pq.Dequeue(true, 0, time.Now())
			require.NoError(t, err)
			assert.Equal(t, uint64(1), m.ID)
			assert.Equal(t, "original", m.Content)

			m, err = pq.Dequeue(true, 0, time.Now())
			require.NoError(t, err)
			assert.Equal(t, uint64(2), m.ID)

			_, err = pq.Dequeue(true, 0, time.Now())
			require.ErrorIs(t, err, errors.ErrEmptyQueue)
		})
	}
}

// TestQueueEnqueueReadyKeepsDeliveryCount re-enqueues a message that was nacked
// back to ready. Before the fix the re-enqueue reset DeliveryCount to 0, so a
// message could dodge DeliveryLimit forever.
func TestQueueEnqueueReadyKeepsDeliveryCount(t *testing.T) {
	pq := newDeliveryLimitQueue(t, "delayed", entity.QueueSettings{AckTimeout: 60, DeliveryLimit: 2})

	_, err := pq.Enqueue(1, "default", 10, "msg", nil)
	require.NoError(t, err)
	_, err = pq.Dequeue(false, 0, time.Now())
	require.NoError(t, err)
	require.NoError(t, pq.Nack(1, 0, nil))

	m, err := pq.Enqueue(1, "default", 10, "msg", nil)
	require.NoError(t, err)
	assert.True(t, m.AlreadyExists)
	assert.Equal(t, uint32(1), m.DeliveryCount)

	m, err = pq.Dequeue(false, 0, time.Now())
	require.NoError(t, err)
	assert.Equal(t, uint32(2), m.DeliveryCount)

	// The second failure reaches the limit, so the message is dropped.
	require.NoError(t, pq.Nack(1, 0, nil))
	_, err = pq.Get(1)
	require.ErrorIs(t, err, errors.ErrMessageNotFound)
}

// TestQueueEnqueueReadyOtherGroupIsIgnored re-enqueues a ready id under another
// group of a fair queue. Before the fix the duplicate check looked only in the
// new group, so the message ended up on the ready queue twice.
func TestQueueEnqueueReadyOtherGroupIsIgnored(t *testing.T) {
	for _, strategy := range []string{"round_robin", "weighted"} {
		t.Run(strategy, func(t *testing.T) {
			pq := newDeliveryLimitQueue(t, "fair", entity.QueueSettings{Strategy: strategy, AckTimeout: 60})

			_, err := pq.Enqueue(1, "g1", 10, "msg", nil)
			require.NoError(t, err)

			m, err := pq.Enqueue(1, "g2", 10, "msg", nil)
			require.NoError(t, err)
			assert.True(t, m.AlreadyExists)
			assert.Equal(t, "g1", m.Group)
			assert.Equal(t, int64(1), pq.GetStats().Ready)

			m, err = pq.Dequeue(true, 0, time.Now())
			require.NoError(t, err)
			assert.Equal(t, uint64(1), m.ID)
			assert.Equal(t, "g1", m.Group)

			_, err = pq.Dequeue(true, 0, time.Now())
			require.ErrorIs(t, err, errors.ErrEmptyQueue)
		})
	}
}
