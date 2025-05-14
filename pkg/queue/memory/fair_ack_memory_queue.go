package memory

import (
	"container/heap"
	"sync"

	avl "github.com/kgantsov/doq/pkg/weighted_avl"
)

type FairAckMemoryQueue struct {
	unackedByGroup *avl.WeightedAVL
	queues         map[string]*PriorityMemoryQueue
	mu             sync.Mutex
}

func NewFairAckMemoryQueue(unackedByGroup *avl.WeightedAVL) *FairAckMemoryQueue {
	return &FairAckMemoryQueue{
		unackedByGroup: unackedByGroup,
		queues:         make(map[string]*PriorityMemoryQueue),
	}
}

func (q *FairAckMemoryQueue) Enqueue(group string, item *Item) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if _, ok := q.queues[group]; !ok {
		q.queues[group] = NewPriorityMemoryQueue(true)
		q.unackedByGroup.Update(group, 1)
	} else {
		q.unackedByGroup.Increment(group, 1)
	}

	heap.Push(q.queues[group], item)
}

func (q *FairAckMemoryQueue) Dequeue() *Item {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.queues) == 0 {
		return nil
	}

	var selectedGroup string

	selectedGroup = q.unackedByGroup.Sample()
	if selectedGroup == "" {
		return nil
	}

	item := heap.Pop(q.queues[selectedGroup]).(*Item)
	if item == nil {
		return nil
	}

	if q.queues[selectedGroup].Len() == 0 {
		delete(q.queues, selectedGroup)
		q.unackedByGroup.Remove(selectedGroup)
	}

	return item
}

func (q *FairAckMemoryQueue) Get(group string, id uint64) *Item {
	q.mu.Lock()
	defer q.mu.Unlock()

	if _, ok := q.queues[group]; !ok {
		return nil
	}

	return q.queues[group].Get(id)
}

func (q *FairAckMemoryQueue) Delete(group string, id uint64) *Item {
	q.mu.Lock()
	defer q.mu.Unlock()

	if _, ok := q.queues[group]; !ok {
		return nil
	}

	item := q.queues[group].Delete(id)
	if item == nil {
		return nil
	}

	q.unackedByGroup.Increment(group, -1)

	if q.queues[group].Len() == 0 {
		delete(q.queues, group)
		q.unackedByGroup.Remove(group)
	}

	return item
}

func (q *FairAckMemoryQueue) UpdatePriority(group string, id uint64, priority int64) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if _, ok := q.queues[group]; !ok {
		return
	}

	q.queues[group].UpdatePriority(id, priority)
}

func (q *FairAckMemoryQueue) Len() uint64 {
	q.mu.Lock()
	defer q.mu.Unlock()

	var total uint64
	for _, queue := range q.queues {
		total += uint64(queue.Len())
	}
	return total
}
