package memory

import (
	"testing"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
)

// TestDelayedQueue tests the priority queue
func TestDelayedQueue(t *testing.T) {
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
			pq := NewDelayedQueue(true)

			for i, m := range tt.messages {
				pq.Enqueue("test", &m)
				assert.Equal(t, pq.Len(), uint64(i+1))
			}
			assert.Equal(t, pq.Len(), uint64(len(tt.messages)))

			for i := 0; i < len(tt.messages); i++ {

				m := pq.Dequeue(false)

				assert.Equal(t, pq.Len(), uint64(len(tt.messages)-(i+1)))

				log.Info().Msgf("ID: %d expected %d", m.ID, tt.expected[i])
				assert.Equal(t, tt.expected[i], m.ID)
			}
		})
	}
}

func TestDelayedQueueDelayed(t *testing.T) {

	pq := NewDelayedQueue(true)

	priority := time.Now().UTC().Add(1 * time.Second).Unix()
	pq.Enqueue("test", &Item{ID: 1, Priority: priority, Group: "default"})

	m1 := pq.Get("test", 1)
	assert.Equal(t, uint64(1), m1.ID)
	assert.Equal(t, priority, m1.Priority)

	m1 = pq.Dequeue(false)
	assert.Nil(t, m1)

	time.Sleep(1 * time.Second)

	m1 = pq.Dequeue(false)
	assert.Equal(t, uint64(1), m1.ID)
	assert.Equal(t, priority, m1.Priority)
}

func BenchmarkDelayedQueueEnqueue(b *testing.B) {
	pq := NewDelayedQueue(true)

	// Pre-fill the queue with items to ensure there’s something to dequeue
	for i := 0; i < b.N; i++ {
		pq.Enqueue("default", &Item{ID: uint64(i), Priority: 10, Group: "default"})
	}
}

func BenchmarkDelayedQueueDequeue(b *testing.B) {
	pq := NewDelayedQueue(true)

	// Pre-fill the queue with items to ensure there’s something to dequeue
	for i := 0; i < b.N; i++ {
		pq.Enqueue("default", &Item{ID: uint64(i), Priority: 10, Group: "default"})
	}

	b.ResetTimer() // Reset timer to focus only on Dequeue operation timing

	for i := 0; i < b.N; i++ {
		pq.Dequeue(false)
	}
}

// TestDelayedQueue_MemoryCleanup verifies that all internal references are
// broken after dequeuing all items, which is the necessary condition for GC.
func TestDelayedQueue_MemoryCleanup(t *testing.T) {
	baseMem := forceGCAndLog(t, "Baseline (Start)")

	const numItems = 1000000
	q := NewDelayedQueue(true)

	t.Logf("Attempting to enqueue %d items...", numItems)
	startTime := time.Now()

	for i := range numItems {
		group := "default"
		item := &Item{
			ID:       uint64(i) + 1,
			Priority: int64(i % 100),
			Group:    group,
		}
		q.Enqueue(group, item)
	}
	t.Logf(
		"Enqueue finished in %v. Total queue length: %d capacity: %d",
		time.Since(startTime),
		q.Len(),
		cap(q.queue.items),
	)

	peakMem := forceGCAndLog(t, "Memory AFTER Enqueue")

	if peakMem.HeapObjects <= baseMem.HeapObjects {
		t.Fatal("Test Invalid: Heap object count did not increase after enqueueing 1M items.")
	}

	if peakMem.HeapAlloc < baseMem.HeapAlloc+(50*1024*1024) {
		t.Fatal("Test Invalid: HeapAlloc did not increase significantly.")
	}

	t.Logf("Attempting to dequeue all items...")
	startTime = time.Now()
	dequeuedCount := 0
	for q.Len() > 0 {
		item := q.Dequeue(true)
		assert.NotNil(t, item)
		dequeuedCount++
	}
	t.Logf(
		"Dequeue finished in %v. Dequeued count: %d length: %d capacity: %d",
		time.Since(startTime),
		dequeuedCount,
		len(q.queue.items),
		cap(q.queue.items),
	)
	assert.Equal(t, numItems, dequeuedCount)

	t.Log("Running final garbage collection check...")
	finalMem := forceGCAndLog(t, "Memory AFTER Dequeue + GC")

	assert.Equal(t, 0, len(q.queue.items))
	assert.Equal(t, 0, cap(q.queue.items))

	margin := uint64(2 * 1024 * 1024)

	if finalMem.HeapAlloc > baseMem.HeapAlloc+margin {
		t.Errorf("Possible Memory Leak! Final Heap (%d MB) is significantly higher than Baseline (%d MB)",
			finalMem.HeapAlloc/1024/1024, baseMem.HeapAlloc/1024/1024)
	}

	if finalMem.HeapObjects > baseMem.HeapObjects+1000 {
		t.Errorf("Object Leak! Objects remaining: %d (Baseline was %d)",
			finalMem.HeapObjects, baseMem.HeapObjects)
	}

	assert.Equal(t, uint64(0), q.Len())
	assert.Equal(t, 0, len(q.queue.items))
	assert.Equal(t, 0, cap(q.queue.items))
	assert.Equal(t, 0, len(q.queue.idToIndex))
}
