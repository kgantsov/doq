package memory

import "time"

type MemoryQueue interface {
	UpdateMaxUnacked(maxUnacked int) error
	Enqueue(group string, item *Item)
	Dequeue(ack bool) *Item
	Get(group string, id uint64) *Item
	Delete(group string, id uint64) *Item
	UpdatePriority(group string, id uint64, priority int64)
	Len() uint64
	UpdateWeights(group string, id uint64) error
	// PeekReady reports, without mutating the queue, whether a message can be
	// dequeued right now. When nothing is ready, nextReadyIn is the duration
	// until the head becomes deliverable if that is time-based and known
	// (delayed queues); it is 0 when the queue is empty or readiness depends on
	// an external event (e.g. an ack freeing an unacked slot).
	PeekReady() (ready bool, nextReadyIn time.Duration)
}
