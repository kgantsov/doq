package memory

import (
	"container/heap"
	"math"
	"strings"
	"sync"
)

// GroupNode represents a node in the hierarchical group tree
type GroupNode struct {
	name         string
	children     map[string]*GroupNode
	childrenList *LinkedList
	currentChild *LinkedListNode
	queue        *PriorityMemoryQueue
	parent       *GroupNode
	messageCount uint64
}

func NewGroupNode(name string, parent *GroupNode) *GroupNode {
	return &GroupNode{
		name:         name,
		children:     make(map[string]*GroupNode),
		childrenList: &LinkedList{},
		queue:        NewPriorityMemoryQueue(true),
		parent:       parent,
		messageCount: 0,
	}
}

// FairRoundRobinQueue is a fair queue that balances between different groups
type FairRoundRobinQueue struct {
	root           *GroupNode
	mu             sync.RWMutex
	maxUnacked     int
	unackedByGroup map[string]int
	totalMessages  uint64
	groupKeyToLeaf map[string]*GroupNode
}

// NewFairRoundRobinQueue creates a new FairQueue
func NewFairRoundRobinQueue(maxUnacked int) *FairRoundRobinQueue {
	if maxUnacked == 0 {
		maxUnacked = math.MaxInt64
	}
	return &FairRoundRobinQueue{
		root:           NewGroupNode("root", nil),
		unackedByGroup: make(map[string]int),
		maxUnacked:     maxUnacked,
		groupKeyToLeaf: make(map[string]*GroupNode),
	}
}

func (fq *FairRoundRobinQueue) UpdateMaxUnacked(maxUnacked int) error {
	fq.mu.Lock()
	defer fq.mu.Unlock()

	if maxUnacked == 0 {
		maxUnacked = math.MaxInt64
	}
	fq.maxUnacked = maxUnacked
	return nil
}

func makeGroupKey(groups []string) string {
	return strings.Join(groups, ".")
}

// Enqueue adds a message to the queue
func (fq *FairRoundRobinQueue) Enqueue(group string, item *Item) {
	fq.mu.Lock()
	defer fq.mu.Unlock()

	if group == "" {
		group = "default"
	}

	groups := strings.Split(group, ".")

	item.Group = group

	// Navigate/create the tree structure
	current := fq.root
	for _, groupName := range groups {
		if _, exists := current.children[groupName]; !exists {
			// Create new child node
			child := NewGroupNode(groupName, current)
			current.children[groupName] = child

			// Add to linked list for round-robin
			node := NewLinkedListNode(groupName, nil)
			node.queue = nil // This is not a leaf node initially
			current.childrenList.Append(node)
		}
		current = current.children[groupName]
	}

	// Current is now the leaf node - add message to its queue
	heap.Push(current.queue, item)

	// Update message counts up the tree
	node := current
	for node != nil {
		node.messageCount++
		node = node.parent
	}

	fq.groupKeyToLeaf[group] = current
	fq.totalMessages++
}

// dequeueFromNode recursively dequeues from a node using round-robin
func (fq *FairRoundRobinQueue) dequeueFromNode(node *GroupNode) *Item {
	// Base case: leaf node with messages
	if len(node.children) == 0 {
		if node.queue.Len() > 0 {
			return heap.Pop(node.queue).(*Item)
		}
		return nil
	}

	// Recursive case: internal node - do round-robin on children
	if node.childrenList.Len() == 0 {
		return nil
	}

	// Initialize current child if needed
	if node.currentChild == nil {
		node.currentChild = node.childrenList.head
	}

	startNode := node.currentChild
	visited := 0
	maxVisits := int(node.childrenList.Len())

	for visited < maxVisits {
		childName := node.currentChild.group
		childNode := node.children[childName]

		// Check unacked limit for this group
		groupKey := fq.getGroupKeyForNode(childNode)
		if fq.unackedByGroup[groupKey] < fq.maxUnacked {
			// Try to dequeue from this child
			item := fq.dequeueFromNode(childNode)
			if item != nil {
				// Move to next child for next dequeue
				node.currentChild = node.currentChild.next
				if node.currentChild == nil {
					node.currentChild = node.childrenList.head
				}

				return item
			}
		}

		// Move to next child
		node.currentChild = node.currentChild.next
		if node.currentChild == nil {
			node.currentChild = node.childrenList.head
		}

		visited++
		if node.currentChild == startNode {
			break
		}
	}

	return nil
}

