package queue

import (
	"testing"

	"github.com/dgraph-io/badger/v4"
	"github.com/kgantsov/doq/pkg/config"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
)

func TestNewQueueManager(t *testing.T) {
	opts := badger.DefaultOptions("/tmp/badger")
	db, err := badger.Open(opts)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}
	defer db.Close()

	queueManager := NewQueueManager(
		db,
		&config.Config{
			Queue: config.QueueConfig{
				AcknowledgementCheckInterval: 1,
			},
		},
		nil,
	)

	queue1, err := queueManager.Create("delayed", "queue_1")
	assert.Nil(t, err)

	queue1, err = queueManager.Create("delayed", "queue_1")
	assert.Nil(t, err)

	q1m1, err := queue1.Enqueue(1, "default", 10, "queue 1 message 1", nil)
	assert.Nil(t, err)
	q1m2, err := queue1.Enqueue(2, "default", 5, "queue 1 message 2", nil)
	assert.Nil(t, err)

	queue2, err := queueManager.Create("delayed", "queue_2")
	assert.Nil(t, err)

	q2m1, err := queue2.Enqueue(1, "default", 20, "queue 2 message 1", nil)
	assert.Nil(t, err)
	q2m2, err := queue2.Enqueue(2, "default", 15, "queue 2 message 2", nil)
	assert.Nil(t, err)

	m, err := queue1.Dequeue(true)
	assert.Nil(t, err)
	assert.Equal(t, q1m2.ID, m.ID)
	assert.Equal(t, q1m2.Content, m.Content)

	m, err = queue1.Dequeue(true)
	assert.Nil(t, err)
	assert.Equal(t, q1m1.ID, m.ID)
	assert.Equal(t, q1m1.Content, m.Content)

	m, err = queue2.Dequeue(true)
	assert.Nil(t, err)
	assert.Equal(t, q2m2.ID, m.ID)
	assert.Equal(t, q2m2.Content, m.Content)

	m, err = queue2.Dequeue(true)
	assert.Nil(t, err)
	assert.Equal(t, q2m1.ID, m.ID)
	assert.Equal(t, q2m1.Content, m.Content)
}

func TestQueueManagerGetQueue(t *testing.T) {
	opts := badger.DefaultOptions("/tmp/badger10")
	db, err := badger.Open(opts)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}
	defer db.Close()

	queueManager := NewQueueManager(
		db,
		&config.Config{
			Queue: config.QueueConfig{
				AcknowledgementCheckInterval: 1,
			},
		},
		nil,
	)

	queue1, err := queueManager.Create("delayed", "queue_1")
	assert.Nil(t, err)

	queue2, err := queueManager.Create("delayed", "queue_2")
	assert.Nil(t, err)

	queue1, err = queueManager.GetQueue("queue_1")
	assert.Nil(t, err)

	q1m1, err := queue1.Enqueue(1, "default", 10, "queue 1 message 1", nil)
	assert.Nil(t, err)
	q1m2, err := queue1.Enqueue(2, "default", 5, "queue 1 message 2", map[string]string{})
	assert.Nil(t, err)

	queue2, err = queueManager.GetQueue("queue_2")
	assert.Nil(t, err)

	q2m1, err := queue2.Enqueue(1, "default", 20, "queue 2 message 1", nil)
	assert.Nil(t, err)
	q2m2, err := queue2.Enqueue(2, "default", 15, "queue 2 message 2", nil)
	assert.Nil(t, err)

	m, err := queue1.Dequeue(true)
	assert.Nil(t, err)
	assert.Equal(t, q1m2.ID, m.ID)
	assert.Equal(t, q1m2.Content, m.Content)

	queue1, err = queueManager.GetQueue("queue_1")
	assert.Nil(t, err)

	m, err = queue1.Dequeue(true)
	assert.Nil(t, err)
	assert.Equal(t, q1m1.ID, m.ID)
	assert.Equal(t, q1m1.Content, m.Content)

	m, err = queue2.Dequeue(true)
	assert.Nil(t, err)
	assert.Equal(t, q2m2.ID, m.ID)
	assert.Equal(t, q2m2.Content, m.Content)

	m, err = queue2.Dequeue(true)
	assert.Nil(t, err)
	assert.Equal(t, q2m1.ID, m.ID)
	assert.Equal(t, q2m1.Content, m.Content)
}
