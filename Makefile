all:
	go generate -v ./cmd/knotidx
	go build -v ./cmd/knotidx
