package queue

import (
	"container/heap"
	"sync"
	"time"
)

type DelayedQueue struct {
	queue *PriorityQueue
	mu    sync.RWMutex
}

func NewDelayedQueue(minFirst bool) *DelayedQueue {
	return &DelayedQueue{
		queue: NewPriorityQueue(minFirst),
	}
}

func (pq *DelayedQueue) Enqueue(group string, item *Item) {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	heap.Push(pq.queue, item)
}

func (pq *DelayedQueue) Dequeue() *Item {
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
