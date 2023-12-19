package store

import (
	"errors"
	"log"

	"github.com/dgraph-io/badger/v4"
)

type BadgerStore struct {
	storePath string
	storeDb   *badger.DB
	inMemory  bool
}

func NewBadgerStore(storePath string, inMemory bool) *BadgerStore {
	return &BadgerStore{
		storePath: storePath,
		inMemory:  inMemory,
	}
}

func NewInMemoryBadgerStore() *BadgerStore {
	return NewBadgerStore("", true)
}

func (store *BadgerStore) Open() (err error) {
	if store.storeDb != nil {
		return
	}
	store.storeDb, err = badger.Open(badger.DefaultOptions(store.storePath).WithInMemory(true))
	if err != nil {
		log.Println(err)
	}
	return
}

func (store *BadgerStore) Close() (err error) {
	if store.storeDb == nil {
		return
	}
	err = store.storeDb.Close()
	if err != nil {
		log.Println(err)
	}
	return
}

func (store *BadgerStore) Add(updates map[string]ItemInfo) {
	store.Open()
	txn := store.storeDb.NewTransaction(true)
	for k, v := range updates {
		if err := txn.Set([]byte(k), v.Encode()); errors.Is(err, badger.ErrTxnTooBig) {
			_ = txn.Commit()
			txn = store.storeDb.NewTransaction(true)
			_ = txn.Set([]byte(k), v.Encode())
		}
	}
	_ = txn.Commit()
	return
}

func (store *BadgerStore) GetAll() (items []*ItemInfo, err error) {
	store.Open()
	err = store.storeDb.View(func(txn *badger.Txn) error {
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
