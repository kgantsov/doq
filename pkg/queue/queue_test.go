package queue

import (
	"os"
	"testing"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/kgantsov/doq/pkg/config"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQueue(t *testing.T) {
	tests := []struct {
		name     string
		messages []struct {
			Priority int64
			Content  string
			Metadata map[string]string
		}
		expected []string
	}{
		{
			name: "First test",
			messages: []struct {
				Priority int64
				Content  string
				Metadata map[string]string
			}{
				{Priority: 10, Content: "test 1", Metadata: map[string]string{"retry": "0"}},
				{Priority: 50, Content: "test 2", Metadata: map[string]string{"retry": "0"}},
				{Priority: 20, Content: "test 3", Metadata: map[string]string{"retry": "0"}},
				{Priority: 70, Content: "test 4", Metadata: map[string]string{"retry": "0"}},
				{Priority: 100, Content: "test 5", Metadata: map[string]string{"retry": "0"}},
				{Priority: 7, Content: "test 6", Metadata: map[string]string{"retry": "0"}},
				{Priority: 4, Content: "test 7", Metadata: map[string]string{"retry": "0"}},
				{Priority: 2, Content: "test 8", Metadata: map[string]string{"retry": "0"}},
				{Priority: 5, Content: "test 9", Metadata: map[string]string{"retry": "0"}},
			},
			expected: []string{"test 8", "test 7", "test 9", "test 6", "test 1", "test 3", "test 2", "test 4", "test 5"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, _ := os.MkdirTemp("", "db*")
			defer os.RemoveAll(tmpDir)
			opts := badger.DefaultOptions(tmpDir)
			db, err := badger.Open(opts)
			if err != nil {
				log.Fatal().Msg(err.Error())
			}

			pq := NewQueue(
				db,
				&config.Config{Queue: config.QueueConfig{
					AcknowledgementCheckInterval: 1,
					QueueStats:                   config.QueueStatsConfig{WindowSide: 10},
				}},
				nil,
			)
			pq.Create("delayed", "test_queue")

			for i, m := range tt.messages {
				pq.Enqueue(uint64(i+1), "default", m.Priority, m.Content, m.Metadata)
				assert.Equal(t, i+1, pq.Len())
			}
			assert.Equal(t, len(tt.messages), pq.Len())

			for i := 0; i < len(tt.messages); i++ {
				m, err := pq.Dequeue(true)
				assert.Nil(t, err)

				assert.Equal(t, len(tt.messages)-(i+1), pq.Len())

				assert.Equal(t, tt.expected[i], m.Content)
			}

			db.Close()
		})
	}
}

func TestQueueEmptyQueue(t *testing.T) {
	dirname, err := os.MkdirTemp("", "store")
	require.NoError(t, err)

	opts := badger.DefaultOptions(dirname)
	db, err := badger.Open(opts)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}
	defer db.Close()

	pq := NewQueue(
		db,
		&config.Config{Queue: config.QueueConfig{
			AcknowledgementCheckInterval: 1,
			QueueStats:                   config.QueueStatsConfig{WindowSide: 10},
		}},
		nil,
	)
	pq.Create("delayed", "test_queue_stored")

	m, err := pq.Dequeue(true)
	assert.EqualError(t, err, ErrEmptyQueue.Error())
	assert.Nil(t, m)
}