func (fq *FairRoundRobinQueue) getGroupKeyForNode(node *GroupNode) string {
	path := []string{}
	current := node
	for current != nil && current.parent != nil {
		path = append([]string{current.name}, path...)
		current = current.parent
	}
	return makeGroupKey(path)
}

func (fq *FairRoundRobinQueue) removeEmptyChild(parent *GroupNode, childName string) {
	child := parent.children[childName]

	// Remove from linked list
	current := parent.childrenList.head
	for current != nil {
		if current.group == childName {
			parent.childrenList.Remove(current)
			break
		}
		current = current.next
	}

	// Remove from maps
	delete(parent.children, childName)

	// remove leaf reference
	groupKey := fq.getGroupKeyForNode(child)
	delete(fq.groupKeyToLeaf, groupKey)

	// Reset RR pointer
	if parent.currentChild != nil && parent.currentChild.group == childName {
		parent.currentChild = parent.childrenList.head
	}

	// recursively clean empty parents
	if parent != fq.root && parent.messageCount == 0 {
		fq.removeEmptyChild(parent.parent, parent.name)
	}
}

// Dequeue removes and returns the next message in a fair way
func (fq *FairRoundRobinQueue) Dequeue(ack bool) *Item {
	fq.mu.Lock()
	defer fq.mu.Unlock()

	item := fq.dequeueFromNode(fq.root)
	if item == nil {
		return nil
	}

	// Update message counts up the tree
	if leaf, exists := fq.groupKeyToLeaf[item.Group]; exists {
		node := leaf
		for node != nil {
			node.messageCount--
			node = node.parent
		}

		if leaf.messageCount == 0 {
			fq.removeEmptyChild(leaf.parent, leaf.name)
		}
	}

	if !ack {
		fq.unackedByGroup[item.Group]++
	}

	fq.totalMessages--
	return item
}

func (fq *FairRoundRobinQueue) Get(group string, id uint64) *Item {
	fq.mu.RLock()
	defer fq.mu.RUnlock()

	if leaf, exists := fq.groupKeyToLeaf[group]; exists {
		return leaf.queue.Get(id)
	}
	return nil
}

func (fq *FairRoundRobinQueue) Delete(group string, id uint64) *Item {
	fq.mu.Lock()
	defer fq.mu.Unlock()

	if leaf, exists := fq.groupKeyToLeaf[group]; exists {
		item := leaf.queue.Delete(id)
		if item != nil {
			// Update message counts up the tree
			node := leaf
			for node != nil {
				node.messageCount--
				node = node.parent
			}

			if leaf.messageCount == 0 {
				fq.removeEmptyChild(leaf.parent, leaf.name)
			}

			fq.totalMessages--
		}
		return item
	}
	return nil
}

func (fq *FairRoundRobinQueue) UpdatePriority(group string, id uint64, priority int64) {
	fq.mu.Lock()
	defer fq.mu.Unlock()

	if leaf, exists := fq.groupKeyToLeaf[group]; exists {
		leaf.queue.UpdatePriority(id, priority)
	}
}

func (fq *FairRoundRobinQueue) Len() uint64 {
	fq.mu.RLock()
	defer fq.mu.RUnlock()
	return fq.totalMessages
}

func (fq *FairRoundRobinQueue) UpdateWeights(group string, id uint64) error {
	fq.mu.Lock()
	defer fq.mu.Unlock()

	count, ok := fq.unackedByGroup[group]
	if !ok {
		return nil
	}

	count--
	if count <= 0 {
		delete(fq.unackedByGroup, group)
	} else {
		fq.unackedByGroup[group] = count
	}
	return nil
}
