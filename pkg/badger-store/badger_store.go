package badgerstore

import (
	"errors"

	"github.com/dgraph-io/badger/v4"
	"github.com/hashicorp/raft"
	"github.com/rs/zerolog/log"
)

const (
	// Permissions to use on the db file. This is only used if the
	// database file does not exist and needs to be created.
	dbFileMode = 0600
)

var (
	// Bucket names we perform transactions in
	dbLogs = []byte("logs")
	dbConf = []byte("conf")

	// An error indicating a given key does not exist
	ErrKeyNotFound = errors.New("not found")
)

type Store interface {
	Close() error
	FirstIndex() (uint64, error)
	LastIndex() (uint64, error)
	GetLog(idx uint64, log *raft.Log) error
	StoreLog(log *raft.Log) error
	StoreLogs(logs []*raft.Log) error
	DeleteRange(min, max uint64) error
	Set(k, v []byte) error
	Get(k []byte) ([]byte, error)
	SetUint64(key []byte, val uint64) error
	GetUint64(key []byte) (uint64, error)
	RunValueLogGC(discardRatio float64) error
	Size() (lsm, vlog int64)
}

// BadgerStore provides access to Badger for Raft to store and retrieve
// log entries. It also provides key/value storage, and can be used as
// a LogStore and StableStore.
type BadgerStore struct {
	// db is the underlying handle to the db.
	db *badger.DB

	// The path to the Badger database file
	path string

	msgpackUseNewTimeFormat bool
}

// Options contains all the configuration used to open the Badger
type Options struct {
	// NoSync causes the database to skip fsync calls after each
	// write to the log. This is unsafe, so it should be used
	// with caution.
	NoSync bool

	// MsgpackUseNewTimeFormat when set to true, force the underlying msgpack
	// codec to use the new format of time.Time when encoding (used in
	// go-msgpack v1.1.5 by default). Decoding is not affected, as all
	// go-msgpack v2.1.0+ decoders know how to decode both formats.
	MsgpackUseNewTimeFormat bool
}

// NewBadgerStore takes a file path and returns a connected Raft backend.
func NewBadgerStore(path string) (*BadgerStore, error) {
	db, err := badger.Open(badger.DefaultOptions(path))
	if err != nil {
		return nil, err
	}

	return New(db, Options{})
}

func addPrefix(prefix []byte, key []byte) []byte {
	return append(prefix, key...)
}

// New uses the supplied options to open the Badger and prepare it for use as a raft backend.
func New(db *badger.DB, options Options) (*BadgerStore, error) {
	// Try to connect

	// Create the new store
	store := &BadgerStore{
		db:                      db,
		msgpackUseNewTimeFormat: options.MsgpackUseNewTimeFormat,
	}
	return store, nil
}

// initialize is used to set up all of the buckets.
func (b *BadgerStore) initialize() error {
	return nil
}

// Close is used to gracefully close the DB connection.
func (b *BadgerStore) Close() error {
	return b.db.Close()
}

// FirstIndex returns the first known index from the Raft log.
func (b *BadgerStore) FirstIndex() (uint64, error) {
	txn := b.db.NewTransaction(false)
	defer txn.Discard()

	opts := badger.DefaultIteratorOptions
	opts.PrefetchSize = 10
	opts.PrefetchValues = false

	it := txn.NewIterator(opts)
	defer it.Close()
	prefix := []byte(dbLogs)

	for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
		item := it.Item()
		key := item.Key()

		return bytesToUint64(key[len(dbLogs):]), nil
	}
	return 0, nil
}

// Copy the prefix into a new slice that is one larger than
// the prefix and add an `0xFF` byte to it so
func End(prefix []byte) []byte {
	end := make([]byte, len(prefix)+1)
	copy(end, prefix)

	end[len(end)-1] = 0xFF
	return end
}

// LastIndex returns the last known index from the Raft log.
func (b *BadgerStore) LastIndex() (uint64, error) {
	txn := b.db.NewTransaction(false)
	defer txn.Discard()

	opts := badger.DefaultIteratorOptions
	opts.PrefetchSize = 10
	opts.PrefetchValues = false
	opts.Reverse = true

	it := txn.NewIterator(opts)
	defer it.Close()
	prefix := []byte(dbLogs)

	it.Rewind()

	for it.Seek(End(prefix)); it.ValidForPrefix(prefix); it.Next() {

		item := it.Item()
		key := item.Key()

		return bytesToUint64(key[len(dbLogs):]), nil
	}
	return 0, nil
}

