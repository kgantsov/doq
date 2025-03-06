package queue

import (
	"os"
	"testing"

	"github.com/dgraph-io/badger/v4"
	"github.com/kgantsov/doq/pkg/config"
	"github.com/kgantsov/doq/pkg/storage"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
)

func TestNewQueueManager(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "db*")
	defer os.RemoveAll(tmpDir)

	opts := badger.DefaultOptions(tmpDir)
	db, err := badger.Open(opts)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}
	defer db.Close()

	queueManager := NewQueueManager(
		storage.NewBadgerStore(db),
		&config.Config{
			Queue: config.QueueConfig{
				AcknowledgementCheckInterval: 1,
				QueueStats:                   config.QueueStatsConfig{WindowSide: 10},
			},
		},
		nil,
	)

	queue1, err := queueManager.CreateQueue("delayed", "queue_1")
	assert.Nil(t, err)

	queue1, err = queueManager.CreateQueue("delayed", "queue_1")
	assert.Nil(t, err)

	q1m1, err := queue1.Enqueue(1, "default", 10, "queue 1 message 1", nil)
	assert.Nil(t, err)
	q1m2, err := queue1.Enqueue(2, "default", 5, "queue 1 message 2", nil)
	assert.Nil(t, err)

	queue2, err := queueManager.CreateQueue("delayed", "queue_2")
	assert.Nil(t, err)

	_, err = queue2.Enqueue(1, "default", 20, "queue 2 message 1", nil)
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

	err = queueManager.DeleteQueue("queue_2")
	assert.Nil(t, err)

	queue2, err = queueManager.GetQueue("queue_2")
	assert.Nil(t, err)

	m, err = queue2.Dequeue(true)
	assert.NotNil(t, err)
	assert.Nil(t, m)
}

func TestQueueManagerGetQueue(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "db*")
	defer os.RemoveAll(tmpDir)

	opts := badger.DefaultOptions(tmpDir)
	db, err := badger.Open(opts)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}
	defer db.Close()

	queueManager := NewQueueManager(
		storage.NewBadgerStore(db),
		&config.Config{
			Queue: config.QueueConfig{
				AcknowledgementCheckInterval: 1,
				QueueStats:                   config.QueueStatsConfig{WindowSide: 10},
			},
		},
		nil,
	)

	queue1, err := queueManager.CreateQueue("delayed", "queue_1")
	assert.Nil(t, err)

	queue2, err := queueManager.CreateQueue("delayed", "queue_2")
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

func TestQueueManagerGetQueues(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "db*")
	defer os.RemoveAll(tmpDir)

	opts := badger.DefaultOptions(tmpDir)
	db, err := badger.Open(opts)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}
	defer db.Close()

	queueManager := NewQueueManager(
		storage.NewBadgerStore(db),
		&config.Config{
			Queue: config.QueueConfig{
				AcknowledgementCheckInterval: 1,
				QueueStats:                   config.QueueStatsConfig{WindowSide: 10},
			},
		},
		nil,
	)

	_, err = queueManager.CreateQueue("delayed", "queue_1")
	assert.Nil(t, err)

	_, err = queueManager.CreateQueue("delayed", "queue_2")
	assert.Nil(t, err)

	queues := queueManager.GetQueues()
	assert.Len(t, queues, 2)

	for _, q := range queues {
		assert.NotNil(t, q)

		assert.Contains(t, []string{"queue_1", "queue_2"}, q.Name)
	}
}
