package main

import "knot/internal/indexer"

func main() {

	idxr := indexer.NewIndexer("/home/shtirlic/", nil)
	idxr.Run()
}
