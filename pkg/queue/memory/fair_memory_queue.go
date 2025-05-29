package memory

import (
	"container/heap"
	"sync"
)

type LinkedListNode struct {
	group string
	queue *PriorityMemoryQueue
	prev  *LinkedListNode
	next  *LinkedListNode
}

func NewLinkedListNode(group string, queue *PriorityMemoryQueue) *LinkedListNode {
	node := new(LinkedListNode)
	node.group = group
	node.queue = queue

	return node
}

func (n *LinkedListNode) Queue() *PriorityMemoryQueue {
	return n.queue
}

type LinkedList struct {
	head  *LinkedListNode
	tail  *LinkedListNode
	total uint64
}

func (l *LinkedList) Append(node *LinkedListNode) {
	if l.head == nil {
		l.head = node
		l.tail = l.head
	} else {
		l.tail.next = node
		node.prev = l.tail
		l.tail = l.tail.next
	}

	l.total++
}

func (l *LinkedList) Remove(node *LinkedListNode) {
	if node.prev != nil {
		node.prev.next = node.next
	} else {
		l.head = node.next
	}

	if node.next != nil {
		node.next.prev = node.prev
	} else {
		l.tail = node.prev
	}

	l.total--
}

func (l *LinkedList) Len() uint64 {
	return l.total
}

// FairMemoryQueue is a fair queue that balances between different groups
type FairMemoryQueue struct {
	queues      map[string]*LinkedListNode
	roundRobin  *LinkedList
	currentNone *LinkedListNode
	mu          sync.RWMutex

	totalMessages uint64
}

// FairQueue creates a new FairQueue
func NewFairMemoryQueue() *FairMemoryQueue {
	return &FairMemoryQueue{
		queues:      make(map[string]*LinkedListNode),
		roundRobin:  &LinkedList{},
		currentNone: nil,
	}
}

// Enqueue adds a message to the queue
func (fq *FairMemoryQueue) Enqueue(group string, item *Item) {
	fq.mu.Lock()
	defer fq.mu.Unlock()

	// Add the message to the respective goups's queue
	if _, exists := fq.queues[group]; !exists {
		// Add group to round robin if not exists
		node := NewLinkedListNode(group, NewPriorityMemoryQueue(true))
		fq.queues[group] = node
		fq.roundRobin.Append(node)
	}
	heap.Push(fq.queues[group].Queue(), item)
	fq.totalMessages++
}

// Dequeue removes and returns the next message in a fair way
func (fq *FairMemoryQueue) Dequeue(ack bool) *Item {
	fq.mu.Lock()
	defer fq.mu.Unlock()

	// No groups in the queue
	if fq.roundRobin.Len() == 0 {
		return nil
	}

	if fq.currentNone == nil {
		fq.currentNone = fq.roundRobin.head
	}

	if fq.currentNone.Queue().Len() == 0 {
		return nil
	}

	// Get the message from the group's queue
	item := heap.Pop(fq.currentNone.Queue()).(*Item)

	// If the group's queue is empty, remove the group from the round-robin list
	if fq.currentNone.Queue().Len() == 0 {
		next := fq.currentNone.next
		fq.roundRobin.Remove(fq.currentNone)
		delete(fq.queues, fq.currentNone.group)
		fq.currentNone = next
	} else {
		// Move to the next group for the next Dequeue operation
		fq.currentNone = fq.currentNone.next
	}

	fq.totalMessages--
	return item
}

func (fq *FairMemoryQueue) Get(group string, id uint64) *Item {
	fq.mu.RLock()
	defer fq.mu.RUnlock()

	if _, exists := fq.queues[group]; !exists {
		return nil
	}

	return fq.queues[group].Queue().Get(id)
}

func (fq *FairMemoryQueue) Delete(group string, id uint64) *Item {
	fq.mu.Lock()
	defer fq.mu.Unlock()

	if _, exists := fq.queues[group]; !exists {
		return nil
	}

	item := fq.queues[group].Queue().Delete(id)
	fq.totalMessages--

	return item
}

func (fq *FairMemoryQueue) UpdatePriority(group string, id uint64, priority int64) {
	fq.mu.Lock()
	defer fq.mu.Unlock()

	if _, exists := fq.queues[group]; !exists {
		return
	}

	fq.queues[group].Queue().UpdatePriority(id, priority)
}

func (fq *FairMemoryQueue) Len() uint64 {
	fq.mu.RLock()
	defer fq.mu.RUnlock()

	return fq.totalMessages
}

func (fq *FairMemoryQueue) UpdateWeights(group string, id uint64) error {
	return nil
}
