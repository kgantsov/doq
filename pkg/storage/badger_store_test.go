package storage

import (
	"io"
	"testing"

	"github.com/dgraph-io/badger/v4"
	"github.com/kgantsov/doq/pkg/entity"
	pb "github.com/kgantsov/doq/pkg/proto"
	"github.com/stretchr/testify/assert"
)

func TestCreateUpdateQueue(t *testing.T) {
	db, err := badger.Open(badger.DefaultOptions("").WithInMemory(true))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	store := NewBadgerStore(db)

	queueName := "new-queue"

	err = store.CreateQueue(
		"fair",
		queueName,
		entity.QueueSettings{
			Strategy:   "WEIGHTED",
			MaxUnacked: 75,
			AckTimeout: 3600,
		},
	)
	assert.NoError(t, err)

	queues, err := store.ListQueues()
	assert.NoError(t, err)
	assert.Len(t, queues, 1)
	assert.Equal(t, queueName, queues[0].Name)
	assert.Equal(t, "fair", queues[0].Type)
	assert.Equal(t, "WEIGHTED", queues[0].Settings.Strategy)
	assert.Equal(t, 75, queues[0].Settings.MaxUnacked)
	assert.Equal(t, uint32(3600), queues[0].Settings.AckTimeout)

	q, err := store.LoadQueue(queueName)
	assert.NoError(t, err)
	assert.Equal(t, queueName, q.Name)
	assert.Equal(t, "fair", q.Type)
	assert.Equal(t, "WEIGHTED", q.Settings.Strategy)
	assert.Equal(t, 75, q.Settings.MaxUnacked)
	assert.Equal(t, uint32(3600), q.Settings.AckTimeout)

	err = store.UpdateQueue(q.Type, q.Name, entity.QueueSettings{
		Strategy:   "WEIGHTED",
		MaxUnacked: 100,
		AckTimeout: 7200,
	})
	assert.NoError(t, err)

	q, err = store.LoadQueue(queueName)
	assert.NoError(t, err)
	assert.Equal(t, queueName, q.Name)
	assert.Equal(t, "fair", q.Type)
	assert.Equal(t, "WEIGHTED", q.Settings.Strategy)
	assert.Equal(t, 100, q.Settings.MaxUnacked)
	assert.Equal(t, uint32(7200), q.Settings.AckTimeout)
}

func TestBadgerStore(t *testing.T) {
	db, err := badger.Open(badger.DefaultOptions("").WithInMemory(true))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	store := NewBadgerStore(db)

	queueName := "test-queue"

	err = store.CreateQueue("delayed", queueName, entity.QueueSettings{})
	assert.NoError(t, err)

	message, err := store.Enqueue(queueName, 534654, "default", 555, "test-message", map[string]string{})
	assert.NoError(t, err)
	assert.Equal(t, uint64(534654), message.ID)
	assert.Equal(t, "test-message", message.Content)
	assert.Equal(t, "default", message.Group)
	assert.Equal(t, int64(555), message.Priority)
	assert.Equal(t, map[string]string{}, message.Metadata)

	message, err = store.Get(queueName, 534654)
	assert.NoError(t, err)
	assert.Equal(t, uint64(534654), message.ID)
	assert.Equal(t, "test-message", message.Content)
	assert.Equal(t, "default", message.Group)
	assert.Equal(t, int64(555), message.Priority)
	assert.Equal(t, map[string]string(nil), message.Metadata)

	err = store.UpdateMessage(queueName, 534654, 1000, "test-message", map[string]string{"retry": "5"})
	assert.NoError(t, err)

	message, err = store.UpdatePriority(queueName, 534654, 1000)
	assert.NoError(t, err)
	assert.Equal(t, uint64(534654), message.ID)
	assert.Equal(t, "test-message", message.Content)
	assert.Equal(t, "default", message.Group)
	assert.Equal(t, int64(1000), message.Priority)
	assert.Equal(t, map[string]string{"retry": "5"}, message.Metadata)

	queues, err := store.ListQueues()
	assert.NoError(t, err)
	assert.Len(t, queues, 1)
	assert.Equal(t, queueName, queues[0].Name)
	assert.Equal(t, "delayed", queues[0].Type)

	messages, err := store.GetMessages(queueName, 100, 0)
	assert.NoError(t, err)
	assert.Len(t, messages, 1)
	assert.Equal(t, uint64(534654), messages[0].ID)
	assert.Equal(t, "test-message", messages[0].Content)
	assert.Equal(t, "default", messages[0].Group)
	assert.Equal(t, int64(1000), messages[0].Priority)
	assert.Equal(t, map[string]string{"retry": "5"}, messages[0].Metadata)

	message, err = store.Dequeue(queueName, 534654, false)
	assert.NoError(t, err)
	assert.Equal(t, uint64(534654), message.ID)
	assert.Equal(t, "test-message", message.Content)
	assert.Equal(t, "default", message.Group)
	assert.Equal(t, int64(1000), message.Priority)
	assert.Equal(t, map[string]string{"retry": "5"}, message.Metadata)

	err = store.Ack(queueName, 534654)
	assert.NoError(t, err)

	err = store.DeleteQueue(queueName)
	assert.NoError(t, err)
}

