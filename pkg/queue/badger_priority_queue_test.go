package queue

import (
	"testing"

	"github.com/dgraph-io/badger/v4"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
)

func TestBadgerPriorityQueue(t *testing.T) {
	tests := []struct {
		name     string
		messages []struct {
			Priority int
			Content  string
		}
		expected []string
	}{
		{
			name: "First test",
			messages: []struct {
				Priority int
				Content  string
			}{
				{Priority: 10, Content: "test 1"},
				{Priority: 50, Content: "test 2"},
				{Priority: 20, Content: "test 3"},
				{Priority: 70, Content: "test 4"},
				{Priority: 100, Content: "test 5"},
				{Priority: 7, Content: "test 6"},
				{Priority: 4, Content: "test 7"},
				{Priority: 2, Content: "test 8"},
				{Priority: 5, Content: "test 9"},
			},
			expected: []string{"test 8", "test 7", "test 9", "test 6", "test 1", "test 3", "test 2", "test 4", "test 5"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := badger.DefaultOptions("/tmp/badger")
			db, err := badger.Open(opts)
			if err != nil {
				log.Fatal().Msg(err.Error())
			}

			pq := NewBadgerPriorityQueue(db, "test_queue")

			for i, m := range tt.messages {
				pq.Enqueue(m.Priority, m.Content)
				assert.Equal(t, i+1, pq.Len())
			}
			assert.Equal(t, len(tt.messages), pq.Len())

			for i := 0; i < len(tt.messages); i++ {
				m, err := pq.Dequeue()
				assert.Nil(t, err)

				assert.Equal(t, len(tt.messages)-(i+1), pq.Len())

				assert.Equal(t, tt.expected[i], m.Content)
			}

			db.Close()
		})
	}
}

func TestBadgerPriorityQueueEmptyQueue(t *testing.T) {
	opts := badger.DefaultOptions("/tmp/badger")
	db, err := badger.Open(opts)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}
	defer db.Close()

	pq := NewBadgerPriorityQueue(db, "test_queue")

	m, err := pq.Dequeue()
	assert.EqualError(t, err, ErrEmptyQueue.Error())
	assert.Nil(t, m)
}

func TestBadgerPriorityQueueChangePriority(t *testing.T) {
	opts := badger.DefaultOptions("/tmp/badger")
	db, err := badger.Open(opts)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}

	pq := NewBadgerPriorityQueue(db, "test_queue")

	m1, err := pq.Enqueue(10, "test 1")
	assert.Nil(t, err)

	m1, err = pq.GetByID(m1.ID)
	assert.Nil(t, err)
	assert.Equal(t, "test 1", m1.Content)
	assert.Equal(t, 10, m1.Priority)

	m2, err := pq.Enqueue(20, "test 2")
	assert.Nil(t, err)

	m2, err = pq.GetByID(m2.ID)
	assert.Nil(t, err)
	assert.Equal(t, "test 2", m2.Content)
	assert.Equal(t, 20, m2.Priority)

	m3, err := pq.Enqueue(30, "test 3")
	assert.Nil(t, err)

	m3, err = pq.GetByID(m3.ID)
	assert.Nil(t, err)
	assert.Equal(t, "test 3", m3.Content)
	assert.Equal(t, 30, m3.Priority)

	m4, err := pq.Enqueue(40, "test 4")
	assert.Nil(t, err)

	m4, err = pq.GetByID(m4.ID)
	assert.Nil(t, err)
	assert.Equal(t, "test 4", m4.Content)
	assert.Equal(t, 40, m4.Priority)

	m5, err := pq.Enqueue(50, "test 5")
	assert.Nil(t, err)

	m5, err = pq.GetByID(m5.ID)
	assert.Nil(t, err)
	assert.Equal(t, "test 5", m5.Content)
	assert.Equal(t, 50, m5.Priority)

	err = pq.UpdatePriority(m1.ID, 60)
	assert.Nil(t, err)

	m1, err = pq.GetByID(m1.ID)
	assert.Nil(t, err)
	assert.Equal(t, "test 1", m1.Content)
	assert.Equal(t, 60, m1.Priority)

	err = pq.UpdatePriority(m4.ID, 55)
	assert.Nil(t, err)

	m4, err = pq.GetByID(m4.ID)
	assert.Nil(t, err)
	assert.Equal(t, "test 4", m4.Content)
	assert.Equal(t, 55, m4.Priority)

	m, err := pq.Dequeue()
	assert.Nil(t, err)

	assert.Equal(t, "test 2", m.Content)

	m, err = pq.Dequeue()
	assert.Nil(t, err)

	assert.Equal(t, "test 3", m.Content)

	m, err = pq.Dequeue()
	assert.Nil(t, err)

	assert.Equal(t, "test 5", m.Content)

	m, err = pq.Dequeue()

	assert.Nil(t, err)

	assert.Equal(t, "test 4", m.Content)

	m, err = pq.Dequeue()
	assert.Nil(t, err)

	assert.Equal(t, "test 1", m.Content)

	db.Close()
}