// GetLog is used to retrieve a log from badger at a given index.
func (b *BadgerStore) GetLog(idx uint64, raftLog *raft.Log) error {
	txn := b.db.NewTransaction(false)
	defer txn.Discard()

	item, err := txn.Get(addPrefix(dbLogs, uint64ToBytes(idx)))
	if err != nil {
		return raft.ErrLogNotFound
	}

	val, err := item.ValueCopy(nil)

	if val == nil || err != nil {
		return raft.ErrLogNotFound
	}
	return DecodeMsgPack(val, raftLog)
}

// StoreLog is used to store a single raft log
func (b *BadgerStore) StoreLog(log *raft.Log) error {
	return b.StoreLogs([]*raft.Log{log})
}

// StoreLogs is used to store a set of raft logs
func (b *BadgerStore) StoreLogs(logs []*raft.Log) error {
	log.Debug().Msgf("Storing logs: %+v", logs)

	txn := b.db.NewTransaction(true)
	defer txn.Discard()

	for _, log := range logs {
		key := uint64ToBytes(log.Index)
		val, err := EncodeMsgPack(log, b.msgpackUseNewTimeFormat)
		if err != nil {
			return err
		}

		if err := txn.Set(addPrefix(dbLogs, key), val.Bytes()); err != nil {
			return err
		}
	}

	return txn.Commit()
}

// DeleteRange is used to delete logs within a given range inclusively.
func (b *BadgerStore) DeleteRange(min, max uint64) error {
	batchSize := 100 // Adjust the batch size as needed

	opts := badger.DefaultIteratorOptions
	opts.PrefetchSize = 10

	// Convert min to the prefixed byte array
	minKey := addPrefix(dbLogs, uint64ToBytes(min))

	for {
		txn := b.db.NewTransaction(true)
		it := txn.NewIterator(opts)

		count := 0
		var lastKey []byte

		for it.Seek(minKey); it.ValidForPrefix(dbLogs); it.Next() {
			item := it.Item()
			k := item.Key()
			lastKey = append([]byte{}, k...)

			if bytesToUint64(k[len(dbLogs):]) > max {
				break
			}

			if err := txn.Delete(k); err != nil {
				it.Close()
				txn.Discard()
				return err
			}

			count++
			if count >= batchSize {
				break
			}
		}

		it.Close()

		if count == 0 {
			// No more items to delete
			txn.Discard()
			break
		}

		// Commit the current transaction
		if err := txn.Commit(); err != nil {
			return err
		}

		// Set the minKey for the next batch to be the lastKey + 1
		minKey = append(lastKey, 0)
	}

	return nil
}

// Set is used to set a key/value set outside of the raft log
func (b *BadgerStore) Set(k, v []byte) error {
	txn := b.db.NewTransaction(true)
	defer txn.Discard()

	if err := txn.Set(addPrefix(dbConf, k), v); err != nil {
		return err
	}

	return txn.Commit()
}

// Get is used to retrieve a value from the k/v store by key
func (b *BadgerStore) Get(k []byte) ([]byte, error) {
	txn := b.db.NewTransaction(false)
	defer txn.Discard()

	item, err := txn.Get(addPrefix(dbConf, k))
	if err != nil {
		return nil, ErrKeyNotFound
	}

	val, err := item.ValueCopy(nil)

	if val == nil || err != nil {
		return nil, ErrKeyNotFound
	}
	return append([]byte(nil), val...), nil
}

// SetUint64 is like Set, but handles uint64 values
func (b *BadgerStore) SetUint64(key []byte, val uint64) error {
	return b.Set(key, uint64ToBytes(val))
}

// GetUint64 is like Get, but handles uint64 values
func (b *BadgerStore) GetUint64(key []byte) (uint64, error) {
	val, err := b.Get(key)
	if err != nil {
		return 0, err
	}
	return bytesToUint64(val), nil
}

func (b *BadgerStore) RunValueLogGC(discardRatio float64) error {
	return b.db.RunValueLogGC(discardRatio)
}

func (b *BadgerStore) Size() (lsm, vlog int64) {
	return b.db.Size()
}
