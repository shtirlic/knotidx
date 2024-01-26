package indexer

import (
	"log/slog"

	"github.com/shtirlic/knotidx/internal/config"
	"github.com/shtirlic/knotidx/internal/store"
)

type Indexer interface {
	Config() config.IndexerConfig // not used for now
	UpdateIndex() error
	CleanIndex(prefix string) error
	Type() IndexerType
	Watch(quit chan bool)
}

type IndexerType string

func NewIndexers(c config.IndexerConfig, s store.Store) []Indexer {
	var indexers []Indexer
	for _, path := range c.Paths {
		switch IndexerType(c.Type) {
		case FsIndexerType:
			indexers = append(indexers, NewFileSystemIndexer(s, path, c))
		default:
			slog.Warn("indexer type is unknown", "type", c.Type)
		}
	}
	return indexers
}
