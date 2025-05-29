package memory

import (
	"container/heap"
	"sync"

	avl "github.com/kgantsov/doq/pkg/weighted_avl"
	"github.com/rs/zerolog/log"
)

type FairAckMemoryQueue struct {
	maxUnacked int

	weights        *avl.WeightedAVL
	unackedByGroup map[string]int
	queues         map[string]*PriorityMemoryQueue
	mu             sync.Mutex
}

// CalculateWeight calculates the weight for a group based on the number of unacked items.
func CalculateWeight(maxUnacked int, unacked int) int {
	if unacked > maxUnacked {
		return 0
	}
	n := (1 - float64(unacked)/float64(maxUnacked)) * 10
	return int(n)
}

func NewFairAckMemoryQueue(maxUnacked int) *FairAckMemoryQueue {
	return &FairAckMemoryQueue{
		weights:        avl.NewWeightedAVL(),
		queues:         make(map[string]*PriorityMemoryQueue),
		unackedByGroup: make(map[string]int),
		maxUnacked:     maxUnacked,
	}
}

func (q *FairAckMemoryQueue) Enqueue(group string, item *Item) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if _, ok := q.queues[group]; !ok {
		q.queues[group] = NewPriorityMemoryQueue(true)
		q.weights.Update(group, CalculateWeight(q.maxUnacked, 0))
	}

	heap.Push(q.queues[group], item)
}

func (q *FairAckMemoryQueue) Dequeue(ack bool) *Item {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.queues) == 0 {
		return nil
	}

	var selectedGroup string

	selectedGroup = q.weights.Sample()
	if selectedGroup == "" {
		log.Info().Msgf("No group selected")
		return nil
	}

	log.Debug().Msgf(
		"Dequeue %+v ::: %+v ====> %s %d",
		q.unackedByGroup,
		q.weights.Items(),
		selectedGroup,
		q.weights.Sum(),
	)

	if _, ok := q.queues[selectedGroup]; !ok {
		log.Info().Msgf("No queue for group %s", selectedGroup)
		if _, ok := q.unackedByGroup[selectedGroup]; ok {
			delete(q.unackedByGroup, selectedGroup)
			q.weights.Remove(selectedGroup)

			selectedGroup = q.weights.Sample()
			if selectedGroup == "" {
				log.Info().Msgf("No group selected")
				return nil
			}
		}
		return nil
	}

	item := heap.Pop(q.queues[selectedGroup]).(*Item)
	if item == nil {
		log.Info().Msgf("No item in queue for group %s", selectedGroup)
		return nil
	}

	if q.queues[selectedGroup].Len() == 0 {
		delete(q.queues, selectedGroup)
		q.weights.Remove(selectedGroup)
	}

	if !ack {
		q.unackedByGroup[selectedGroup]++
		q.weights.Update(
			selectedGroup, CalculateWeight(q.maxUnacked, q.unackedByGroup[selectedGroup]),
		)
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

	if q.queues[group].Len() == 0 {
		delete(q.queues, group)
		delete(q.unackedByGroup, group)
		q.weights.Remove(group)
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

func (q *FairAckMemoryQueue) UpdateWeights(group string, id uint64) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if _, ok := q.unackedByGroup[group]; !ok {
		return nil
	}

	q.unackedByGroup[group]--
	q.weights.Update(group, CalculateWeight(q.maxUnacked, q.unackedByGroup[group]))

	return nil
}
