package memory

import "time"

type MemoryQueue interface {
	UpdateMaxUnacked(maxUnacked int) error
	Enqueue(group string, item *Item)
	// Dequeue removes and returns the next message. seed is the only source of
	// randomness for strategies that sample (weighted fair queues), and now is
	// the only clock for strategies that gate on time (delayed queues): the same
	// seed and now on the same state must return the same message. Under Raft
	// they are the log index and the leader's append time, so replicas and log
	// replays stay identical.
	Dequeue(ack bool, seed uint64, now time.Time) *Item
	Get(group string, id uint64) *Item
	Delete(group string, id uint64) *Item
	UpdatePriority(group string, id uint64, priority int64)
	Len() uint64
	UpdateWeights(group string, id uint64) error
	// MarkUnacked records one more in-flight (unacked) message for group, as a
	// Dequeue(ack=false) would, without removing anything from the queue. It is
	// used when restoring unacked messages from a snapshot.
	MarkUnacked(group string)
	// PeekReady reports, without mutating the queue, whether a message can be
	// dequeued right now. When nothing is ready, nextReadyIn is the duration
	// until the head becomes deliverable if that is time-based and known
	// (delayed queues); it is 0 when the queue is empty or readiness depends on
	// an external event (e.g. an ack freeing an unacked slot).
	PeekReady() (ready bool, nextReadyIn time.Duration)
}
