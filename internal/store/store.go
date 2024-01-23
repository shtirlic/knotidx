package store

import (
	"errors"
	"fmt"

	"github.com/shtirlic/knotidx/internal/config"
)

var (
	ErrOpenStore  = errors.New("open store error")
	ErrCloseStore = errors.New("close store error")
)

type Store interface {
	Open() error
	Close() error
	Reset() error
	Delete(key string) error
	Find(key string) *ItemInfo
	Info() string
	Maintenance()
	Type() DatabaseType
	Keys(prefix string) []string

	Add(map[string]ItemInfo) error
	Items() ([]*ItemInfo, error)
}

const (
	BatchCount int = 100
)

type DatabaseType string

func NewStore(c config.StoreConfig) (s Store, err error) {

	switch DatabaseType(c.Type) {
	case BadgerDatabaseType:
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
