package memory

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFairRoundRobinQueueSingleCustomer(t *testing.T) {
	tests := []struct {
		name     string
		messages []struct {
			group string
			item  *Item
		}
		expectedOrder []int
	}{
		{
			name: "The same priority",
			messages: []struct {
				group string
				item  *Item
			}{
				{
					group: "customer1",
					item:  &Item{ID: 1, Priority: 10},
				},
				{
					group: "customer1",
					item:  &Item{ID: 2, Priority: 10},
				},
				{
					group: "customer1",
					item:  &Item{ID: 3, Priority: 10},
				},
				{
					group: "customer1",
					item:  &Item{ID: 4, Priority: 10},
				},
				{
					group: "customer1",
					item:  &Item{ID: 5, Priority: 10},
				},
				{
					group: "customer1",
					item:  &Item{ID: 6, Priority: 10},
				},
				{
					group: "customer1",
					item:  &Item{ID: 7, Priority: 10},
				},
				{
					group: "customer1",
					item:  &Item{ID: 8, Priority: 10},
				},
				{
					group: "customer1",
					item:  &Item{ID: 9, Priority: 10},
				},
				{
					group: "customer1",
					item:  &Item{ID: 10, Priority: 10},
				},
			},
			expectedOrder: []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9},
		},
		{
			name: "Different priorities",
			messages: []struct {
				group string
				item  *Item
			}{
				{
					group: "customer1",
					item:  &Item{ID: 1, Priority: 901},
				},
				{
					group: "customer1",
					item:  &Item{ID: 2, Priority: 86},
				},
				{
					group: "customer1",
					item:  &Item{ID: 3, Priority: 1543},
				},
				{
					group: "customer1",
					item:  &Item{ID: 4, Priority: 10123},
				},
				{
					group: "customer1",
					item:  &Item{ID: 5, Priority: 1},
				},
				{
					group: "customer1",
					item:  &Item{ID: 6, Priority: 435},
				},
				{
					group: "customer1",
					item:  &Item{ID: 7, Priority: 3},
				},
				{
					group: "customer1",
					item:  &Item{ID: 8, Priority: 6585},
				},
				{
					group: "customer1",
					item:  &Item{ID: 9, Priority: 54},
				},
				{
					group: "customer1",
					item:  &Item{ID: 10, Priority: 99},
				},
			},
			expectedOrder: []int{4, 6, 8, 1, 9, 5, 0, 2, 7, 3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fq := NewFairRoundRobinQueue(75)

			for i, message := range tt.messages {
				assert.Equal(t, uint64(i), fq.Len())
				fq.Enqueue(message.group, message.item)
				assert.Equal(t, uint64(i+1), fq.Len())
			}

			if tt.name == "Different priorities" {
				for _, expectedIndex := range tt.expectedOrder {
					expectedMessage := tt.messages[expectedIndex]
					message := fq.Get(expectedMessage.group, expectedMessage.item.ID)
					assert.Equal(t, expectedMessage.item.ID, message.ID)

					assert.Equal(t, expectedMessage.item.ID, fq.Dequeue(true).ID)
					assert.Nil(t, fq.Get(expectedMessage.group, expectedMessage.item.ID))
				}
			}
		})
	}
}

func TestFairRoundRobinQueueDelete(t *testing.T) {
	fq := NewFairRoundRobinQueue(8)

	fq.Enqueue("customer1", &Item{ID: 1, Priority: 10})
	fq.Enqueue("customer1", &Item{ID: 4, Priority: 10})
	fq.Enqueue("customer2", &Item{ID: 5, Priority: 10})

	fq.Delete("customer1", 1)

	assert.Equal(t, uint64(4), fq.Dequeue(true).ID)
	assert.Equal(t, uint64(5), fq.Dequeue(true).ID)
}

func TestFairRoundRobinQueueUpdatePriority(t *testing.T) {
	fq := NewFairRoundRobinQueue(8)

	fq.Enqueue("customer1", &Item{ID: 1, Priority: 10})
	fq.Enqueue("customer1", &Item{ID: 4, Priority: 10})
	fq.Enqueue("customer2", &Item{ID: 5, Priority: 10})

	fq.UpdatePriority("customer1", 1, 20)

	assert.Equal(t, uint64(4), fq.Dequeue(true).ID)
	assert.Equal(t, uint64(5), fq.Dequeue(true).ID)
	assert.Equal(t, uint64(1), fq.Dequeue(true).ID)
}