func TestQueueLoad(t *testing.T) {
	dirname, err := os.MkdirTemp("", "store")
	require.NoError(t, err)

	opts := badger.DefaultOptions(dirname)
	db, err := badger.Open(opts)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}
	defer db.Close()

	pq := NewQueue(
		db,
		&config.Config{Queue: config.QueueConfig{
			AcknowledgementCheckInterval: 1,
			QueueStats:                   config.QueueStatsConfig{WindowSide: 10},
		}},
		nil,
	)
	err = pq.Create("delayed", "test_queue")
	assert.Nil(t, err)
	assert.Equal(t, "delayed", pq.config.Type)
	assert.Equal(t, "test_queue", pq.config.Name)

	pq.Enqueue(1, "default", 10, "test 1", map[string]string{"retry": "0"})
	pq.Enqueue(2, "default", 5, "test 2", map[string]string{"retry": "0"})
	pq.Enqueue(3, "default", 8, "test 3", map[string]string{"retry": "0"})
	pq.Enqueue(4, "default", 1, "test 4", map[string]string{"retry": "0"})

	pq1 := NewQueue(
		db,
		&config.Config{Queue: config.QueueConfig{
			AcknowledgementCheckInterval: 1,
			QueueStats:                   config.QueueStatsConfig{WindowSide: 10},
		}},
		nil,
	)
	err = pq1.Load("test_queue", true)
	assert.Nil(t, err)
	assert.Equal(t, "delayed", pq1.config.Type)
	assert.Equal(t, "test_queue", pq1.config.Name)

	m, err := pq1.Dequeue(true)
	assert.Nil(t, err)
	assert.Equal(t, int64(1), m.Priority)
	assert.Equal(t, "test 4", m.Content)

	m, err = pq1.Dequeue(true)
	assert.Nil(t, err)
	assert.Equal(t, int64(5), m.Priority)
	assert.Equal(t, "test 2", m.Content)

	m, err = pq1.Dequeue(true)
	assert.Nil(t, err)
	assert.Equal(t, int64(8), m.Priority)
	assert.Equal(t, "test 3", m.Content)

	m, err = pq1.Dequeue(true)
	assert.Nil(t, err)
	assert.Equal(t, int64(10), m.Priority)
	assert.Equal(t, "test 1", m.Content)
}

func TestQueueDeleteQueue(t *testing.T) {
	dirname, err := os.MkdirTemp("", "store")
	require.NoError(t, err)

	opts := badger.DefaultOptions(dirname)
	db, err := badger.Open(opts)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}
	defer db.Close()

	pq := NewQueue(
		db,
		&config.Config{Queue: config.QueueConfig{
			AcknowledgementCheckInterval: 1,
			QueueStats:                   config.QueueStatsConfig{WindowSide: 10},
		}},
		nil,
	)
	err = pq.Create("delayed", "test_queue")
	assert.Nil(t, err)
	assert.Equal(t, "delayed", pq.config.Type)
	assert.Equal(t, "test_queue", pq.config.Name)

	pq.Enqueue(1, "default", 10, "test 1", map[string]string{"retry": "0"})
	pq.Enqueue(2, "default", 5, "test 2", map[string]string{"retry": "0"})
	pq.Enqueue(3, "default", 8, "test 3", map[string]string{"retry": "0"})
	pq.Enqueue(4, "default", 1, "test 4", map[string]string{"retry": "0"})

	err = pq.DeleteQueue()
	assert.Nil(t, err)
}

func TestQueueDelete(t *testing.T) {
	dirname, err := os.MkdirTemp("", "store")
	require.NoError(t, err)

	opts := badger.DefaultOptions(dirname)
	db, err := badger.Open(opts)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}
	defer db.Close()

	pq := NewQueue(
		db,
		&config.Config{Queue: config.QueueConfig{
			AcknowledgementCheckInterval: 1,
			QueueStats:                   config.QueueStatsConfig{WindowSide: 10},
		}},
		nil,
	)
	err = pq.Create("delayed", "test_queue")
	assert.Nil(t, err)
	assert.Equal(t, "delayed", pq.config.Type)
	assert.Equal(t, "test_queue", pq.config.Name)

	pq.Enqueue(1, "default", 10, "test 1", map[string]string{"retry": "0"})
	pq.Enqueue(2, "default", 5, "test 2", map[string]string{"retry": "0"})
	pq.Enqueue(3, "default", 8, "test 3", map[string]string{"retry": "0"})
	pq.Enqueue(4, "default", 1, "test 4", map[string]string{"retry": "0"})

	err = pq.Delete(2)
	assert.Nil(t, err)

	m, err := pq.Get(2)
	assert.Nil(t, m)
	assert.EqualError(t, err, ErrMessageNotFound.Error())
}

