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

// ItemType represents the type of an item.
type ItemType string

// ItemInfo represents information about an item in the store.
type ItemInfo struct {
	Name     string    // Name of the item.
	Path     string    // Path to the item.
	Type     ItemType  // Type of the item.
	MimeType string    // MIME type of the item.
	ModTime  time.Time // Modification time of the item.
	Size     int64     // Size of the item.
	Hash     string    // Hash of the item.
}

// NewItemInfo creates a new ItemInfo with the specified attributes.
func NewItemInfo(name string, path string, modTime time.Time, size int64, t ItemType) (i ItemInfo) {
	i = ItemInfo{Name: name, Path: path, ModTime: modTime, Size: size, Type: t}
	return
}

// XXhash calculates and returns the XXhash of the item.
func (o *ItemInfo) XXhash() string {
	return strconv.FormatUint(xxhash.Sum64String(o.String()), 16)
}

// String converts the item information to a string for hashing purposes.
func (o *ItemInfo) String() string {

	return strings.Join([]string{
		o.Path,
		string(o.Type),
		o.MimeType,
		o.ModTime.String(),
		strconv.FormatInt(o.Size, 10),
	}, ":")
}

// KeyName generates a key name for the item.
func (o *ItemInfo) KeyName() string {
	return fmt.Sprintf("%s_%s", o.Type, o.Path)
}

// Encode serializes the item information to a byte slice.
func (o *ItemInfo) Encode() []byte {
	var buff bytes.Buffer
	enc := gob.NewEncoder(&buff)
	err := enc.Encode(o)
	if err != nil {
		return nil
	}
	return buff.Bytes()
}

// Decode deserializes the item information from a byte slice.
func (o *ItemInfo) Decode(data []byte) {
	var buff bytes.Buffer
	enc := gob.NewDecoder(&buff)
	buff.Write(data)
	err := enc.Decode(o)
	if err != nil {
		return
	}
}
