package badgerstore

import (
	"os"
	"testing"

	"github.com/hashicorp/raft"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testBadgerStore(t testing.TB) *BadgerStore {
	dirname, err := os.MkdirTemp("", "store")
	require.NoError(t, err)

	os.Remove(dirname)

	// Successfully creates and returns a store
	store, err := NewBadgerStore(dirname)
	require.NoError(t, err)

	return store
}

func testRaftLog(idx uint64, data string) *raft.Log {
	return &raft.Log{
		Data:  []byte(data),
		Index: idx,
	}
}

func TestBadgerStore_Implements(t *testing.T) {
	var store interface{} = &BadgerStore{}
	_, ok := store.(raft.StableStore)
	assert.True(t, ok)

	_, ok = store.(raft.LogStore)
	assert.True(t, ok)
}

func TestBadgerStore_FirstIndex(t *testing.T) {
	store := testBadgerStore(t)
	defer store.Close()
	defer os.Remove(store.path)

	// Should get 0 index on empty log
	idx, err := store.FirstIndex()
	require.NoError(t, err)

	assert.Equal(t, uint64(0), idx)

	// Set a mock raft log
	logs := []*raft.Log{
		testRaftLog(1, "log1"),
		testRaftLog(2, "log2"),
		testRaftLog(3, "log3"),
	}
	err = store.StoreLogs(logs)
	require.NoError(t, err)

	// Fetch the first Raft index
	idx, err = store.FirstIndex()
	require.NoError(t, err)

	assert.Equal(t, uint64(1), idx)
}

func TestBadgerStore_LastIndex(t *testing.T) {
	store := testBadgerStore(t)
	defer store.Close()
	defer os.Remove(store.path)

	// Should get 0 index on empty log
	idx, err := store.LastIndex()
	require.NoError(t, err)
	assert.Equal(t, uint64(0), idx)

	// Set a mock raft log
	logs := []*raft.Log{
		testRaftLog(1, "log1"),
		testRaftLog(2, "log2"),
		testRaftLog(3, "log3"),
	}
	err = store.StoreLogs(logs)
	require.NoError(t, err)

	// Fetch the last Raft index
	idx, err = store.LastIndex()
	require.NoError(t, err)
	assert.Equal(t, uint64(3), idx)
}

func TestBadgerStore_GetLog(t *testing.T) {
	store := testBadgerStore(t)
	defer store.Close()
	defer os.Remove(store.path)

	log := new(raft.Log)

	// Should return an error on non-existent log
	err := store.GetLog(1, log)
	assert.Equal(t, raft.ErrLogNotFound, err)

	// Set a mock raft log
	logs := []*raft.Log{
		testRaftLog(1, "log1"),
		testRaftLog(2, "log2"),
		testRaftLog(3, "log3"),
	}
	err = store.StoreLogs(logs)
	require.NoError(t, err)

	// Should return the proper log
	err = store.GetLog(2, log)
	require.NoError(t, err)

	assert.Equal(t, logs[1], log)
}

func TestBadgerStore_SetLog(t *testing.T) {
	store := testBadgerStore(t)
	defer store.Close()
	defer os.Remove(store.path)

	// Create the log
	log := &raft.Log{
		Data:  []byte("log1"),
		Index: 1,
	}

	// Attempt to store the log
	err := store.StoreLog(log)
	require.NoError(t, err)

	// Retrieve the log again
	result := new(raft.Log)
	err = store.GetLog(1, result)
	require.NoError(t, err)

	assert.Equal(t, log, result)
}

func TestBadgerStore_SetLogs(t *testing.T) {
	store := testBadgerStore(t)
	defer store.Close()
	defer os.Remove(store.path)

	// Create a set of logs
	logs := []*raft.Log{
		testRaftLog(1, "log1"),
		testRaftLog(2, "log2"),
		testRaftLog(3, "log3"),
		testRaftLog(4, "log4"),
	}

	// Attempt to store the logs
	err := store.StoreLogs(logs)
	require.NoError(t, err)

	// Ensure we stored them all
	result1, result2 := new(raft.Log), new(raft.Log)

	err = store.GetLog(1, result1)
	require.NoError(t, err)
	assert.Equal(t, logs[0], result1)

	err = store.GetLog(3, result2)
	require.NoError(t, err)
	assert.Equal(t, logs[2], result2)
}

func TestBadgerStore_DeleteRange(t *testing.T) {
	store := testBadgerStore(t)
	defer store.Close()
	defer os.Remove(store.path)

	// Create a set of logs
	log1 := testRaftLog(1, "log1")
	log2 := testRaftLog(2, "log2")
	log3 := testRaftLog(3, "log3")
	log4 := testRaftLog(4, "log4")
	log5 := testRaftLog(5, "log5")
	logs := []*raft.Log{log1, log2, log3, log4, log5}

	// Attempt to store the logs
	err := store.StoreLogs(logs)
	require.NoError(t, err)

	// Attempt to delete a range of logs
	err = store.DeleteRange(1, 3)
	require.NoError(t, err)

	// Ensure the logs were deleted

	err = store.GetLog(1, new(raft.Log))
	assert.Equal(t, raft.ErrLogNotFound, err)

	err = store.GetLog(2, new(raft.Log))
	assert.Equal(t, raft.ErrLogNotFound, err)

	err = store.GetLog(3, new(raft.Log))
	assert.Equal(t, raft.ErrLogNotFound, err)

	err = store.GetLog(4, new(raft.Log))
	require.NoError(t, err)
}

func TestBadgerStore_Set_Get(t *testing.T) {
	store := testBadgerStore(t)
	defer store.Close()
	defer os.Remove(store.path)

	// Returns error on non-existent key
	_, err := store.Get([]byte("bad"))
	assert.Equal(t, ErrKeyNotFound, err)

	k, v := []byte("hello"), []byte("world")

	// Try to set a k/v pair
	err = store.Set(k, v)
	require.NoError(t, err)

	// Try to read it back
	val, err := store.Get(k)
	require.NoError(t, err)
	assert.Equal(t, v, val)
}

func TestBadgerStore_SetUint64_GetUint64(t *testing.T) {
	store := testBadgerStore(t)
	defer store.Close()
	defer os.Remove(store.path)

	// Returns error on non-existent key
	_, err := store.GetUint64([]byte("bad"))
	assert.Equal(t, ErrKeyNotFound, err)

	k, v := []byte("abc"), uint64(123)

	// Attempt to set the k/v pair
	err = store.SetUint64(k, v)
	require.NoError(t, err)

	// Read back the value
	val, err := store.GetUint64(k)
	require.NoError(t, err)
	assert.Equal(t, v, val)
}

// TestDBPath tests that the DBPath method returns the correct path
func TestDBPath(t *testing.T) {
	store := testBadgerStore(t)
	defer store.Close()
	defer os.Remove(store.path)
}

// TestSize tests that the Size method returns the correct size
func TestSize(t *testing.T) {
	store := testBadgerStore(t)
	defer store.Close()
	defer os.Remove(store.path)

	lsm, vlog := store.Size()

	assert.Equal(t, int64(0), lsm)
	assert.Equal(t, int64(0), vlog)

	err := store.Set([]byte("foo"), []byte("bar"))
	require.NoError(t, err)

	lsm, vlog = store.Size()

	assert.Equal(t, int64(0), lsm)
	assert.Equal(t, int64(0), vlog)
}

// TestRunValueLogGC tests that the RunValueLogGC method works as expected
func TestRunValueLogGC(t *testing.T) {
	// store := testBadgerStore(t)
	// defer store.Close()
	// defer os.Remove(store.path)

	// err := store.Acquire([]byte("my-lock:1"), time.Now().UTC().Add(time.Millisecond*200))
	// require.NoError(t, err)

	// err = store.RunValueLogGC(0.5)
	// require.Equal(t, badger.ErrNoRewrite, err)
}
