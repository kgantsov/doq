package memory

import (
	"container/heap"
	"fmt"
)

type Item struct {
	ID       uint64
	Priority int64
	Group    string
}

func (m *Item) UpdatePriority(newPriority int64) {
	m.Priority = newPriority
}

type PriorityMemoryQueue struct {
	minFirst  bool
	items     []*Item
	idToIndex map[uint64]int
}

func NewPriorityMemoryQueue(minFirst bool) *PriorityMemoryQueue {
	return &PriorityMemoryQueue{
		items:     []*Item{},
		idToIndex: make(map[uint64]int),
		minFirst:  minFirst,
	}
}

func (pq PriorityMemoryQueue) Len() int { return len(pq.items) }

func (pq PriorityMemoryQueue) Less(i, j int) bool {
	if pq.items[i].Priority == pq.items[j].Priority {
		if pq.minFirst {
			return pq.items[i].ID < pq.items[j].ID
		} else {
			return pq.items[i].ID > pq.items[j].ID
		}
	}
	if pq.minFirst {
		return pq.items[i].Priority < pq.items[j].Priority
	} else {
		return pq.items[i].Priority > pq.items[j].Priority
	}
}

func (pq PriorityMemoryQueue) Swap(i, j int) {
	if len(pq.items) > 0 {
		pq.items[i], pq.items[j] = pq.items[j], pq.items[i]
		pq.idToIndex[pq.items[i].ID] = i
		pq.idToIndex[pq.items[j].ID] = j
	}
}

func (pq *PriorityMemoryQueue) Push(x any) {
	n := len(pq.items)
	item := x.(*Item)
	pq.idToIndex[item.ID] = n
	pq.items = append(pq.items, item)
}

func (pq *PriorityMemoryQueue) Peek() any {
	return pq.items[0]
}

func (pq *PriorityMemoryQueue) Pop() any {
	if len(pq.items) == 0 {
		return nil
	}

	old := pq.items
	n := len(old)
	item := old[n-1]
	old[n-1] = nil // avoid memory leak
	pq.items = old[0 : n-1]
	delete(pq.idToIndex, item.ID)

	// Check for empty and reset items capacity and idToIndex to avoid memory leaks
	if len(pq.items) == 0 {
		pq.items = []*Item{}                // Reset to a new zero-length/zero-capacity slice
		pq.idToIndex = make(map[uint64]int) // Reset to a new zero-length/zero-capacity map
	}

	return item
}

func (pq *PriorityMemoryQueue) Get(id uint64) *Item {
	index, ok := pq.idToIndex[id]
	if !ok {
		return nil
	}
	return pq.items[index]
}

func (pq *PriorityMemoryQueue) Delete(id uint64) *Item {
	index, ok := pq.idToIndex[id]
	if !ok {
		return nil
	}

	item := heap.Remove(pq, index).(*Item)
	delete(pq.idToIndex, id)

	return item
}

func (pq *PriorityMemoryQueue) UpdatePriority(id uint64, priority int64) {
	index, ok := pq.idToIndex[id]
	if !ok {
		fmt.Println("Message not found")
		return
	}
	pq.items[index].Priority = priority
	heap.Fix(pq, index)
}
