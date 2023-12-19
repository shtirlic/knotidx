package store

import (
	"bytes"
	"encoding/gob"
	"strconv"
	"strings"
	"time"

	"github.com/dgraph-io/badger/v4"
)

type ItemType string

const (
	DIR  ItemType = "dir"
	FILE ItemType = "file"
)

type ItemInfo struct {
	Name    string
	Path    string
	Type    ItemType
	ModTime time.Time
	Size    int64
	Hash    string
}

func (o *ItemInfo) String() string {

	return strings.Join([]string{o.Path, string(o.Type), o.ModTime.String(), strconv.FormatInt(o.Size, 10)}, "")
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
