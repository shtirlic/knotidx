package store

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/dgraph-io/badger/v4"
	"github.com/dgraph-io/badger/v4/options"
)

const (
	BadgerDatabaseType DatabaseType = "badger"
)

type BadgerStore struct {
	storePath string
	db        *badger.DB
	inMemory  bool
	dbType    DatabaseType
}

func NewBadgerStore(storePath string, inMemory bool) *BadgerStore {
	return &BadgerStore{
		storePath: storePath,
		inMemory:  inMemory,
		dbType:    BadgerDatabaseType,
	}
}

func NewDiskBadgerStore(storePath string) *BadgerStore {
	return NewBadgerStore(storePath, false)
}

func NewInMemoryBadgerStore() *BadgerStore {
	return NewBadgerStore("", true)
}

func (s *BadgerStore) Type() DatabaseType {
	return s.dbType
}

func (s *BadgerStore) Info() string {
	return fmt.Sprintf("Badger Store memory:%v path:%v", s.inMemory, s.storePath)
}

func (s *BadgerStore) Open() (err error) {
	if s.db != nil {
		return
	}
	slog.Debug("Opening store", "store", s)

	opts := badger.DefaultOptions(s.storePath)
	opts.NumMemtables = 3
	opts.NumLevelZeroTables = 3
	opts.NumLevelZeroTablesStall = 6
	opts.NumCompactors = 2
	opts.BlockCacheSize = 0
	opts.Compression = options.None
	opts.MemTableSize = 8 << 20
	opts.IndexCacheSize = 16 << 20
	opts.ValueLogFileSize = 256 << 20
	opts.Logger = nil
	s.db, err = badger.Open(opts.WithInMemory(s.inMemory))
	if err != nil {
		slog.Debug("error while opening store", "store", s, "error", err)
	}
	return
}

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

// TODO: Need mutex for read and writes
func (s *BadgerStore) Reset() (err error) {
	err = s.db.DropAll()
	if err != nil {
		slog.Debug("error while reseting store", "store", s, "error", err)
		return
	}
	return
}

func (s *BadgerStore) Find(i ItemInfo) *ItemInfo {
	s.Open()
	var found bool = false
	s.db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		prefix := []byte(i.KeyName())
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			if string(item.Key()) == i.KeyName() {
				found = true
				break
			}
		}
		return nil
	})

	if found {
		return &i
	}
	return nil
}

func (s *BadgerStore) GetAllKeys() []string {
	var keys []string
	s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			keys = append(keys, string(item.Key()))
		}
		return nil
	})
	return keys
}

func (s *BadgerStore) Add(updates map[string]ItemInfo) {
	s.Open()
	txn := s.db.NewTransaction(true)
	defer txn.Discard()
	for k, v := range updates {
		if err := txn.Set([]byte(k), v.Encode()); errors.Is(err, badger.ErrTxnTooBig) {
			_ = txn.Commit()
			txn = s.db.NewTransaction(true)
			_ = txn.Set([]byte(k), v.Encode())
		}
	}
	_ = txn.Commit()
}

func (s *BadgerStore) GetAll() (items []*ItemInfo, err error) {
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
