package store

import (
	"errors"
	"fmt"
	"log/slog"
	"math"
	"strings"

	"github.com/dgraph-io/badger/v4"
	"github.com/dgraph-io/badger/v4/options"
)

// BadgerDatabaseType represents the Badger database type.
const BadgerDatabaseType DatabaseType = "badger"

// BadgerStore is an implementation of the Store interface using the Badger database.
type BadgerStore struct {
	storePath string
	db        *badger.DB
	inMemory  bool
}

// Maintenance performs maintenance tasks on the Badger store, such as value log garbage collection.
func (s *BadgerStore) Maintenance() {
	s.db.RunValueLogGC(0.7)
}

// NewBadgerStore creates a new BadgerStore instance based on the provided parameters.
func NewBadgerStore(storePath string, inMemory bool) Store {
	return &BadgerStore{
		storePath: storePath,
		inMemory:  inMemory,
	}
}

// NewDiskBadgerStore creates a new BadgerStore instance for disk-based storage.
func NewDiskBadgerStore(storePath string) Store {
	return NewBadgerStore(storePath, false)
}

// NewInMemoryBadgerStore creates a new BadgerStore instance for in-memory storage.
func NewInMemoryBadgerStore() Store {
	return NewBadgerStore("", true)
}

// Type returns the type of the database (Badger in this case).
func (s *BadgerStore) Type() DatabaseType {
	return BadgerDatabaseType
}

// Info returns information about the Badger store, including whether it's in-memory and the store path.
func (s *BadgerStore) Info() string {
	return fmt.Sprintf("Badger Store memory:%v path:%v", s.inMemory, s.storePath)
}

// Open opens the Badger store.
func (s *BadgerStore) Open() (err error) {
	if s.db != nil {
		return
	}
	slog.Debug("Opening store", "store", s)

	// Configure Badger options.
	opts := badger.DefaultOptions(s.storePath)
	opts.NumMemtables = 2
	opts.NumLevelZeroTables = 2
	opts.NumLevelZeroTablesStall = 4
	opts.NumCompactors = 2
	opts.BlockCacheSize = 0
	opts.Compression = options.None
	opts.MemTableSize = 8 << 20
	opts.IndexCacheSize = 16 << 20
	opts.ValueLogFileSize = 256 << 20
	opts.Logger = nil

	// Open the Badger database.
	s.db, err = badger.Open(opts.WithInMemory(s.inMemory))
	if err != nil {
		err = errors.Join(ErrOpenStore, err)
		slog.Debug("error while opening store", "store", s, "error", err)
	}
	return
}

// Close closes the Badger store.
func (s *BadgerStore) Close() (err error) {

	if s.db == nil {
		return
	}
	slog.Debug("Closing store", "store", s)

	err = s.db.Close()
	if err != nil {
		slog.Debug("error while closing store", "store", s, "error", err)
	}
	return
}

// Reset resets the Badger store, dropping all data.
// TODO: Need mutex for read and writes
func (s *BadgerStore) Reset() (err error) {

	err = s.db.DropAll()
	if err != nil {
		slog.Debug("error while reseting store", "store", s, "error", err)
		return
	}
	return
}

// Delete deletes an item from the Badger store based on the key.
func (s *BadgerStore) Delete(key string) (err error) {
	s.Open()
	txn := s.db.NewTransaction(true)
	defer txn.Discard()

	// Delete the item using the transaction.
	err = txn.Delete([]byte(key))
	if err != nil {
		return
	}

	// Commit the transaction.
	if err = txn.Commit(); err != nil {
		return
	}

	slog.Debug("Store Delete", "key", key)
	return nil
}

// Find retrieves information about an item from the Badger store based on the key.
func (s *BadgerStore) Find(key string) (item *ItemInfo) {
	s.Open()
	s.db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		prefix := []byte(key)
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			ib := it.Item()
			if string(ib.Key()) == key {
				item = Item(ib)
				break
			}
		}
		return nil
	})
	return
}

// Keys retrieves keys from the Badger store based on the prefix, pattern, and limit.
func (s *BadgerStore) Keys(prefix string, pattern string, limit int) (keys []string) {
	if limit == 0 {
		limit = math.MaxInt
	}
	s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Seek([]byte(prefix)); it.ValidForPrefix([]byte(prefix)) && limit > 0; it.Next() {
			item := it.Item()
			if strings.Contains(string(item.Key()), pattern) {
				keys = append(keys, string(item.Key()))
				limit--
			}
		}
		return nil
	})
	return
}

// Add adds or updates items in the Badger store.
// If the transaction becomes too big, it is committed, and a new transaction is started.
func (s *BadgerStore) Add(updates map[string]ItemInfo) (err error) {
	s.Open()
	txn := s.db.NewTransaction(true)
	defer txn.Discard()
	for k, v := range updates {
		// Set the key-value pair in the transaction.
		if err := txn.Set([]byte(k), v.Encode()); errors.Is(err, badger.ErrTxnTooBig) {
			// If the transaction becomes too big, commit it and start a new transaction.
			_ = txn.Commit()
			txn = s.db.NewTransaction(true)
			_ = txn.Set([]byte(k), v.Encode())
		}
	}
	err = txn.Commit()
	return
}

// Items retrieves all items from the Badger store.
func (s *BadgerStore) Items() (items []*ItemInfo, err error) {
	s.Open()
	err = s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			i := it.Item()
			storeItem := Item(i)
			items = append(items, storeItem)
		}
		return nil
	})
	if err != nil {
		slog.Error("GetAll", "error", err)
	}
	return
}

// Item returns an ItemInfo from a Badger item.
func Item(item *badger.Item) *ItemInfo {
	obj := &ItemInfo{}
	err := item.Value(func(v []byte) error {
		obj.Decode(v)
		return nil
	})
	if err != nil {
		return &ItemInfo{}
	}
	return obj
}
