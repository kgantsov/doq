package queue

import (
	"testing"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/kgantsov/doq/pkg/config"
	"github.com/kgantsov/doq/pkg/entity"
	"github.com/kgantsov/doq/pkg/errors"
	"github.com/kgantsov/doq/pkg/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newDeliveryLimitQueue(t *testing.T, queueType string, settings entity.QueueSettings) *Queue {
	t.Helper()

	db, err := badger.Open(badger.DefaultOptions("").WithInMemory(true))
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	pq := NewQueue(
		storage.NewBadgerStore(db),
		&config.Config{Queue: config.QueueConfig{
			AcknowledgementCheckInterval: 1,
			QueueStats:                   config.QueueStatsConfig{WindowSide: 10},
		}},
		nil,
	)
	require.NoError(t, pq.CreateQueue(queueType, "test_queue", settings))
	pq.StopAckQueueMonitoring() // drive the ack sweep by hand

	return pq
}

// TestQueueDeliveryCountCountsDeliveries checks that DeliveryCount is truthful:
// it is 1 while the first delivery is in flight, not 0, and it moves on delivery
// rather than on failure.
func TestQueueDeliveryCountCountsDeliveries(t *testing.T) {
	pq := newDeliveryLimitQueue(t, "delayed", entity.QueueSettings{AckTimeout: 60})

	m, err := pq.Enqueue(1, "default", 10, "test 1", nil)
	require.NoError(t, err)
	assert.Equal(t, uint32(0), m.DeliveryCount, "an enqueued message has not been delivered yet")

	m, err = pq.Dequeue(false, 0, time.Now())
	require.NoError(t, err)
	assert.Equal(t, uint32(1), m.DeliveryCount, "first delivery must report 1")

	// The count is durable, not just a field on the returned copy.
	stored, err := pq.Get(1)
	require.NoError(t, err)
	assert.Equal(t, uint32(1), stored.DeliveryCount)

	// A nack does not move the counter; the next delivery does.
	require.NoError(t, pq.Nack(1, 0, nil))
	stored, err = pq.Get(1)
	require.NoError(t, err)
	assert.Equal(t, uint32(1), stored.DeliveryCount, "nack must not count as a delivery")

	m, err = pq.Dequeue(false, 0, time.Now())
	require.NoError(t, err)
	assert.Equal(t, uint32(2), m.DeliveryCount, "second delivery must report 2")
}

// TestQueueDeliveryCountAutoAck checks that an auto-acked dequeue still reports
// the delivery it just made, even though the message is deleted in the same
// transaction.
func TestQueueDeliveryCountAutoAck(t *testing.T) {
	pq := newDeliveryLimitQueue(t, "delayed", entity.QueueSettings{AckTimeout: 60})

	_, err := pq.Enqueue(1, "default", 10, "test 1", nil)
	require.NoError(t, err)

	m, err := pq.Dequeue(true, 0, time.Now())
	require.NoError(t, err)
	assert.Equal(t, uint32(1), m.DeliveryCount)

	_, err = pq.Get(1)
	assert.ErrorIs(t, err, errors.ErrMessageNotFound)
}

// TestQueueDeliveryLimitUnlimited guards the default: DeliveryLimit 0 means
// unlimited, so a message survives any number of failed deliveries.
func TestQueueDeliveryLimitUnlimited(t *testing.T) {
	pq := newDeliveryLimitQueue(t, "delayed", entity.QueueSettings{AckTimeout: 60})

	_, err := pq.Enqueue(1, "default", 10, "test 1", nil)
	require.NoError(t, err)

	for i := 0; i < 5; i++ {
		m, err := pq.Dequeue(false, 0, time.Now())
		require.NoErrorf(t, err, "message gone after %d failed deliveries", i)
		require.Equal(t, uint32(i+1), m.DeliveryCount)
		require.NoError(t, pq.Nack(1, 0, nil))
	}

	stored, err := pq.Get(1)
	require.NoError(t, err)
	assert.Equal(t, uint32(5), stored.DeliveryCount)
	assert.Equal(t, 1, pq.Len())
}