func TestQueueChangePriority(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "db*")
	defer os.RemoveAll(tmpDir)

	opts := badger.DefaultOptions(tmpDir)
	db, err := badger.Open(opts)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}

	pq := NewQueue(
		db,
		&config.Config{Queue: config.QueueConfig{
			AcknowledgementCheckInterval: 1,
			QueueStats:                   config.QueueStatsConfig{WindowSide: 10},
		}},
		nil,
	)
	pq.Create("delayed", "test_queue")

	m1, err := pq.Enqueue(1, "default", 10, "test 1", map[string]string{"retry": "1"})
	assert.Nil(t, err)

	m1, err = pq.Get(m1.ID)
	assert.Nil(t, err)
	assert.Equal(t, "test 1", m1.Content)
	assert.Equal(t, int64(10), m1.Priority)
	assert.Equal(t, "1", m1.Metadata["retry"])

	m2, err := pq.Enqueue(2, "default", 20, "test 2", map[string]string{"retry": "2"})
	assert.Nil(t, err)

	m2, err = pq.Get(m2.ID)
	assert.Nil(t, err)
	assert.Equal(t, "test 2", m2.Content)
	assert.Equal(t, int64(20), m2.Priority)
	assert.Equal(t, "2", m2.Metadata["retry"])

	m3, err := pq.Enqueue(3, "default", 30, "test 3", map[string]string{"retry": "0"})
	assert.Nil(t, err)

	m3, err = pq.Get(m3.ID)
	assert.Nil(t, err)
	assert.Equal(t, "test 3", m3.Content)
	assert.Equal(t, int64(30), m3.Priority)
	assert.Equal(t, "0", m3.Metadata["retry"])

	m4, err := pq.Enqueue(4, "default", 40, "test 4", map[string]string{"retry": "0"})
	assert.Nil(t, err)

	m4, err = pq.Get(m4.ID)
	assert.Nil(t, err)
	assert.Equal(t, "test 4", m4.Content)
	assert.Equal(t, int64(40), m4.Priority)

	m5, err := pq.Enqueue(5, "default", 50, "test 5", map[string]string{"retry": "0"})
	assert.Nil(t, err)

	m5, err = pq.Get(m5.ID)
	assert.Nil(t, err)
	assert.Equal(t, "test 5", m5.Content)
	assert.Equal(t, int64(50), m5.Priority)

	err = pq.UpdatePriority(m1.ID, 60)
	assert.Nil(t, err)

	m1, err = pq.Get(m1.ID)
	assert.Nil(t, err)
	assert.Equal(t, "test 1", m1.Content)
	assert.Equal(t, int64(60), m1.Priority)

	err = pq.UpdatePriority(m4.ID, 55)
	assert.Nil(t, err)

	m4, err = pq.Get(m4.ID)
	assert.Nil(t, err)
	assert.Equal(t, "test 4", m4.Content)
	assert.Equal(t, int64(55), m4.Priority)

	m, err := pq.Dequeue(true)
	assert.Nil(t, err)

	assert.Equal(t, "test 2", m.Content)

	m, err = pq.Dequeue(true)
	assert.Nil(t, err)

	assert.Equal(t, "test 3", m.Content)

	m, err = pq.Dequeue(true)
	assert.Nil(t, err)

	assert.Equal(t, "test 5", m.Content)

	m, err = pq.Dequeue(true)

	assert.Nil(t, err)

	assert.Equal(t, "test 4", m.Content)

	m, err = pq.Dequeue(true)
	assert.Nil(t, err)

	assert.Equal(t, "test 1", m.Content)

	db.Close()
}

func TestQueueDelayedMessage(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "db*")
	defer os.RemoveAll(tmpDir)

	opts := badger.DefaultOptions(tmpDir)
	db, err := badger.Open(opts)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}

	pq := NewQueue(
		db,
		&config.Config{Queue: config.QueueConfig{
			AcknowledgementCheckInterval: 1,
			QueueStats:                   config.QueueStatsConfig{WindowSide: 10},
		}},
		nil,
	)
	pq.Create("delayed", "test_queue_1")

	priority := time.Now().UTC().Add(1 * time.Second).Unix()
	m1, err := pq.Enqueue(1, "default", priority, "delayed message 1", map[string]string{})
	assert.Nil(t, err)

	m1, err = pq.Get(m1.ID)
	assert.Nil(t, err)
	assert.Equal(t, "delayed message 1", m1.Content)
	assert.Equal(t, priority, m1.Priority)

	m1, err = pq.Dequeue(true)
	assert.Nil(t, m1)
	assert.EqualError(t, err, ErrEmptyQueue.Error())

	time.Sleep(1 * time.Second)

	m1, err = pq.Dequeue(true)
	assert.Nil(t, err)
	assert.Equal(t, "delayed message 1", m1.Content)
	assert.Equal(t, priority, m1.Priority)
}