func TestFairRoundRobinQueue(t *testing.T) {
	fq := NewFairRoundRobinQueue(9)

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

	assert.Equal(t, uint64(1), fq.Dequeue(true).ID)
	assert.Equal(t, uint64(5), fq.Dequeue(true).ID)
	assert.Equal(t, uint64(7), fq.Dequeue(true).ID)
	assert.Equal(t, uint64(9), fq.Dequeue(true).ID)
	assert.Equal(t, uint64(10), fq.Dequeue(true).ID)
	assert.Equal(t, uint64(2), fq.Dequeue(true).ID)
	assert.Equal(t, uint64(6), fq.Dequeue(true).ID)
	assert.Equal(t, uint64(8), fq.Dequeue(true).ID)
	assert.Equal(t, uint64(3), fq.Dequeue(true).ID)
	assert.Equal(t, uint64(4), fq.Dequeue(true).ID)
}

func TestFairRoundRobinQueue1(t *testing.T) {
	fq := NewFairRoundRobinQueue(7)

	fq.Enqueue("customer1", &Item{ID: 1, Priority: 10})
	fq.Enqueue("customer1", &Item{ID: 2, Priority: 10})
	fq.Enqueue("customer2", &Item{ID: 3, Priority: 10})

	assert.Equal(t, uint64(1), fq.Dequeue(true).ID)
	assert.Equal(t, uint64(3), fq.Dequeue(true).ID)
	assert.Equal(t, uint64(2), fq.Dequeue(true).ID)
	assert.Nil(t, fq.Dequeue(true))
}

func TestFairRoundRobinQueueWithSubgroups(t *testing.T) {
	fq := NewFairRoundRobinQueue(9)

	fq.Enqueue("customer-1.project-1.user-1", &Item{ID: 1, Priority: 10})
	fq.Enqueue("customer-1.project-1.user-1", &Item{ID: 2, Priority: 10})
	fq.Enqueue("customer-1.project-1.user-2", &Item{ID: 3, Priority: 10})
	fq.Enqueue("customer-1.project-2.user-1", &Item{ID: 4, Priority: 10})
	fq.Enqueue("customer-1.project-2.user-1", &Item{ID: 5, Priority: 10})
	fq.Enqueue("customer-1.project-2.user-2", &Item{ID: 6, Priority: 10})
	fq.Enqueue("customer-1.project-3.user-1", &Item{ID: 7, Priority: 10})
	fq.Enqueue("customer-1.project-3.user-2", &Item{ID: 8, Priority: 10})
	fq.Enqueue("customer-2", &Item{ID: 9, Priority: 10})
	fq.Enqueue("customer-3.project-1", &Item{ID: 10, Priority: 10})
	fq.Enqueue("customer-3.project-1", &Item{ID: 11, Priority: 10})
	fq.Enqueue("customer-3.project-2", &Item{ID: 12, Priority: 10})
	fq.Enqueue("customer-4.project-1.user-1", &Item{ID: 13, Priority: 10})
	fq.Enqueue("customer-4.project-1.user-2", &Item{ID: 14, Priority: 10})
	fq.Enqueue("customer-4.project-2.user-2", &Item{ID: 15, Priority: 10})

	assert.Equal(t, uint64(1), fq.Dequeue(true).ID)
	assert.Equal(t, uint64(9), fq.Dequeue(true).ID)
	assert.Equal(t, uint64(10), fq.Dequeue(true).ID)
	assert.Equal(t, uint64(13), fq.Dequeue(true).ID)
	assert.Equal(t, uint64(4), fq.Dequeue(true).ID)
	assert.Equal(t, uint64(12), fq.Dequeue(true).ID)
	assert.Equal(t, uint64(15), fq.Dequeue(true).ID)
	assert.Equal(t, uint64(7), fq.Dequeue(true).ID)
	assert.Equal(t, uint64(11), fq.Dequeue(true).ID)
	assert.Equal(t, uint64(14), fq.Dequeue(true).ID)
	assert.Equal(t, uint64(3), fq.Dequeue(true).ID)
	assert.Equal(t, uint64(6), fq.Dequeue(true).ID)
	assert.Equal(t, uint64(8), fq.Dequeue(true).ID)
	assert.Equal(t, uint64(2), fq.Dequeue(true).ID)
	assert.Equal(t, uint64(5), fq.Dequeue(true).ID)
}

