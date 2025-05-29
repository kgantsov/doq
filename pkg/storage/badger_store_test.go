package storage

import (
	"bufio"
	"io"
	"testing"

	"github.com/dgraph-io/badger/v4"
	"github.com/kgantsov/doq/pkg/entity"
	"github.com/stretchr/testify/assert"
)

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
	assert.Equal(t, message.ID, uint64(534654))
	assert.Equal(t, message.Content, "test-message")
	assert.Equal(t, message.Group, "default")
	assert.Equal(t, message.Priority, int64(555))
	assert.Equal(t, message.Metadata, map[string]string{})

	message, err = store.Get(queueName, 534654)
	assert.NoError(t, err)
	assert.Equal(t, message.ID, uint64(534654))
	assert.Equal(t, message.Content, "test-message")
	assert.Equal(t, message.Group, "default")
	assert.Equal(t, message.Priority, int64(555))
	assert.Equal(t, message.Metadata, map[string]string{})

	err = store.UpdateMessage(queueName, 534654, 1000, "test-message", map[string]string{"retry": "5"})
	assert.NoError(t, err)

	message, err = store.UpdatePriority(queueName, 534654, 1000)
	assert.NoError(t, err)
	assert.Equal(t, message.ID, uint64(534654))
	assert.Equal(t, message.Content, "test-message")
	assert.Equal(t, message.Group, "default")
	assert.Equal(t, message.Priority, int64(1000))
	assert.Equal(t, message.Metadata, map[string]string{"retry": "5"})

	message, err = store.Dequeue(queueName, 534654, false)
	assert.NoError(t, err)
	assert.Equal(t, message.ID, uint64(534654))
	assert.Equal(t, message.Content, "test-message")
	assert.Equal(t, message.Group, "default")
	assert.Equal(t, message.Priority, int64(1000))
	assert.Equal(t, message.Metadata, map[string]string{"retry": "5"})

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

	queueName := "test-queue"

	err = store.CreateQueue("delayed", queueName, entity.QueueSettings{})
	assert.NoError(t, err)

	message, err := store.Enqueue(queueName, 534654, "default", 555, "test-message", map[string]string{})
	assert.NoError(t, err)
	assert.Equal(t, message.ID, uint64(534654))
	assert.Equal(t, message.Content, "test-message")
	assert.Equal(t, message.Group, "default")
	assert.Equal(t, message.Priority, int64(555))

	sink := &MockSink{}

	err = store.PersistSnapshot("delayed", queueName, sink)
	assert.NoError(t, err)

	scanner := bufio.NewScanner(sink)

	scanner.Scan()
	line := scanner.Bytes()

	message, err = entity.MessageFromBytes(line)

	assert.NoError(t, err)
	assert.Equal(t, message.ID, uint64(534654))
	assert.Equal(t, message.Content, "test-message")
	assert.Equal(t, message.Group, "default")
	assert.Equal(t, message.Priority, int64(555))
}
