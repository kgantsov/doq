package memory

import (
	"container/heap"
	"math"
	"sync"
	"time"

	avl "github.com/kgantsov/doq/pkg/weighted_avl"
	"github.com/rs/zerolog/log"
)

type FairWeightedQueue struct {
	maxUnacked int

	weights        *avl.WeightedAVL
	unackedByGroup map[string]int
	queues         map[string]*PriorityMemoryQueue
	mu             sync.RWMutex
}

// CalculateWeight calculates the weight for a group based on unacked items and queue size.
func CalculateWeight(maxUnacked int, unacked int, queueSize int) int {
	if queueSize == 0 {
		return 0
	}

	if maxUnacked == 0 {
		// default to max int
		maxUnacked = math.MaxInt64
	}

	if unacked >= maxUnacked {
		return 0
	}
	unackedFactor := 1 - float64(unacked)/float64(maxUnacked+1)
	queueSizeFactor := float64(1)
	// queueSizeFactor := float64(queueSize) / float64(queueSize+unacked) // Optional: normalizes queue size impact
	weight := unackedFactor * queueSizeFactor * 10
	return int(weight)
}

func NewFairWeightedQueue(maxUnacked int) *FairWeightedQueue {
	return &FairWeightedQueue{
		weights:        avl.NewWeightedAVL(),
		queues:         make(map[string]*PriorityMemoryQueue),
		unackedByGroup: make(map[string]int),
		maxUnacked:     maxUnacked,
	}
}

func (q *FairWeightedQueue) UpdateMaxUnacked(maxUnacked int) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if maxUnacked == 0 {
		// default to max int
		maxUnacked = math.MaxInt64
	}
	q.maxUnacked = maxUnacked
	return nil
}

func (q *FairWeightedQueue) Enqueue(group string, item *Item) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if _, ok := q.queues[group]; !ok {
		q.queues[group] = NewPriorityMemoryQueue(true)
	}

	heap.Push(q.queues[group], item)

	q.weights.Update(
		group,
		CalculateWeight(q.maxUnacked, q.unackedByGroup[group], q.queues[group].Len()),
	)
}

func (q *FairWeightedQueue) Dequeue(ack bool) *Item {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.queues) == 0 {
		return nil
	}

	var selectedGroup string

	selectedGroup = q.weights.Sample()
	if selectedGroup == "" {
		log.Debug().Msgf("No group selected")
		return nil
	}

	item := heap.Pop(q.queues[selectedGroup]).(*Item)
	if item == nil {
		log.Debug().Msgf("No item in queue for group %s", selectedGroup)
		return nil
	}

	if !ack {
		q.unackedByGroup[selectedGroup]++
	}
	if q.unackedByGroup[selectedGroup] == 0 && q.queues[selectedGroup].Len() == 0 {
		delete(q.queues, selectedGroup)
		delete(q.unackedByGroup, selectedGroup)
		q.weights.Remove(selectedGroup)
	} else {
		q.weights.Update(
			selectedGroup,
			CalculateWeight(
				q.maxUnacked, q.unackedByGroup[selectedGroup], q.queues[selectedGroup].Len(),
			),
		)
	}

	return item
}

// PeekReady reports whether Dequeue would currently return a message: some group
// must hold messages and sit under the unacked limit (i.e. carry a non-zero
// weight). Readiness is event-based (an ack frees an unacked slot), so
// nextReadyIn is always 0.
func (q *FairWeightedQueue) PeekReady() (bool, time.Duration) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	// Each group's stored weight is CalculateWeight(...), which is always >= 0
	// and is > 0 exactly when that group has a deliverable message under the
	// unacked limit. The AVL keeps a running root sum, so a positive total means
	// at least one group is ready — an O(1) check instead of scanning every group.
	return q.weights.Sum() > 0, 0
}

func (q *FairWeightedQueue) Get(group string, id uint64) *Item {
	q.mu.Lock()
	defer q.mu.Unlock()

	if _, ok := q.queues[group]; !ok {
		return nil
	}

	return q.queues[group].Get(id)
}

func (q *FairWeightedQueue) Delete(group string, id uint64) *Item {
	q.mu.Lock()
	defer q.mu.Unlock()

	if _, ok := q.queues[group]; !ok {
		return nil
	}

	item := q.queues[group].Delete(id)
	if item == nil {
		return nil
	}

	if q.queues[group].Len() == 0 {
		delete(q.queues, group)
		delete(q.unackedByGroup, group)
		q.weights.Remove(group)
	}

	return item
}

func (q *FairWeightedQueue) UpdatePriority(group string, id uint64, priority int64) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if _, ok := q.queues[group]; !ok {
		return
	}

	q.queues[group].UpdatePriority(id, priority)
}

func (q *FairWeightedQueue) Len() uint64 {
	q.mu.Lock()
	defer q.mu.Unlock()

	var total uint64
	for _, queue := range q.queues {
		total += uint64(queue.Len())
	}
	return total
}

func (q *FairWeightedQueue) UpdateWeights(group string, id uint64) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if _, ok := q.unackedByGroup[group]; !ok {
		return nil
	}

	q.unackedByGroup[group]--

	q.weights.Update(
		group,
		CalculateWeight(
			q.maxUnacked, q.unackedByGroup[group], q.queues[group].Len(),
		),
	)

	if q.queues[group].Len() == 0 && q.unackedByGroup[group] <= 0 {
		// if q.unackedByGroup[group] <= 0 {
		delete(q.unackedByGroup, group)
		delete(q.queues, group)
		q.weights.Remove(group)
	}

	return nil
}