func TestFairRoundRobinQueueMaxUnacked(t *testing.T) {
	queue := NewFairRoundRobinQueue(2)

	// Enqueue items for a single group
	queue.Enqueue("group1", &Item{ID: 1, Priority: 10, Group: "group1"})
	queue.Enqueue("group1", &Item{ID: 2, Priority: 10, Group: "group1"})
	queue.Enqueue("group1", &Item{ID: 3, Priority: 10, Group: "group1"})
	queue.Enqueue("group1", &Item{ID: 4, Priority: 10, Group: "group1"})

	// Dequeue without acking until max unacked is reached
	item1 := queue.Dequeue(false)
	assert.NotNil(t, item1)

	item2 := queue.Dequeue(false)
	assert.NotNil(t, item2)

	item3 := queue.Dequeue(false)
	assert.Nil(t, item3) // Should be nil because max unacked is reached

	// Ack one message
	err := queue.UpdateWeights("group1", item1.ID)
	assert.NoError(t, err)

	// Should be able to dequeue again
	item4 := queue.Dequeue(false)
	assert.NotNil(t, item4)
}

func BenchmarkFairRoundRobinQueueEnqueue(b *testing.B) {
	fq := NewFairRoundRobinQueue(8)

	customers := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		customers[i] = fmt.Sprintf("customer-%d", i)
	}

	b.ResetTimer() // Reset timer to focus only on Dequeue operation timing

	// Pre-fill the queue with items to ensure there’s something to dequeue
	for i := 0; i < b.N; i++ {
		customer := customers[rand.Intn(len(customers))]
		fq.Enqueue(customer, &Item{ID: uint64(i), Priority: 10})
	}
}

func BenchmarkFairRoundRobinQueueDequeue(b *testing.B) {
	fq := NewFairRoundRobinQueue(8)

	customers := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		customers[i] = fmt.Sprintf("customer-%d", i)
	}

	// Pre-fill the queue with items to ensure there’s something to dequeue
	for i := 0; i < b.N; i++ {
		customer := customers[rand.Intn(len(customers))]
		fq.Enqueue(customer, &Item{ID: uint64(i), Priority: 10})
	}

	b.ResetTimer() // Reset timer to focus only on Dequeue operation timing

	for i := 0; i < b.N; i++ {
		fq.Dequeue(true)
	}
}

// TestFairRoundRobinQueue_MemoryCleanup verifies that all internal references are
// broken after dequeuing all items, which is the necessary condition for GC.
func TestFairRoundRobinQueue_MemoryCleanup(t *testing.T) {
	const numItems = 1000000
	groups := []string{
		"customer-1.group-a.user-1",
		"customer-1.group-a.user-2",
		"customer-1.group-b.user-1",
		"customer-1.group-b.user-2",
		"customer-1.group-c.user-1",
		"customer-1.group-c.user-2",
		"customer-1.group-d.user-1",
		"customer-1.group-d.user-2",
		"customer-2.group-a.user-1",
		"customer-2.group-a.user-2",
		"customer-2.group-b.user-1",
		"customer-2.group-b.user-2",
		"customer-2.group-c.user-1",
		"customer-2.group-c.user-2",
		"customer-2.group-d.user-1",
		"customer-2.group-d.user-2",
	}
	q := NewFairRoundRobinQueue(1000)

	t.Logf("Attempting to enqueue %d items...", numItems)
	startTime := time.Now()
	for i := range numItems {
		group := groups[i%len(groups)]
		item := &Item{
			ID:       uint64(i) + 1,
			Priority: int64(i % 100),
			Group:    group,
		}
		q.Enqueue(group, item)
	}
	t.Logf("Enqueue finished in %v. Total queue length: %d", time.Since(startTime), q.Len())

	forceGCAndLog(t, "Memory AFTER Enqueue")

	t.Logf("Attempting to dequeue all items...")
	startTime = time.Now()
	dequeuedCount := 0
	for q.Len() > 0 {
		item := q.Dequeue(true)
		assert.NotNil(t, item)
		dequeuedCount++
	}
	assert.Equal(t, numItems, dequeuedCount)
	t.Logf("Dequeue finished in %v. Total queue length: %d", time.Since(startTime), dequeuedCount)

	t.Log("Running final garbage collection check...")
	forceGCAndLog(t, "Memory AFTER Dequeue + GC")

	assert.Equal(t, uint64(0), q.Len(), "queue length should be zero")
	assert.Equal(t, uint64(0), q.totalMessages, "totalMessages should be zero")
	assert.Equal(t, 0, len(q.groupKeyToLeaf), "groupKeyToLeaf should be empty")
	assert.Equal(t, 0, len(q.unackedByGroup), "unackedByGroup should be empty")
}
