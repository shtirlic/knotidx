package store

type Store interface {
	Open() error
	Close() error
	Add(map[string]ItemInfo)
	GetAll() ([]*ItemInfo, error)
}
