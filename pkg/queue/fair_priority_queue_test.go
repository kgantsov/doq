package queue

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFairPriorityQueueSingleCustomer(t *testing.T) {
	fq := NewFairPriorityQueue()

	fq.Enqueue("customer1", &Item{ID: 1, Priority: 10})
	fq.Enqueue("customer1", &Item{ID: 2, Priority: 10})
	fq.Enqueue("customer1", &Item{ID: 3, Priority: 10})
	fq.Enqueue("customer1", &Item{ID: 4, Priority: 10})
	fq.Enqueue("customer1", &Item{ID: 5, Priority: 10})
	fq.Enqueue("customer1", &Item{ID: 6, Priority: 10})
	fq.Enqueue("customer1", &Item{ID: 7, Priority: 10})
	fq.Enqueue("customer1", &Item{ID: 8, Priority: 10})
	fq.Enqueue("customer1", &Item{ID: 9, Priority: 10})
	fq.Enqueue("customer1", &Item{ID: 10, Priority: 10})

	assert.Equal(t, uint64(1), fq.Dequeue().ID)
	assert.Equal(t, uint64(2), fq.Dequeue().ID)
	assert.Equal(t, uint64(3), fq.Dequeue().ID)
	assert.Equal(t, uint64(4), fq.Dequeue().ID)
	assert.Equal(t, uint64(5), fq.Dequeue().ID)
	assert.Equal(t, uint64(6), fq.Dequeue().ID)
	assert.Equal(t, uint64(7), fq.Dequeue().ID)
	assert.Equal(t, uint64(8), fq.Dequeue().ID)
	assert.Equal(t, uint64(9), fq.Dequeue().ID)
	assert.Equal(t, uint64(10), fq.Dequeue().ID)
}

func TestFairPriorityQueue(t *testing.T) {
	fq := NewFairPriorityQueue()

	fq.Enqueue("customer1", &Item{ID: 1, Priority: 10})
	fq.Enqueue("customer1", &Item{ID: 2, Priority: 10})
	fq.Enqueue("customer1", &Item{ID: 3, Priority: 10})
	fq.Enqueue("customer1", &Item{ID: 4, Priority: 10})
	fq.Enqueue("customer2", &Item{ID: 5, Priority: 10})
	fq.Enqueue("customer2", &Item{ID: 6, Priority: 10})
	fq.Enqueue("customer3", &Item{ID: 7, Priority: 10})
	fq.Enqueue("customer3", &Item{ID: 8, Priority: 10})
	fq.Enqueue("customer4", &Item{ID: 9, Priority: 10})
	fq.Enqueue("customer5", &Item{ID: 10, Priority: 10})

	assert.Equal(t, uint64(1), fq.Dequeue().ID)
	assert.Equal(t, uint64(5), fq.Dequeue().ID)
	assert.Equal(t, uint64(7), fq.Dequeue().ID)
	assert.Equal(t, uint64(9), fq.Dequeue().ID)
	assert.Equal(t, uint64(10), fq.Dequeue().ID)
	assert.Equal(t, uint64(2), fq.Dequeue().ID)
	assert.Equal(t, uint64(6), fq.Dequeue().ID)
	assert.Equal(t, uint64(8), fq.Dequeue().ID)
	assert.Equal(t, uint64(3), fq.Dequeue().ID)
	assert.Equal(t, uint64(4), fq.Dequeue().ID)
}

func TestFairPriorityQueue1(t *testing.T) {
	fq := NewFairPriorityQueue()

	fq.Enqueue("customer1", &Item{ID: 1, Priority: 10})
	fq.Enqueue("customer1", &Item{ID: 2, Priority: 10})
	fq.Enqueue("customer2", &Item{ID: 3, Priority: 10})

	assert.Equal(t, uint64(1), fq.Dequeue().ID)
	assert.Equal(t, uint64(3), fq.Dequeue().ID)
	assert.Equal(t, uint64(2), fq.Dequeue().ID)
	assert.Nil(t, fq.Dequeue())
}
