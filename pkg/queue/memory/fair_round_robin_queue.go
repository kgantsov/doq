package memory

import (
	"container/heap"
	"math"
	"strings"
	"sync"
	"time"
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
	// groupKey is the dot-joined path from the root's child down to this node,
	// computed once at insert time so readiness/dequeue checks don't rebuild
	// (and allocate) it on every visit. The root's groupKey is "".
	groupKey string
	// ready caches whether this leaf currently contributes to readyLeaves, so a
	// mutation can adjust the aggregate counter by a delta instead of recounting.
	// Only ever true for leaf nodes (see refreshLeafReady).
	ready bool
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
	// readyLeaves counts leaves that currently hold a deliverable message (queue
	// non-empty and group under the unacked limit). PeekReady reads it in O(1)
	// instead of walking the whole tree. It is maintained incrementally by
	// refreshLeafReady at every mutation point.
	readyLeaves int
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

	// The limit gates every leaf's readiness, so recompute the counter. Only
	// non-empty leaves can be ready, and those are exactly the entries in
	// groupKeyToLeaf (emptied leaves are removed from the tree).
	for _, leaf := range fq.groupKeyToLeaf {
		fq.refreshLeafReady(leaf)
	}
	return nil
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
			// Cache the full path key once; the root has an empty groupKey so
			// its direct children key on just their own name.
			if current.groupKey == "" {
				child.groupKey = groupName
			} else {
				child.groupKey = current.groupKey + "." + groupName
			}
			current.children[groupName] = child

			// Add to linked list for round-robin
			node := NewLinkedListNode(groupName, nil)
			node.queue = nil // This is not a leaf node initially
			current.childrenList.Append(node)

			// current just gained a child and is no longer a leaf, so it must
			// not count towards readyLeaves.
			fq.refreshLeafReady(current)
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

	// The destination leaf's queue just became non-empty.
	fq.refreshLeafReady(current)
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

// getGroupKeyForNode returns the node's precomputed dot-joined path key. The key
// is populated at insert time (see Enqueue), so this is an O(1) field read.
func (fq *FairRoundRobinQueue) getGroupKeyForNode(node *GroupNode) string {
	return node.groupKey
}

// refreshLeafReady recomputes whether node is a ready leaf and keeps readyLeaves
// in sync via the node's cached ready flag. A node counts as ready only when it
// is a leaf (no children), holds at least one message, and its group is under the
// unacked limit — the same leaf-level gating dequeueFromNode applies. Call it after
// any change to the node's queue length, its group's unacked count, or its
// children. Must be called under the write lock.
//
// The per-node unacked gating that dequeueFromNode also applies to *internal*
// nodes is deliberately not modelled here: an internal node only carries an
// unacked count in the degenerate case where the same group is used both as a
// full path and as a prefix of another (which strands messages regardless). In
// that case this counter may report ready when a walk would not — a harmless
// false positive (an extra no-op Dequeue), never a false negative, so no
// deliverable message is ever missed.
func (fq *FairRoundRobinQueue) refreshLeafReady(node *GroupNode) {
	ready := len(node.children) == 0 &&
		node.queue.Len() > 0 &&
		fq.unackedByGroup[node.groupKey] < fq.maxUnacked

	if ready == node.ready {
		return
	}
	node.ready = ready
	if ready {
		fq.readyLeaves++
	} else {
		fq.readyLeaves--
	}
}

func (fq *FairRoundRobinQueue) removeEmptyChild(parent *GroupNode, childName string) {
	child := parent.children[childName]

	// Drop the child's readiness contribution before it leaves the tree.
	if child.ready {
		child.ready = false
		fq.readyLeaves--
	}

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
	} else {
		// parent kept (root, or still holds messages); if it just lost its last
		// child it is a leaf again and may itself be ready.
		fq.refreshLeafReady(parent)
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

	// Account for the new unacked slot first so the readiness refresh below sees
	// the up-to-date count (a group that just hit its limit is no longer ready).
	if !ack {
		fq.unackedByGroup[item.Group]++
	}

	// Update message counts up the tree
	if leaf, exists := fq.groupKeyToLeaf[item.Group]; exists {
		node := leaf
		for node != nil {
			node.messageCount--
			node = node.parent
		}

		if leaf.messageCount == 0 {
			// Emptied leaf leaves the tree; removeEmptyChild drops its readiness.
			fq.removeEmptyChild(leaf.parent, leaf.name)
		} else {
			fq.refreshLeafReady(leaf)
		}
	}

	fq.totalMessages--
	return item
}

// hasReadyNode reports whether any leaf reachable from node holds a message
// whose group is under the unacked limit. It mirrors the gating in
// dequeueFromNode but visits every child (order-independent) and mutates no
// round-robin state. PeekReady no longer uses it (readyLeaves answers in O(1));
// it is retained as an independent oracle for tests.
func (fq *FairRoundRobinQueue) hasReadyNode(node *GroupNode) bool {
	if len(node.children) == 0 {
		return node.queue.Len() > 0
	}

	for _, childNode := range node.children {
		groupKey := fq.getGroupKeyForNode(childNode)
		if fq.unackedByGroup[groupKey] < fq.maxUnacked && fq.hasReadyNode(childNode) {
			return true
		}
	}
	return false
}

// PeekReady reports whether Dequeue would currently return a message. Readiness
// is event-based (an ack frees an unacked slot), so nextReadyIn is always 0.
// Backed by the incrementally maintained readyLeaves counter, so it is O(1)
// regardless of how many groups the tree holds.
func (fq *FairRoundRobinQueue) PeekReady() (bool, time.Duration) {
	fq.mu.RLock()
	defer fq.mu.RUnlock()

	return fq.readyLeaves > 0, 0
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
			} else {
				fq.refreshLeafReady(leaf)
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

	// Freeing an unacked slot can make a previously blocked leaf ready again.
	if leaf, exists := fq.groupKeyToLeaf[group]; exists {
		fq.refreshLeafReady(leaf)
	}
	return nil
}
