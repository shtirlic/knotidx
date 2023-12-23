package store

import (
	"errors"
	"fmt"
	"log"

	"github.com/dgraph-io/badger/v4"
)

type BadgerStore struct {
	storePath string
	db        *badger.DB
	inMemory  bool
}

func NewBadgerStore(storePath string, inMemory bool) *BadgerStore {
	return &BadgerStore{
		storePath: storePath,
		inMemory:  inMemory,
	}
}

func NewDiskBadgerStore(storePath string) *BadgerStore {
	return NewBadgerStore(storePath, false)
}

func NewInMemoryBadgerStore() *BadgerStore {
	return NewBadgerStore("", true)
}

func (s *BadgerStore) Info() string {
	return fmt.Sprintf("Badger Store memory:%v path:%v", s.inMemory, s.storePath)
}

func (s *BadgerStore) Open() (err error) {
	if s.db != nil {
		return
	}
	s.db, err = badger.Open(badger.DefaultOptions(s.storePath).WithInMemory(s.inMemory))
	if err != nil {
		log.Println(err)
	}
	return
}

func (s *BadgerStore) Close() (err error) {
	if s.db == nil {
		return
	}
	err = s.db.Close()
	if err != nil {
		log.Println(err)
	}
	return
}

func (s *BadgerStore) Add(updates map[string]ItemInfo) {
	s.Open()
	txn := s.db.NewTransaction(true)
	for k, v := range updates {
		if err := txn.Set([]byte(k), v.Encode()); errors.Is(err, badger.ErrTxnTooBig) {
			_ = txn.Commit()
			txn = s.db.NewTransaction(true)
			_ = txn.Set([]byte(k), v.Encode())
		}
	}
	_ = txn.Commit()
	return
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
		log.Println(err)
	}
	return
}
