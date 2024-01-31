package indexer

import (
	"log/slog"

	"github.com/shtirlic/knotidx/internal/config"
	"github.com/shtirlic/knotidx/internal/store"
)

// Indexer represents the interface for different types of indexers.
type Indexer interface {
	Config() config.IndexerConfig   // Config returns the configuration of the indexer.
	UpdateIndex() error             // UpdateIndex updates the index.
	CleanIndex(prefix string) error // CleanIndex cleans the index based on the provided prefix.
	Type() IndexerType              // Type returns the type of the indexer.
	Watch(quit chan bool)           // Watch monitors for changes in the index.
}

// IndexerType represents the type of an indexer.
type IndexerType string

// NewIndexers creates a slice of indexers based on the provided configuration and store.
func NewIndexers(c config.IndexerConfig, s store.Store) []Indexer {
	var indexers []Indexer
	for _, path := range c.Paths {
		switch IndexerType(c.Type) {
		case FileSystemIndexerType:
			// Create a new FileSystemIndexer for each path.
			indexers = append(indexers, NewFileSystemIndexer(s, path, c))
		default:
			slog.Warn("indexer type is unknown", "type", c.Type)
		}
	}
	return indexers
}
