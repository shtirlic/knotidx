package indexer

import "github.com/shtirlic/knot/internal/store"

type Indexer interface {
	Config() *Config
	ModifiedIndex(store.Store)
	NewIndex(store.Store)
	NewItemInfo() store.ItemInfo
}

type Config struct {
	Name   string
	Params map[string]string
}

func NewIndexer(path string) Indexer {
	return NewFileSystemIndexer(path, nil, nil)
}
