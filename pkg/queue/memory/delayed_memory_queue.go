package memory

import (
	"container/heap"
	"sync"
	"time"
)

type DelayedMemoryQueue struct {
	queue *PriorityMemoryQueue
	mu    sync.RWMutex
}

func NewDelayedMemoryQueue(minFirst bool) *DelayedMemoryQueue {
	return &DelayedMemoryQueue{
		queue: NewPriorityMemoryQueue(minFirst),
	}
}

func (pq *DelayedMemoryQueue) Enqueue(group string, item *Item) {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	heap.Push(pq.queue, item)
}

func (pq *DelayedMemoryQueue) Dequeue() *Item {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	if pq.queue.Len() == 0 {
		return nil
	}

	queueItem := pq.queue.Peek().(*Item)

	if int64(queueItem.Priority) > time.Now().UTC().Unix() {
		// Delayed message handling
		// If a priority is in the future, return nil
		return nil
	}

	return heap.Pop(pq.queue).(*Item)
}

func (pq *DelayedMemoryQueue) Get(group string, id uint64) *Item {
	pq.mu.RLock()
	defer pq.mu.RUnlock()

	return pq.queue.Get(id)
}

func (pq *DelayedMemoryQueue) Delete(group string, id uint64) *Item {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	return pq.queue.Delete(id)
}

func (pq *DelayedMemoryQueue) UpdatePriority(group string, id uint64, priority int64) {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	pq.queue.UpdatePriority(id, priority)
}

func (pq *DelayedMemoryQueue) Len() uint64 {
	pq.mu.RLock()
	defer pq.mu.RUnlock()

	return uint64(pq.queue.Len())
}
