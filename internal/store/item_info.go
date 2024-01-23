package store

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/cespare/xxhash/v2"
)

type ItemType string

type ItemInfo struct {
	Name     string
	Path     string
	Type     ItemType
	MimeType string
	ModTime  time.Time
	Size     int64
	Hash     string
}

func NewItemInfo(name string, path string, modTime time.Time, size int64, t ItemType) (i ItemInfo) {
	i = ItemInfo{Name: name, Path: path, ModTime: modTime, Size: size, Type: t}
	return
}

func (o *ItemInfo) XXhash() string {
	return strconv.FormatUint(xxhash.Sum64String(o.String()), 16)
}

func (o *ItemInfo) String() string {

	return strings.Join([]string{
		o.Path,
		string(o.Type),
		o.MimeType,
		o.ModTime.String(),
		strconv.FormatInt(o.Size, 10),
	}, "")
}

func (o *ItemInfo) KeyName() string {
	return fmt.Sprintf("%s_%s", o.Type, o.Path)
}

func (o *ItemInfo) Encode() []byte {
	var buff bytes.Buffer
	enc := gob.NewEncoder(&buff)
	err := enc.Encode(o)
	if err != nil {
		return nil
	}
	return buff.Bytes()
}

func (o *ItemInfo) Decode(data []byte) {
	var buff bytes.Buffer
	enc := gob.NewDecoder(&buff)
	buff.Write(data)
	err := enc.Decode(o)
	if err != nil {
		return
	}
}