func TestBadgerStoreDeleteMessage(t *testing.T) {
	db, err := badger.Open(badger.DefaultOptions("").WithInMemory(true))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	store := NewBadgerStore(db)

	queueName := "test-queue"

	err = store.CreateQueue("delayed", queueName, entity.QueueSettings{})
	assert.NoError(t, err)

	message, err := store.Enqueue(queueName, 534654, "default", 555, "test-message", map[string]string{})
	assert.NoError(t, err)
	assert.Equal(t, message.ID, uint64(534654))
	assert.Equal(t, message.Content, "test-message")
	assert.Equal(t, message.Group, "default")
	assert.Equal(t, message.Priority, int64(555))

	m, err := store.Delete(queueName, 534654)
	assert.NoError(t, err)
	assert.Equal(t, m.ID, uint64(534654))
	assert.Equal(t, m.Content, "test-message")
	assert.Equal(t, m.Group, "default")
	assert.Equal(t, m.Priority, int64(555))

	message, err = store.Get(queueName, 534654)
	assert.Error(t, err)
	assert.Nil(t, message)

	message, err = store.Dequeue(queueName, 534654, false)
	assert.Error(t, err)
	assert.Nil(t, message)

	err = store.DeleteQueue(queueName)
	assert.NoError(t, err)
}

type MockSink struct {
	data      []byte
	readIndex int64
}

func (ms *MockSink) ID() string {
	return "sink-id"
}

func (ms *MockSink) Cancel() error {
	return nil
}

func (ms *MockSink) Write(p []byte) (n int, err error) {
	ms.data = append(ms.data, p...)
	return len(p), nil
}

func (ms *MockSink) Read(p []byte) (n int, err error) {
	if ms.readIndex >= int64(len(ms.data)) {
		err = io.EOF
		return
	}

	n = copy(p, ms.data[ms.readIndex:])
	ms.readIndex += int64(n)
	return
}

func (ms *MockSink) Close() error {
	return nil
}