func TestQueueAck(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "db*")
	defer os.RemoveAll(tmpDir)

	opts := badger.DefaultOptions(tmpDir)
	db, err := badger.Open(opts)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}

	pq := NewQueue(
		db,
		&config.Config{Queue: config.QueueConfig{
			AcknowledgementCheckInterval: 1,
			QueueStats:                   config.QueueStatsConfig{WindowSide: 10},
		}},
		nil,
	)
	pq.Create("delayed", "test_queue")

	m1, err := pq.Enqueue(1, "default", 10, "test 1", map[string]string{"retry": "0"})
	assert.Nil(t, err)
	assert.Equal(t, "test 1", m1.Content)
	assert.Equal(t, int64(10), m1.Priority)

	m2, err := pq.Enqueue(2, "default", 20, "test 2", map[string]string{"retry": "0"})
	assert.Nil(t, err)
	assert.Equal(t, "test 2", m2.Content)
	assert.Equal(t, int64(20), m2.Priority)

	m3, err := pq.Enqueue(3, "default", 30, "test 3", map[string]string{"retry": "0"})
	assert.Nil(t, err)
	assert.Equal(t, "test 3", m3.Content)
	assert.Equal(t, int64(30), m3.Priority)

	m4, err := pq.Enqueue(4, "default", 40, "test 4", map[string]string{"retry": "0"})
	assert.Nil(t, err)
	assert.Equal(t, "test 4", m4.Content)
	assert.Equal(t, int64(40), m4.Priority)

	m1, err = pq.Dequeue(false)
	assert.Nil(t, err)
	assert.Equal(t, "test 1", m1.Content)

	m2, err = pq.Dequeue(false)
	assert.Nil(t, err)
	assert.Equal(t, "test 2", m2.Content)

	m3, err = pq.Dequeue(false)
	assert.Nil(t, err)
	assert.Equal(t, "test 3", m3.Content)

	m4, err = pq.Dequeue(false)
	assert.Nil(t, err)
	assert.Equal(t, "test 4", m4.Content)

	err = pq.Ack(m1.ID)
	assert.Nil(t, err)

	err = pq.Ack(m2.ID)
	assert.Nil(t, err)

	err = pq.Ack(m3.ID)
	assert.Nil(t, err)

	err = pq.Ack(m4.ID)
	assert.Nil(t, err)

	err = pq.Ack(100)
	assert.EqualError(t, err, ErrMessageNotFound.Error())

	db.Close()
}

