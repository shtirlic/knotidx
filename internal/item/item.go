package item

import (
	"bytes"
	"encoding/gob"
	"time"

	"github.com/dgraph-io/badger/v4"
)

type Type string

const (
	DIR  Type = "dir"
	FILE Type = "file"
)

type ItemInfo struct {
	Hash    string
	Name    string
	Path    string
	ModTime time.Time
	Size    int64
	Type    Type
}

func (o *ItemInfo) Encode() []byte {
	var buff bytes.Buffer
	enc := gob.NewEncoder(&buff)
	err := enc.Encode(o)
	if err != nil {
		return nil
	}
	// fmt.Println(buff.Len())
	return buff.Bytes()
}

func (o *ItemInfo) Decode(data []byte) {
	// fmt.Println(len(data))
	var buff bytes.Buffer
	enc := gob.NewDecoder(&buff)
	buff.Write(data)
	// fmt.Println(buff.Len())
	err := enc.Decode(o)
	if err != nil {
		return
	}
	// fmt.Println(buff.Len())

}

func Item(item *badger.Item) *ItemInfo {
	obj := &ItemInfo{}
	err := item.Value(func(v []byte) error {
		obj.Decode(v)
		return nil
	})
	if err != nil {
		return &ItemInfo{}
	}
	return obj
}
