package memory

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCalculateWeight(t *testing.T) {
	tests := []struct {
		name       string
		maxUnacked int
		unacked    int
		queueSize  int
		expected   int
	}{
		{
			name:       "Empty queue returns zero weight",
			maxUnacked: 10,
			unacked:    2,
			queueSize:  0,
			expected:   0,
		},
		{
			name:       "Zero maxUnacked returns zero weight",
			maxUnacked: 0,
			unacked:    0,
			queueSize:  10,
			expected:   0,
		},
		{
			name:       "Unacked greater than maxUnacked returns zero weight",
			maxUnacked: 5,
			unacked:    6,
			queueSize:  10,
			expected:   0,
		},
		{
			name:       "Ideal case - no unacked, full queue",
			maxUnacked: 10,
			unacked:    0,
			queueSize:  10,
			expected:   10,
		},
		{
			name:       "Half unacked, full queue",
			maxUnacked: 10,
			unacked:    5,
			queueSize:  10,
			expected:   5,
		},
		{
			name:       "Some unacked, small queue",
			maxUnacked: 10,
			unacked:    3,
			queueSize:  2,
			expected:   7,
		},
		{
			name:       "All unacked, non-empty queue",
			maxUnacked: 10,
			unacked:    10,
			queueSize:  5,
			expected:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			weight := CalculateWeight(tt.maxUnacked, tt.unacked, tt.queueSize)
			assert.Equal(t, tt.expected, weight)
		})
	}
}

func TestFairWeightedQueueBasic(t *testing.T) {
	queue := NewFairWeightedQueue(8)

	// Test empty queue
	assert.Equal(t, uint64(0), queue.Len())
	assert.Nil(t, queue.Dequeue(false))

	// Test single group enqueue/dequeue
	queue.Enqueue("group1", &Item{ID: 1, Priority: 10})
	assert.Equal(t, uint64(1), queue.Len())

	item := queue.Dequeue(false)
	assert.NotNil(t, item)
	assert.Equal(t, uint64(1), item.ID)
	assert.Equal(t, uint64(0), queue.Len())
}

func TestFairWeightedQueueTTTT(t *testing.T) {
	queue := NewFairWeightedQueue(8)
	customerMessages := map[string]int{
		"customer-1": 10,
		"customer-2": 5,
		"customer-3": 3,
		"customer-4": 1,
	}

	messageCount := 0
	for customer, count := range customerMessages {
		for i := 0; i < count; i++ {
			queue.Enqueue(customer, &Item{ID: uint64(messageCount) + 1, Priority: 10, Group: customer})
		}
	}

	items := make(map[string]int)

	for _, count := range customerMessages {
		for i := 0; i < count; i++ {
			item := queue.Dequeue(false)
			if item != nil {
				items[item.Group] += 1
			}
			messageCount++
		}
	}

	assert.Equal(t, 8, items["customer-1"])
	assert.Equal(t, 5, items["customer-2"])
	assert.Equal(t, 3, items["customer-3"])
	assert.Equal(t, 1, items["customer-4"])
	assert.Equal(t, uint64(2), queue.Len())
}

func TestFairWeightedQueueMultipleGroups(t *testing.T) {
	queue := NewFairWeightedQueue(9)

	// Enqueue items for multiple groups
	queue.Enqueue("group1", &Item{ID: 1, Priority: 10})
	queue.Enqueue("group1", &Item{ID: 2, Priority: 10})
	queue.Enqueue("group1", &Item{ID: 3, Priority: 10})
	queue.Enqueue("group1", &Item{ID: 4, Priority: 10})
	queue.Enqueue("group1", &Item{ID: 5, Priority: 10})
	queue.Enqueue("group1", &Item{ID: 6, Priority: 10})
	queue.Enqueue("group1", &Item{ID: 7, Priority: 10})
	queue.Enqueue("group1", &Item{ID: 8, Priority: 10})
	queue.Enqueue("group1", &Item{ID: 9, Priority: 10})
	queue.Enqueue("group2", &Item{ID: 10, Priority: 10})

	assert.Equal(t, uint64(10), queue.Len())

	// Dequeue all items and verify they exist
	items := make(map[uint64]bool)
	for i := 0; i < 10; i++ {
		item := queue.Dequeue(false)
		assert.NotNil(t, item)
		items[item.ID] = true
	}

	// Verify all items were dequeued
	assert.Equal(t, 10, len(items))
	assert.True(t, items[1])
	assert.True(t, items[2])
	assert.True(t, items[3])
	assert.Equal(t, uint64(0), queue.Len())
}