func TestQueueNack(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "db*")
	defer os.RemoveAll(tmpDir)

	opts := badger.DefaultOptions(tmpDir)
	db, err := badger.Open(opts)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}

	pq := NewQueue(
		db,
		&config.Config{Queue: config.QueueConfig{
			AcknowledgementCheckInterval: 1,
			QueueStats:                   config.QueueStatsConfig{WindowSide: 10},
		}},
		nil,
	)
	pq.Create("delayed", "test_queue")

	m1, err := pq.Enqueue(1, "default", 10, "test 1", map[string]string{"retry": "0"})
	assert.Nil(t, err)
	assert.Equal(t, "test 1", m1.Content)
	assert.Equal(t, int64(10), m1.Priority)
	assert.Equal(t, map[string]string{"retry": "0"}, m1.Metadata)

	m2, err := pq.Enqueue(2, "default", 20, "test 2", map[string]string{"retry": "0"})
	assert.Nil(t, err)
	assert.Equal(t, "test 2", m2.Content)
	assert.Equal(t, int64(20), m2.Priority)
	assert.Equal(t, map[string]string{"retry": "0"}, m2.Metadata)

	m3, err := pq.Enqueue(3, "default", 30, "test 3", nil)
	assert.Nil(t, err)
	assert.Equal(t, "test 3", m3.Content)
	assert.Equal(t, int64(30), m3.Priority)
	assert.Equal(t, map[string]string(nil), m3.Metadata)

	m4, err := pq.Enqueue(4, "default", 40, "test 4", nil)
	assert.Nil(t, err)
	assert.Equal(t, "test 4", m4.Content)
	assert.Equal(t, int64(40), m4.Priority)
	assert.Equal(t, map[string]string(nil), m4.Metadata)

	m1, err = pq.Dequeue(false)
	assert.Nil(t, err)
	assert.Equal(t, "test 1", m1.Content)

	m2, err = pq.Dequeue(false)
	assert.Nil(t, err)
	assert.Equal(t, "test 2", m2.Content)

	m3, err = pq.Dequeue(false)
	assert.Nil(t, err)
	assert.Equal(t, "test 3", m3.Content)

	m4, err = pq.Dequeue(false)
	assert.Nil(t, err)
	assert.Equal(t, "test 4", m4.Content)

	err = pq.Nack(m1.ID, 30, map[string]string{"retry": "1"})
	assert.Nil(t, err)

	err = pq.Nack(m2.ID, m2.Priority, map[string]string{"retry": "2"})
	assert.Nil(t, err)

	err = pq.Nack(m3.ID, m3.Priority, nil)
	assert.Nil(t, err)

	err = pq.Nack(m4.ID, m4.Priority, nil)
	assert.Nil(t, err)

	err = pq.Nack(100, 10, nil)
	assert.EqualError(t, err, ErrMessageNotFound.Error())

	m1, err = pq.Dequeue(true)
	assert.Nil(t, err)
	assert.Equal(t, "test 2", m1.Content)
	assert.Equal(t, map[string]string{"retry": "2"}, m1.Metadata)

	m2, err = pq.Dequeue(true)
	assert.Nil(t, err)
	assert.Equal(t, "test 1", m2.Content)
	assert.Equal(t, map[string]string{"retry": "1"}, m2.Metadata)

	m3, err = pq.Dequeue(true)
	assert.Nil(t, err)
	assert.Equal(t, "test 3", m3.Content)
	assert.Equal(t, map[string]string(nil), m3.Metadata)

	m4, err = pq.Dequeue(true)
	assert.Nil(t, err)
	assert.Equal(t, "test 4", m4.Content)
	assert.Equal(t, map[string]string(nil), m4.Metadata)

	m4, err = pq.Dequeue(true)
	assert.EqualError(t, err, ErrEmptyQueue.Error())
}

func BenchmarkQueueEnqueue(b *testing.B) {
	tempFolder, _ := os.MkdirTemp("", "testdir")

	opts := badger.DefaultOptions(tempFolder)
	db, err := badger.Open(opts)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}

	pq := NewQueue(
		db,
		&config.Config{Queue: config.QueueConfig{
			AcknowledgementCheckInterval: 1,
			QueueStats:                   config.QueueStatsConfig{WindowSide: 10},
		}},
		nil,
	)
	pq.Create("delayed", "test_queue")

	// Pre-fill the queue with items to ensure there’s something to dequeue
	for i := 0; i < b.N; i++ {
		pq.Enqueue(uint64(i), "customer1", 10, "content 1", nil)
	}
}

func BenchmarkQueueDequeue(b *testing.B) {
	tempFolder, _ := os.MkdirTemp("", "testdir")
	opts := badger.DefaultOptions(tempFolder)
	db, err := badger.Open(opts)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}

	pq := NewQueue(
		db,
		&config.Config{Queue: config.QueueConfig{
			AcknowledgementCheckInterval: 1,
			QueueStats:                   config.QueueStatsConfig{WindowSide: 10},
		}},
		nil,
	)
	pq.Create("delayed", "test_queue")

	// Pre-fill the queue with items to ensure there’s something to dequeue
	for i := 0; i < b.N; i++ {
		pq.Enqueue(uint64(i), "customer1", 10, "content 1", nil)
	}

	b.ResetTimer() // Reset timer to focus only on Dequeue operation timing

	for i := 0; i < b.N; i++ {
		pq.Dequeue(true)
	}
}
