package indexer

import (
	"log/slog"

	"github.com/shtirlic/knotidx/internal/config"
	"github.com/shtirlic/knotidx/internal/store"
)

type Indexer interface {
	Config() *Config
	ModifiedIndex(store.Store)
	NewIndex() error
	NewItemInfo() store.ItemInfo
	Type() IndexerType
}

type IndexerType string

type Config struct {
	Name   string
	Params map[string]string
}

func NewIndexers(c config.IndexerConfig, s store.Store) []Indexer {
	var indexers []Indexer
	for _, path := range c.Paths {
		switch IndexerType(c.Type) {
		case FsIndexerType:
			indexers = append(indexers, NewFileSystemIndexer(s, path, nil, nil))
		default:
			slog.Warn("indexer type is unknown", "type", c.Type)
		}
	}
	return indexers
}