// TestQueueDeliveryLimitDropsOnNack is the core table: with DeliveryLimit N a
// message is delivered N times and dropped on the Nth failure.
func TestQueueDeliveryLimitDropsOnNack(t *testing.T) {
	for _, limit := range []uint32{1, 2, 3} {
		t.Run(string(rune('0'+limit)), func(t *testing.T) {
			pq := newDeliveryLimitQueue(t, "delayed", entity.QueueSettings{
				AckTimeout:    60,
				DeliveryLimit: limit,
			})

			_, err := pq.Enqueue(1, "default", 10, "test 1", nil)
			require.NoError(t, err)

			// The first limit-1 failures redeliver.
			for i := uint32(0); i < limit-1; i++ {
				m, err := pq.Dequeue(false, 0, time.Now())
				require.NoErrorf(t, err, "dropped early on delivery %d", i+1)
				require.Equal(t, uint32(i+1), m.DeliveryCount)
				require.NoError(t, pq.Nack(1, 0, nil))
				require.Equal(t, 1, pq.Len(), "redelivered message must be back on the ready queue")
			}

			// The limit-th failure drops it.
			m, err := pq.Dequeue(false, 0, time.Now())
			require.NoError(t, err)
			require.Equal(t, limit, m.DeliveryCount)
			require.NoError(t, pq.Nack(1, 0, nil))

			_, err = pq.Get(1)
			assert.ErrorIs(t, err, errors.ErrMessageNotFound, "dropped message must be gone from storage")
			assert.Equal(t, 0, pq.Len(), "dropped message must not be back on the ready queue")
			assert.Nil(t, pq.ackQueue.Get("default", 1), "dropped message must leave the ack queue")

			stats := pq.GetStats()
			assert.Equal(t, int64(0), stats.Ready)
			assert.Equal(t, int64(0), stats.Unacked)
			assert.Equal(t, int64(0), stats.Total)
		})
	}
}

// TestQueueDeliveryLimitDropsOnAckTimeout exercises the other failure path: a
// consumer that never acks must exhaust the limit too, or the limit would only
// bind consumers that politely nack.
func TestQueueDeliveryLimitDropsOnAckTimeout(t *testing.T) {
	pq := newDeliveryLimitQueue(t, "delayed", entity.QueueSettings{
		AckTimeout:    60,
		DeliveryLimit: 2,
	})

	_, err := pq.Enqueue(1, "default", 10, "test 1", nil)
	require.NoError(t, err)

	// First delivery times out and is redelivered.
	m, err := pq.Dequeue(false, 0, time.Now())
	require.NoError(t, err)
	require.Equal(t, uint32(1), m.DeliveryCount)
	require.NoError(t, pq.AckTimeout(1))
	assert.Equal(t, 1, pq.Len())

	// Second delivery times out and is dropped.
	m, err = pq.Dequeue(false, 0, time.Now())
	require.NoError(t, err)
	require.Equal(t, uint32(2), m.DeliveryCount)
	require.NoError(t, pq.AckTimeout(1))

	_, err = pq.Get(1)
	assert.ErrorIs(t, err, errors.ErrMessageNotFound)
	assert.Equal(t, 0, pq.Len())
	assert.Nil(t, pq.ackQueue.Get("default", 1))
}

// TestQueueDeliveryLimitDropThroughAckSweep drives the real ack sweep rather than
// calling AckTimeout directly, and checks it makes progress past a dropped
// message instead of spinning on it as the queue head.
func TestQueueDeliveryLimitDropThroughAckSweep(t *testing.T) {
	pq := newDeliveryLimitQueue(t, "delayed", entity.QueueSettings{
		AckTimeout:    60,
		DeliveryLimit: 1,
	})

	var handled []uint64
	pq.ackTimeoutHandler = func(queueName string, id uint64) error {
		handled = append(handled, id)
		return pq.AckTimeout(id)
	}

	_, err := pq.Enqueue(1, "default", 10, "test 1", nil)
	require.NoError(t, err)

	m, err := pq.Dequeue(false, 0, time.Now())
	require.NoError(t, err)
	require.Equal(t, uint32(1), m.DeliveryCount)

	pq.ackQueue.UpdatePriority("default", m.ID, time.Now().UTC().Unix()-1)
	pq.expireAcks()

	assert.Equal(t, []uint64{m.ID}, handled)
	_, err = pq.Get(1)
	assert.ErrorIs(t, err, errors.ErrMessageNotFound)
	assert.Equal(t, uint64(0), pq.ackQueue.Len(), "the sweep must not leave the dropped message behind")
}

