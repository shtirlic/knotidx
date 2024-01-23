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

// Maintenance implements Store.
func (s *BadgerStore) Maintenance() {
	s.db.RunValueLogGC(0.7)
}

func NewBadgerStore(storePath string, inMemory bool) Store {
	return &BadgerStore{
		storePath: storePath,
		inMemory:  inMemory,
		dbType:    BadgerDatabaseType,
	}
}

func NewDiskBadgerStore(storePath string) Store {
	return NewBadgerStore(storePath, false)
}

func NewInMemoryBadgerStore() Store {
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
	s.db, err = badger.Open(opts.WithInMemory(s.inMemory))
	if err != nil {
		err = errors.Join(ErrOpenStore, err)
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

func (s *BadgerStore) Delete(key string) (err error) {
	s.Open()
	txn := s.db.NewTransaction(true)
	defer txn.Discard()

	err = txn.Delete([]byte(key))
	if err != nil {
		return
	}

	if err = txn.Commit(); err != nil {
		return
	}

	slog.Debug("Store Delete", "key", key)
	return nil
}

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

func (s *BadgerStore) Keys(prefix string) (keys []string) {
	s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Seek([]byte(prefix)); it.ValidForPrefix([]byte(prefix)); it.Next() {
			item := it.Item()
			keys = append(keys, string(item.Key()))
		}
		return nil
	})
	return
}

func (s *BadgerStore) Add(updates map[string]ItemInfo) (err error) {
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
	err = txn.Commit()
	return
}

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
