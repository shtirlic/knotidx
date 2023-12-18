package indexer

import (
	"context"
	"log/slog"
	"time"

	"github.com/shtirlic/knotidx/internal/config"
	"github.com/shtirlic/knotidx/internal/store"
)

// Indexer represents the interface for different types of indexers.
type Indexer interface {
	Config() config.IndexerConfig        // Config returns the configuration of the indexer.
	UpdateIndex() (time.Duration, error) // UpdateIndex updates the index.
	CleanIndex(prefix string) error      // CleanIndex cleans the index based on the provided prefix.
	Type() IndexerType                   // Type returns the type of the indexer.
	Watch()                              // Watch monitors for changes in the index.
	Info() IndexerRuntimeInfo            // Get information about the Indexer runtime stutus.
	Feedback() chan IndexerRuntimeInfo
}

type IndexerRuntimeInfo struct {
	StartTime  time.Time
	FinishTime time.Time
	Duration   time.Duration
	Status     string
}

// IndexerType represents the type of an indexer.
type IndexerType string

// NewIndexers creates a slice of indexers based on the provided configuration and store.
func NewIndexers(ctx context.Context, c config.IndexerConfig, s store.Store) []Indexer {
	var indexers []Indexer
	for _, path := range c.Paths {
		switch IndexerType(c.Type) {
		case FileSystemIndexerType:
			// Create a new FileSystemIndexer for each path.
			indexers = append(indexers, NewFileSystemIndexer(ctx, s, path, c))
		default:
			slog.Warn("indexer type is unknown", "type", c.Type)
		}
	}
	return indexers
}
