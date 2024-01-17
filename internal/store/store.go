package store

import (
	"fmt"

	"github.com/shtirlic/knot/internal/config"
)

type Store interface {
	Open() error
	Close() error
	Reset() error
	Info() string
	Type() DatabaseType
	Find(ItemInfo) *ItemInfo
	GetAllKeys() []string

	Add(map[string]ItemInfo) error
	GetAll() ([]*ItemInfo, error)
}

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