// TestQueueDeliveryLimitAckBeforeLimit checks the happy path is untouched: a
// message that fails once and then succeeds is acked normally and leaves no
// state behind.
func TestQueueDeliveryLimitAckBeforeLimit(t *testing.T) {
	pq := newDeliveryLimitQueue(t, "delayed", entity.QueueSettings{
		AckTimeout:    60,
		DeliveryLimit: 3,
	})

	_, err := pq.Enqueue(1, "default", 10, "test 1", nil)
	require.NoError(t, err)

	_, err = pq.Dequeue(false, 0, time.Now())
	require.NoError(t, err)
	require.NoError(t, pq.Nack(1, 0, nil))

	m, err := pq.Dequeue(false, 0, time.Now())
	require.NoError(t, err)
	require.Equal(t, uint32(2), m.DeliveryCount)
	require.NoError(t, pq.Ack(1))

	_, err = pq.Get(1)
	assert.ErrorIs(t, err, errors.ErrMessageNotFound)
	stats := pq.GetStats()
	assert.Equal(t, int64(0), stats.Total)
}

// TestQueueDeliveryLimitNackPreservesPriorityAndMetadata checks that a
// redelivered message still gets the caller's priority and metadata, and that a
// dropped one is not resurrected by that same write.
func TestQueueDeliveryLimitNackPreservesPriorityAndMetadata(t *testing.T) {
	pq := newDeliveryLimitQueue(t, "delayed", entity.QueueSettings{
		AckTimeout:    60,
		DeliveryLimit: 2,
	})

	_, err := pq.Enqueue(1, "default", 10, "test 1", map[string]string{"retry": "0"})
	require.NoError(t, err)

	_, err = pq.Dequeue(false, 0, time.Now())
	require.NoError(t, err)
	require.NoError(t, pq.Nack(1, 30, map[string]string{"retry": "1"}))

	stored, err := pq.Get(1)
	require.NoError(t, err)
	assert.Equal(t, int64(30), stored.Priority)
	assert.Equal(t, map[string]string{"retry": "1"}, stored.Metadata)
	assert.Equal(t, uint32(1), stored.DeliveryCount, "the metadata write must not clobber the count")

	// The failure that exhausts the limit carries metadata too: the message must
	// still be gone, not rewritten as a partial record under its key.
	_, err = pq.Dequeue(false, 0, time.Now())
	require.NoError(t, err)
	require.NoError(t, pq.Nack(1, 40, map[string]string{"retry": "2"}))

	_, err = pq.Get(1)
	assert.ErrorIs(t, err, errors.ErrMessageNotFound)
}

// TestQueueDeliveryLimitFairReleasesUnackedSlot drops the last in-flight message
// of a group and then uses that group again. A drop takes the message out of the
// in-flight state exactly as a redelivery does, so it must release the group's
// unacked slot — otherwise the counter leaks, the group's weight pins to 0 and
// it is starved forever ("No group selected").
func TestQueueDeliveryLimitFairReleasesUnackedSlot(t *testing.T) {
	for _, strategy := range []string{"round_robin", "weighted"} {
		t.Run(strategy, func(t *testing.T) {
			pq := newDeliveryLimitQueue(t, "fair", entity.QueueSettings{
				Strategy:      strategy,
				MaxUnacked:    1,
				AckTimeout:    60,
				DeliveryLimit: 1,
			})

			_, err := pq.Enqueue(1, "g1", 10, "poison", nil)
			require.NoError(t, err)

			m, err := pq.Dequeue(false, 0, time.Now())
			require.NoError(t, err)
			require.Equal(t, uint64(1), m.ID)

			// Exhausts the limit on its first failure, so it is dropped.
			require.NoError(t, pq.Nack(1, 0, nil))
			_, err = pq.Get(1)
			require.ErrorIs(t, err, errors.ErrMessageNotFound)

			// The group is at MaxUnacked 1, so this only works if the drop
			// released the slot.
			_, err = pq.Enqueue(2, "g1", 10, "next", nil)
			require.NoError(t, err)

			ready, _ := pq.PeekReady()
			require.True(t, ready, "group starved after a drop (unacked leaked)")

			m, err = pq.Dequeue(false, 0, time.Now())
			require.NoError(t, err, "group starved after a drop (unacked leaked)")
			require.Equal(t, uint64(2), m.ID)
		})
	}
}

