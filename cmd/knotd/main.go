package main

import (
	"github.com/shtirlic/knot/internal/indexer"
	"github.com/shtirlic/knot/internal/store"
)

func main() {
	var indx indexer.Indexer

	indx = indexer.NewFsIndexer("/home/shtirlic", nil, nil)
	indx.Run(store.NewInMemoryBadgerStore())
}
