package queue

import (
	"container/heap"
	"sync"
	"time"
)

type DelayedPriorityQueue struct {
	queue *PriorityQueue
	mu    sync.RWMutex
}

func NewDelayedPriorityQueue(minFirst bool) *DelayedPriorityQueue {
	return &DelayedPriorityQueue{
		queue: NewPriorityQueue(minFirst),
	}
}

func (pq *DelayedPriorityQueue) Enqueue(group string, item *Item) {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	heap.Push(pq.queue, item)
}

func (pq *DelayedPriorityQueue) Dequeue() *Item {
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

func (pq *DelayedPriorityQueue) GetByID(group string, id uint64) *Item {
	pq.mu.RLock()
	defer pq.mu.RUnlock()

	return pq.queue.GetByID(id)
}

func (pq *DelayedPriorityQueue) DeleteByID(group string, id uint64) *Item {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	return pq.queue.DeleteByID(id)
}

func (pq *DelayedPriorityQueue) UpdatePriority(group string, id uint64, priority int64) {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	pq.queue.UpdatePriority(id, priority)
}

func (pq *DelayedPriorityQueue) Len() uint64 {
	pq.mu.RLock()
	defer pq.mu.RUnlock()

	return uint64(pq.queue.Len())
}
