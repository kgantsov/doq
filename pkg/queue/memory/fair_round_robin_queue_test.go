package memory

import (
	"testing"

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

	// Pre-fill the queue with items to ensure there’s something to dequeue
	for i := 0; i < b.N; i++ {
		fq.Enqueue("customer1", &Item{ID: uint64(i), Priority: 10})
	}
}

func BenchmarkFairRoundRobinQueueDequeue(b *testing.B) {
	fq := NewFairRoundRobinQueue(8)

	// Pre-fill the queue with items to ensure there’s something to dequeue
	for i := 0; i < b.N; i++ {
		fq.Enqueue("customer1", &Item{ID: uint64(i), Priority: 10})
	}

	b.ResetTimer() // Reset timer to focus only on Dequeue operation timing

	for i := 0; i < b.N; i++ {
		fq.Dequeue(true)
	}
}
