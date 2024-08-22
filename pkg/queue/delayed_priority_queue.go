package queue

import (
	"container/heap"
	"time"
)

type DelayedPriorityQueue struct {
	queue *PriorityQueue
}

func NewDelayedPriorityQueue(minFirst bool) *DelayedPriorityQueue {
	return &DelayedPriorityQueue{
		queue: NewPriorityQueue(minFirst),
	}
}

func (pq *DelayedPriorityQueue) Enqueue(group string, item *Item) {
	heap.Push(pq.queue, item)
}

func (pq *DelayedPriorityQueue) Dequeue() *Item {
	queueItem := pq.queue.Peek().(*Item)

	if int64(queueItem.Priority) > time.Now().UTC().Unix() {
		// Delayed message handling
		// If a priority is in the future, return nil
		return nil
	}

	return heap.Pop(pq.queue).(*Item)
}

func (pq *DelayedPriorityQueue) GetByID(group string, id uint64) *Item {
	return pq.queue.GetByID(id)
}

func (pq *DelayedPriorityQueue) DeleteByID(group string, id uint64) *Item {
	return pq.queue.DeleteByID(id)
}

func (pq *DelayedPriorityQueue) UpdatePriority(group string, id uint64, priority int64) {
	pq.queue.UpdatePriority(id, priority)
}

func (pq *DelayedPriorityQueue) Len() uint64 {
	return uint64(pq.queue.Len())
}