func TestBadgerStorePersistSnapshot(t *testing.T) {
	db, err := badger.Open(badger.DefaultOptions("").WithInMemory(true))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	store := NewBadgerStore(db)

	queues := []struct {
		Queue    *entity.QueueConfig
		Messages []struct {
			Id       uint64
			Group    string
			Priority int64
			Content  string
			Metadata map[string]string
		}
	}{
		{
			Queue: &entity.QueueConfig{
				Name:     "transcode-queue",
				Type:     "delayed",
				Settings: entity.QueueSettings{},
			},
			Messages: []struct {
				Id       uint64
				Group    string
				Priority int64
				Content  string
				Metadata map[string]string
			}{
				{
					Id:       534654,
					Group:    "default",
					Priority: 555,
					Content:  "test-message-0",
					Metadata: map[string]string{},
				},
				{
					Id:       534655,
					Group:    "default",
					Priority: 555,
					Content:  "test-message-1",
					Metadata: map[string]string{},
				},
				{
					Id:       534656,
					Group:    "default",
					Priority: 555,
					Content:  "test-message-2",
					Metadata: map[string]string{},
				},
			},
		},
		{
			Queue: &entity.QueueConfig{
				Name: "transfers-queue",
				Type: "fair",
				Settings: entity.QueueSettings{
					Strategy:   "WEIGHTED",
					MaxUnacked: 75,
					AckTimeout: 300,
				},
			},
			Messages: []struct {
				Id       uint64
				Group    string
				Priority int64
				Content  string
				Metadata map[string]string
			}{
				{
					Id:       24,
					Group:    "customer-1",
					Priority: 10,
					Content:  "Message #1",
					Metadata: map[string]string{},
				},
				{
					Id:       25,
					Group:    "customer-1",
					Priority: 10,
					Content:  "Message #2",
					Metadata: map[string]string{},
				},
				{
					Id:       26,
					Group:    "customer-2",
					Priority: 10,
					Content:  "Message #3",
					Metadata: map[string]string{},
				},
				{
					Id:       27,
					Group:    "customer-3",
					Priority: 10,
					Content:  "Message #4",
					Metadata: map[string]string{},
				},
			},
		},
		{
			Queue: &entity.QueueConfig{
				Name: "empty-queue",
				Type: "fair",
				Settings: entity.QueueSettings{
					Strategy:   "ROUND_ROBIN",
					MaxUnacked: 0,
					AckTimeout: 600,
				},
			},
			Messages: []struct {
				Id       uint64
				Group    string
				Priority int64
				Content  string
				Metadata map[string]string
			}{},
		},
	}

	// Create queues and enqueue messages
	for _, q := range queues {
		err = store.CreateQueue(q.Queue.Type, q.Queue.Name, q.Queue.Settings)
		assert.NoError(t, err)

		for _, m := range q.Messages {
			_, err = store.Enqueue(q.Queue.Name, m.Id, m.Group, m.Priority, m.Content, m.Metadata)
			assert.NoError(t, err)
		}
	}

	sink := &MockSink{}

	txn := db.NewTransaction(false)

	// Persist snapshot for each queue
	// This simulates the process of saving the state of each queue and its messages
	// to a snapshot sink, which could be used for recovery or replication.
	// In a real-world scenario, this would be part of a Raft snapshot process.
	for _, q := range queues {
		err = store.PersistSnapshot(q.Queue, sink, txn)
		assert.NoError(t, err)
	}

	// Read the snapshot data from the sink
	for _, q := range queues {
		// Read the queue item from the snapshot
		line, err := ReadSnapshotQueueItem(sink)
		assert.NoError(t, err)
		assert.NotNil(t, line)

		assert.Equal(t, q.Queue.Name, line.Name)
		assert.Equal(t, q.Queue.Type, line.Type)
		assert.Equal(
			t,
			pb.QueueSettings_Strategy(pb.QueueSettings_Strategy_value[q.Queue.Settings.Strategy]),
			line.Settings.Strategy,
		)
		assert.Equal(t, uint32(q.Queue.Settings.MaxUnacked), line.Settings.MaxUnacked)
		assert.Equal(t, uint32(q.Queue.Settings.AckTimeout), line.Settings.AckTimeout)

		// Read each message for the queue
		for _, m := range q.Messages {
			msg, err := ReadSnapshotMessageItem(sink)
			assert.NoError(t, err)
			assert.NotNil(t, msg)

			assert.Equal(t, m.Id, msg.Id)
			assert.Equal(t, m.Content, msg.Content)
			assert.Equal(t, m.Group, msg.Group)
			assert.Equal(t, m.Priority, msg.Priority)
			// assert.Equal(t, m.Metadata, msg.Metadata)
		}
	}
}