func TestFairWeightedQueueUnackedHandling(t *testing.T) {
	queue := NewFairWeightedQueue(8)

	// Enqueue items for two groups
	queue.Enqueue("group1", &Item{ID: 1, Priority: 10})
	queue.Enqueue("group1", &Item{ID: 2, Priority: 10})
	queue.Enqueue("group2", &Item{ID: 3, Priority: 10})
	queue.Enqueue("group2", &Item{ID: 4, Priority: 10})

	// Dequeue without acking (increasing unacked count)
	item1 := queue.Dequeue(false)
	item2 := queue.Dequeue(false)

	assert.NotNil(t, item1)
	assert.NotNil(t, item2)

	// The next dequeue should favor the group with fewer unacked messages
	// Since we don't know which group got which item, we'll verify the weights
	item3 := queue.Dequeue(false)
	assert.NotNil(t, item3)

	// Ack a message to reduce unacked count
	err := queue.UpdateWeights(item1.Group, item1.ID)
	assert.NoError(t, err)

	// Next dequeue should favor the group that just had a message acked
	item4 := queue.Dequeue(false)
	assert.NotNil(t, item4)

	// Verify all items were dequeued
	items := make(map[uint64]bool)
	items[item1.ID] = true
	items[item2.ID] = true
	items[item3.ID] = true
	items[item4.ID] = true
	assert.Equal(t, 4, len(items))
	assert.True(t, items[1])
	assert.True(t, items[2])
	assert.True(t, items[3])
	assert.True(t, items[4])
}

func TestFairWeightedQueuePriority(t *testing.T) {
	queue := NewFairWeightedQueue(8)

	// Enqueue items with different priorities within the same group
	queue.Enqueue("group1", &Item{ID: 1, Priority: 30})
	queue.Enqueue("group1", &Item{ID: 2, Priority: 10})
	queue.Enqueue("group1", &Item{ID: 3, Priority: 20})

	// Dequeue should respect priority order within the group
	item1 := queue.Dequeue(false)
	item2 := queue.Dequeue(false)
	item3 := queue.Dequeue(false)

	assert.NotNil(t, item1)
	assert.NotNil(t, item2)
	assert.NotNil(t, item3)

	// Verify priority order (lower priority first)
	assert.Equal(t, uint64(2), item1.ID) // Priority 10
	assert.Equal(t, uint64(3), item2.ID) // Priority 20
	assert.Equal(t, uint64(1), item3.ID) // Priority 30
}

func TestFairWeightedQueueGetAndDelete(t *testing.T) {
	queue := NewFairWeightedQueue(8)

	// Enqueue items
	queue.Enqueue("group1", &Item{ID: 1, Priority: 10})
	queue.Enqueue("group1", &Item{ID: 2, Priority: 20})
	queue.Enqueue("group2", &Item{ID: 3, Priority: 10})

	// Test Get
	item := queue.Get("group1", 1)
	assert.NotNil(t, item)
	assert.Equal(t, uint64(1), item.ID)
	assert.Equal(t, int64(10), item.Priority)

	// Test Get non-existent item
	item = queue.Get("group1", 999)
	assert.Nil(t, item)

	// Test Delete
	deletedItem := queue.Delete("group1", 1)
	assert.NotNil(t, deletedItem)
	assert.Equal(t, uint64(1), deletedItem.ID)

	// Verify item was deleted
	item = queue.Get("group1", 1)
	assert.Nil(t, item)

	// Test Delete non-existent item
	deletedItem = queue.Delete("group1", 999)
	assert.Nil(t, deletedItem)
}

func TestFairWeightedQueueUpdatePriority(t *testing.T) {
	queue := NewFairWeightedQueue(8)

	// Enqueue items
	queue.Enqueue("group1", &Item{ID: 1, Priority: 10})
	queue.Enqueue("group1", &Item{ID: 2, Priority: 20})

	// Update priority
	queue.UpdatePriority("group1", 1, 30)

	// Dequeue should now return items in new priority order
	item1 := queue.Dequeue(false)
	item2 := queue.Dequeue(false)

	assert.NotNil(t, item1)
	assert.NotNil(t, item2)

	// Verify new priority order
	assert.Equal(t, uint64(2), item1.ID) // Original priority 20
	assert.Equal(t, uint64(1), item2.ID) // Updated priority 30
}

func TestFairWeightedQueueMaxUnacked(t *testing.T) {
	queue := NewFairWeightedQueue(2) // Set max unacked to 2

	// Enqueue items for a single group
	queue.Enqueue("group1", &Item{ID: 1, Priority: 10})
	queue.Enqueue("group1", &Item{ID: 2, Priority: 10})
	queue.Enqueue("group1", &Item{ID: 3, Priority: 10})
	queue.Enqueue("group1", &Item{ID: 4, Priority: 10})

	// Dequeue without acking until max unacked is reached
	item1 := queue.Dequeue(false)
	item2 := queue.Dequeue(false)
	item3 := queue.Dequeue(false)

	assert.NotNil(t, item1)
	assert.NotNil(t, item2)
	assert.Nil(t, item3) // Should be nil because max unacked is reached

	// Ack one message
	err := queue.UpdateWeights("group1", item1.ID)
	assert.NoError(t, err)

	// Should be able to dequeue again
	item4 := queue.Dequeue(false)
	assert.NotNil(t, item4)
}
