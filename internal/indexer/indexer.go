package indexer

import "github.com/shtirlic/knot/internal/store"

type Indexer interface {
	Run(store.Store)
	Config() *Config
	ModifiedIndex(store.Store)
	NewIndex(store.Store)
}

type Config struct {
	Name   string
	Params map[string]string
}
