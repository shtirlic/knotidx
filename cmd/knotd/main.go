package main

import "github.com/shtirlic/knot/internal/indexer"

func main() {

	indx := indexer.NewIndexer("/home/shtirlic", nil, nil)
	indx.Run()
}
