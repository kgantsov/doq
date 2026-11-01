package memory

import (
	"container/heap"
	"sync"
	"time"
)

type DelayedQueue struct {
	queue *PriorityMemoryQueue
	mu    sync.RWMutex
}

func NewDelayedQueue(minFirst bool) *DelayedQueue {
	return &DelayedQueue{
		queue: NewPriorityMemoryQueue(minFirst),
	}
}

func (pq *DelayedQueue) UpdateMaxUnacked(maxUnacked int) error {
	return nil
}

func (pq *DelayedQueue) Enqueue(group string, item *Item) {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	heap.Push(pq.queue, item)
}

func (pq *DelayedQueue) Dequeue(ack bool, seed uint64, now time.Time) *Item {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	if pq.queue.Len() == 0 {
		return nil
	}

	queueItem := pq.queue.Peek().(*Item)

	if int64(queueItem.Priority) > now.UTC().Unix() {
		// Delayed message handling
		// If a priority is in the future, return nil
		return nil
	}

	return heap.Pop(pq.queue).(*Item)
}

// PeekReady reports whether the head message is deliverable now, using the same
// scheduled-time gating as Dequeue but without popping. When the head is
// scheduled in the future, nextReadyIn is the time until it matures.
func (pq *DelayedQueue) PeekReady() (bool, time.Duration) {
	pq.mu.RLock()
	defer pq.mu.RUnlock()

	if pq.queue.Len() == 0 {
		return false, 0
	}

	head := pq.queue.Peek().(*Item)
	now := time.Now().UTC().Unix()
	if head.Priority > now {
		return false, time.Duration(head.Priority-now) * time.Second
	}
	return true, 0
}

// PeekReadyItem returns the head item if it is deliverable now, without popping
// it. It returns nil when the queue is empty or the head is scheduled in the
// future.
func (pq *DelayedQueue) PeekReadyItem() *Item {
	pq.mu.RLock()
	defer pq.mu.RUnlock()

	if pq.queue.Len() == 0 {
		return nil
	}

	head := pq.queue.Peek().(*Item)
	if head.Priority > time.Now().UTC().Unix() {
		return nil
	}
	return head
}

func (pq *DelayedQueue) Get(group string, id uint64) *Item {
	pq.mu.RLock()
	defer pq.mu.RUnlock()

	return pq.queue.Get(id)
}

func (pq *DelayedQueue) Delete(group string, id uint64) *Item {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	return pq.queue.Delete(id)
}

func (pq *DelayedQueue) UpdatePriority(group string, id uint64, priority int64) {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	pq.queue.UpdatePriority(id, priority)
}

func (pq *DelayedQueue) Len() uint64 {
	pq.mu.RLock()
	defer pq.mu.RUnlock()

	return uint64(pq.queue.Len())
}

func (pq *DelayedQueue) UpdateWeights(group string, id uint64) error {

	return nil
}

func (pq *DelayedQueue) MarkUnacked(group string) {}

// Items returns a copy of every item in the queue, in no particular order.
func (pq *DelayedQueue) Items() []Item {
	pq.mu.RLock()
	defer pq.mu.RUnlock()

	items := make([]Item, 0, len(pq.queue.items))
	for _, item := range pq.queue.items {
		items = append(items, *item)
	}
	return items
}
