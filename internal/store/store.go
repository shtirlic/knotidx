package store

import (
	"errors"
	"fmt"

	"github.com/shtirlic/knotidx/internal/config"
)

// Errors related to store operations.
var (
	ErrOpenStore  = errors.New("open store error")
	ErrCloseStore = errors.New("close store error")
)

// Store defines the interface for a key-value store.
type Store interface {
	Open() error                                            // Open the store.
	Close() error                                           // Close the store.
	Reset() error                                           // Reset the store.
	Delete(key string) error                                // Delete a key from the store.
	Find(key string) ItemInfo                               // Find information about a key in the store.
	Info() string                                           // Get information about the store.
	Maintenance()                                           // Perform maintenance tasks on the store.
	Type() DatabaseType                                     // Get the type of the database.
	Keys(prefix string, pattern string, limit int) []string // Get keys based on prefix, pattern, and limit.

	Add(map[string]ItemInfo) error // Add items to the store.
	Items() ([]*ItemInfo, error)   // Get all items from the store. // DEBUG func
}

// BatchCount specifies the batch count for store operations.
const BatchCount int = 100

// DatabaseType represents the type of the database.
type DatabaseType string

// NewStore creates a new store based on the provided configuration.
func NewStore(c config.StoreConfig) (s Store, err error) {
	switch DatabaseType(c.Type) {
	case BadgerDatabaseType:
		// Create either a Disk Badger store or an In-Memory Badger store based on the configuration.
		if c.Path != "" {
			s = NewDiskBadgerStore(c.Path)
		} else {
			s = NewInMemoryBadgerStore()
		}
		err = s.Open()
	default:
		err = fmt.Errorf("database type %s is unknown", c.Type)
	}
	return
}