// TestQueueDeliveryLimitFairDropOnAckTimeoutReleasesUnackedSlot is the same
// check for the ack-timeout path, which removes the message from the ack queue
// on a different schedule than Nack does.
func TestQueueDeliveryLimitFairDropOnAckTimeoutReleasesUnackedSlot(t *testing.T) {
	for _, strategy := range []string{"round_robin", "weighted"} {
		t.Run(strategy, func(t *testing.T) {
			pq := newDeliveryLimitQueue(t, "fair", entity.QueueSettings{
				Strategy:      strategy,
				MaxUnacked:    1,
				AckTimeout:    60,
				DeliveryLimit: 1,
			})

			_, err := pq.Enqueue(1, "g1", 10, "poison", nil)
			require.NoError(t, err)

			_, err = pq.Dequeue(false, 0, time.Now())
			require.NoError(t, err)

			require.NoError(t, pq.AckTimeout(1))
			_, err = pq.Get(1)
			require.ErrorIs(t, err, errors.ErrMessageNotFound)

			_, err = pq.Enqueue(2, "g1", 10, "next", nil)
			require.NoError(t, err)

			m, err := pq.Dequeue(false, 0, time.Now())
			require.NoError(t, err, "group starved after a drop (unacked leaked)")
			require.Equal(t, uint64(2), m.ID)
		})
	}
}

// TestQueueDeliveryLimitLoweredOnLiveQueue checks that UpdateQueue's new limit
// binds messages that are already in flight.
func TestQueueDeliveryLimitLoweredOnLiveQueue(t *testing.T) {
	pq := newDeliveryLimitQueue(t, "delayed", entity.QueueSettings{AckTimeout: 60})

	_, err := pq.Enqueue(1, "default", 10, "test 1", nil)
	require.NoError(t, err)

	for i := 0; i < 3; i++ {
		_, err = pq.Dequeue(false, 0, time.Now())
		require.NoError(t, err)
		require.NoError(t, pq.Nack(1, 0, nil))
	}

	stored, err := pq.Get(1)
	require.NoError(t, err)
	require.Equal(t, uint32(3), stored.DeliveryCount)

	require.NoError(t, pq.UpdateQueue(entity.QueueSettings{AckTimeout: 60, DeliveryLimit: 2}))

	// Already over the new limit, so the next failure drops it.
	_, err = pq.Dequeue(false, 0, time.Now())
	require.NoError(t, err)
	require.NoError(t, pq.Nack(1, 0, nil))

	_, err = pq.Get(1)
	assert.ErrorIs(t, err, errors.ErrMessageNotFound)
}

// TestQueueDeliveryLimitFairDropReleasesExactlyOneSlot pins the drop path to a
// single unacked release. Releasing twice is invisible when the dropped message
// is the group's last one — UpdateWeights early-returns once the group is gone —
// so this keeps other messages in the group and checks the counter lands on the
// right number rather than merely a non-starving one.
func TestQueueDeliveryLimitFairDropReleasesExactlyOneSlot(t *testing.T) {
	for _, strategy := range []string{"round_robin", "weighted"} {
		t.Run(strategy, func(t *testing.T) {
			pq := newDeliveryLimitQueue(t, "fair", entity.QueueSettings{
				Strategy:      strategy,
				MaxUnacked:    2,
				AckTimeout:    60,
				DeliveryLimit: 1,
			})

			for id := uint64(1); id <= 4; id++ {
				_, err := pq.Enqueue(id, "g1", 10, "msg", nil)
				require.NoError(t, err)
			}

			// Fill the group's two unacked slots.
			first, err := pq.Dequeue(false, 0, time.Now())
			require.NoError(t, err)
			_, err = pq.Dequeue(false, 0, time.Now())
			require.NoError(t, err)

			// At MaxUnacked, so the group is not deliverable.
			_, err = pq.Dequeue(false, 0, time.Now())
			require.ErrorIs(t, err, errors.ErrEmptyQueue)

			// Drop one in-flight message: that frees exactly one slot.
			require.NoError(t, pq.Nack(first.ID, 0, nil))
			_, err = pq.Get(first.ID)
			require.ErrorIs(t, err, errors.ErrMessageNotFound)

			// One delivery fits in the freed slot...
			_, err = pq.Dequeue(false, 0, time.Now())
			require.NoError(t, err, "drop did not release the group's unacked slot")

			// ...and no more. A second success here means the drop released two
			// slots and the group is now over its unacked limit.
			_, err = pq.Dequeue(false, 0, time.Now())
			require.ErrorIs(
				t, err, errors.ErrEmptyQueue,
				"drop released more than one unacked slot, so the group exceeded MaxUnacked",
			)
		})
	}
}
