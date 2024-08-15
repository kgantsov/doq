package queue

import (
	"container/heap"
	"testing"

	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
)

// TestPriorityQueue tests the priority queue
func TestPriorityQueue(t *testing.T) {
	tests := []struct {
		name     string
		messages []Item
		expected []uint64
	}{
		{
			name:     "No messages test",
			messages: []Item{},
			expected: []uint64{},
		},
		{
			name: "First test",
			messages: []Item{
				{ID: 1, Priority: 10},
				{ID: 2, Priority: 50},
				{ID: 3, Priority: 20},
				{ID: 4, Priority: 70},
				{ID: 5, Priority: 100},
				{ID: 6, Priority: 7},
				{ID: 7, Priority: 4},
				{ID: 8, Priority: 2},
				{ID: 9, Priority: 5},
			},
			expected: []uint64{8, 7, 9, 6, 1, 3, 2, 4, 5},
		},
		{
			name: "Second test",
			messages: []Item{
				{ID: 1, Priority: 40},
				{ID: 2, Priority: 1},
				{ID: 3, Priority: 30},
				{ID: 4, Priority: 53},
				{ID: 5, Priority: 10},
				{ID: 6, Priority: 41},
				{ID: 7, Priority: 32},
				{ID: 8, Priority: 22},
				{ID: 9, Priority: 7},
				{ID: 10, Priority: 6},
				{ID: 11, Priority: 53},
				{ID: 12, Priority: 53},
			},
			expected: []uint64{2, 10, 9, 5, 8, 3, 7, 1, 6, 4, 11, 12},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pq := NewPriorityQueue(true)

			for i, m := range tt.messages {
				heap.Push(pq, &m)
				assert.Equal(t, pq.Len(), i+1)
			}
			assert.Equal(t, pq.Len(), len(tt.messages))

			for i := 0; i < len(tt.messages); i++ {
				m := pq.Peek().(*Item)
				assert.Equal(t, tt.expected[i], m.ID)

				m = heap.Pop(pq).(*Item)

				assert.Equal(t, pq.Len(), len(tt.messages)-(i+1))

				log.Info().Msgf("ID: %d expected %d", m.ID, tt.expected[i])
				assert.Equal(t, tt.expected[i], m.ID)
			}
		})
	}
}

// TestPriorityQueue tests the priority queue
func TestPriorityQueueMaxFirst(t *testing.T) {
	tests := []struct {
		name     string
		messages []Item
		expected []uint64
	}{
		{
			name:     "No messages test",
			messages: []Item{},
			expected: []uint64{},
		},
		{
			name: "First test",
			messages: []Item{
				{ID: 1, Priority: 10},
				{ID: 2, Priority: 50},
				{ID: 3, Priority: 20},
				{ID: 4, Priority: 70},
				{ID: 5, Priority: 100},
				{ID: 6, Priority: 7},
				{ID: 7, Priority: 4},
				{ID: 8, Priority: 2},
				{ID: 9, Priority: 5},
			},
			expected: []uint64{5, 4, 2, 3, 1, 6, 9, 7, 8},
		},
		{
			name: "Second test",
			messages: []Item{
				{ID: 1, Priority: 40},
				{ID: 2, Priority: 1},
				{ID: 3, Priority: 30},
				{ID: 4, Priority: 53},
				{ID: 5, Priority: 10},
				{ID: 6, Priority: 41},
				{ID: 7, Priority: 32},
				{ID: 8, Priority: 22},
				{ID: 9, Priority: 7},
				{ID: 10, Priority: 6},
				{ID: 11, Priority: 53},
				{ID: 12, Priority: 53},
			},
			expected: []uint64{12, 11, 4, 6, 1, 7, 3, 8, 5, 9, 10, 2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pq := NewPriorityQueue(false)

			for i, m := range tt.messages {
				heap.Push(pq, &m)
				assert.Equal(t, pq.Len(), i+1)
			}
			assert.Equal(t, pq.Len(), len(tt.messages))

			for i := 0; i < len(tt.messages); i++ {
				m := pq.Peek().(*Item)
				assert.Equal(t, tt.expected[i], m.ID)

				m = heap.Pop(pq).(*Item)

				assert.Equal(t, pq.Len(), len(tt.messages)-(i+1))

				log.Info().Msgf("ID: %d expected %d", m.ID, tt.expected[i])
				assert.Equal(t, tt.expected[i], m.ID)
			}
		})
	}
}

func TestPriorityQueueUpdatePriority(t *testing.T) {
	pq := NewPriorityQueue(true)

	heap.Push(pq, &Item{ID: 1, Priority: 1})
	heap.Push(pq, &Item{ID: 2, Priority: 2})
	heap.Push(pq, &Item{ID: 3, Priority: 3})
	heap.Push(pq, &Item{ID: 4, Priority: 4})

	pq.UpdatePriority(4, 0)
	pq.UpdatePriority(3, 1)

	m := heap.Pop(pq).(*Item)
	assert.Equal(t, uint64(4), m.ID)

	m = heap.Pop(pq).(*Item)
	assert.Equal(t, uint64(1), m.ID)

	m = heap.Pop(pq).(*Item)
	assert.Equal(t, uint64(3), m.ID)

	m = heap.Pop(pq).(*Item)
	assert.Equal(t, uint64(2), m.ID)
}

func TestPriorityQueuePeek(t *testing.T) {
	pq := NewPriorityQueue(true)

	heap.Push(pq, &Item{ID: 4, Priority: 4})
	m := pq.Peek().(*Item)
	assert.Equal(t, uint64(4), m.ID)

	heap.Push(pq, &Item{ID: 5, Priority: 5})
	m = pq.Peek().(*Item)
	assert.Equal(t, uint64(4), m.ID)

	heap.Push(pq, &Item{ID: 3, Priority: 3})
	m = pq.Peek().(*Item)
	assert.Equal(t, uint64(3), m.ID)

	heap.Push(pq, &Item{ID: 6, Priority: 6})
	m = pq.Peek().(*Item)
	assert.Equal(t, uint64(3), m.ID)

	heap.Push(pq, &Item{ID: 2, Priority: 2})
	m = pq.Peek().(*Item)
	assert.Equal(t, uint64(2), m.ID)

	heap.Push(pq, &Item{ID: 1, Priority: 1})
	m = pq.Peek().(*Item)
	assert.Equal(t, uint64(1), m.ID)
}
