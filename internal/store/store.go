package store

import (
	"errors"
	"log"

	"github.com/dgraph-io/badger/v4"
)

var GlobalStore = &Store{
	storePath: "",
	storeDb:   nil,
}

func init() {
	err := GlobalStore.Open()
	if err != nil {
		log.Fatal(err)
		return
	}
}

type Store struct {
	storePath string
	storeDb   *badger.DB
}

func (store *Store) Open() (err error) {
	store.storeDb, err = badger.Open(badger.DefaultOptions(store.storePath).WithInMemory(true))
	if err != nil {
		log.Println(err)
	}
	return
}

func (store *Store) Close() (err error) {
	err = store.storeDb.Close()
	if err != nil {
		log.Println(err)
	}
	return
}

func (store *Store) Add(updates map[string]ItemInfo) {
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

func (store *Store) GetAll() (items []*ItemInfo, err error) {
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
