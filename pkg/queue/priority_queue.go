package queue

import (
	"container/heap"
	"fmt"
)

type Item struct {
	ID       uint64
	Priority int
}

func (m *Item) UpdatePriority(newPriority int) {
	m.Priority = newPriority
}

type PriorityQueue struct {
	items     []*Item
	idToIndex map[uint64]int
}

func NewPriorityQueue() *PriorityQueue {
	return &PriorityQueue{
		items:     []*Item{},
		idToIndex: make(map[uint64]int),
	}
}

func (pq PriorityQueue) Len() int { return len(pq.items) }

func (pq PriorityQueue) Less(i, j int) bool {
	if pq.items[i].Priority == pq.items[j].Priority {
		return pq.items[i].ID < pq.items[j].ID
	}
	return pq.items[i].Priority < pq.items[j].Priority
}

func (pq PriorityQueue) Swap(i, j int) {
	pq.items[i], pq.items[j] = pq.items[j], pq.items[i]
	pq.idToIndex[pq.items[i].ID] = i
	pq.idToIndex[pq.items[j].ID] = j
}

func (pq *PriorityQueue) Push(x any) {
	n := len(pq.items)
	item := x.(*Item)
	pq.idToIndex[item.ID] = n
	pq.items = append(pq.items, item)
}

func (pq *PriorityQueue) Pop() any {
	old := pq.items
	n := len(old)
	item := old[n-1]
	old[n-1] = nil // avoid memory leak
	pq.items = old[0 : n-1]
	delete(pq.idToIndex, item.ID)
	return item
}

func (pq *PriorityQueue) GetByID(id uint64) *Item {
	index, ok := pq.idToIndex[id]
	if !ok {
		return nil
	}
	return pq.items[index]
}

func (pq *PriorityQueue) UpdatePriority(id uint64, priority int) {
	index, ok := pq.idToIndex[id]
	if !ok {
		fmt.Println("Message not found")
		return
	}
	pq.items[index].Priority = priority
	heap.Fix(pq, index)
}
