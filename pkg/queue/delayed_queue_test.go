package queue

import (
	"testing"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
)

// TestDelayedMemoryQueue tests the priority queue
func TestDelayedMemoryQueue(t *testing.T) {
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
			pq := NewDelayedMemoryQueue(true)

			for i, m := range tt.messages {
				pq.Enqueue("test", &m)
				assert.Equal(t, pq.Len(), uint64(i+1))
			}
			assert.Equal(t, pq.Len(), uint64(len(tt.messages)))

			for i := 0; i < len(tt.messages); i++ {

				m := pq.Dequeue()

				assert.Equal(t, pq.Len(), uint64(len(tt.messages)-(i+1)))

				log.Info().Msgf("ID: %d expected %d", m.ID, tt.expected[i])
				assert.Equal(t, tt.expected[i], m.ID)
			}
		})
	}
}

func TestDelayedMemoryQueueDelayed(t *testing.T) {

	pq := NewDelayedMemoryQueue(true)

	priority := time.Now().UTC().Add(1 * time.Second).Unix()
	pq.Enqueue("test", &Item{ID: 1, Priority: priority, Group: "default"})

	m1 := pq.Get("test", 1)
	assert.Equal(t, uint64(1), m1.ID)
	assert.Equal(t, priority, m1.Priority)

	m1 = pq.Dequeue()
	assert.Nil(t, m1)

	time.Sleep(1 * time.Second)

	m1 = pq.Dequeue()
	assert.Equal(t, uint64(1), m1.ID)
	assert.Equal(t, priority, m1.Priority)
}

func BenchmarkDelayedMemoryQueueEnqueue(b *testing.B) {
	pq := NewDelayedMemoryQueue(true)

	// Pre-fill the queue with items to ensure there’s something to dequeue
	for i := 0; i < b.N; i++ {
		pq.Enqueue("default", &Item{ID: uint64(i), Priority: 10, Group: "default"})
	}
}

func BenchmarkDelayedMemoryQueueDequeue(b *testing.B) {
	pq := NewDelayedMemoryQueue(true)

	// Pre-fill the queue with items to ensure there’s something to dequeue
	for i := 0; i < b.N; i++ {
		pq.Enqueue("default", &Item{ID: uint64(i), Priority: 10, Group: "default"})
	}

	b.ResetTimer() // Reset timer to focus only on Dequeue operation timing

	for i := 0; i < b.N; i++ {
		pq.Dequeue()
	}
}
