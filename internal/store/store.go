package store

import (
	"errors"
	"fmt"
	"log"

	"github.com/dgraph-io/badger/v4"

	"knot/internal/item"
)

var GlobalStore = &Store{
	storePath: "",
	storeDb:   nil,
}

func init() {
	GlobalStore.Open()
}

type Store struct {
	storePath string
	storeDb   *badger.DB
}

func (store *Store) Open() {
	var err error
	store.storeDb, err = badger.Open(badger.DefaultOptions(store.storePath).WithInMemory(true))
	if err != nil {
		log.Fatal(err)
	}
}

func (store *Store) Add(updates map[string]item.ItemInfo) {
	txn := store.storeDb.NewTransaction(true)
	for k, v := range updates {
		if err := txn.Set([]byte(k), v.Encode()); errors.Is(err, badger.ErrTxnTooBig) {
			_ = txn.Commit()
			txn = store.storeDb.NewTransaction(true)
			_ = txn.Set([]byte(k), v.Encode())
		}
	}
	_ = txn.Commit()
}

func (store *Store) List() {
	err := store.storeDb.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			i := it.Item()
			storeItem := item.Item(i)
			fmt.Println(*storeItem)
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
}
