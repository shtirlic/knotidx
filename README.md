# knotidx Fast object indexer

__knotidx__ is a fast object indexer daemon with GRPC backend and cli client.

> [!NOTE]
> Please keep in mind that project is under active development.

## Getting started

### Prerequisites

knotidx requires [Go](https://go.dev/) version [1.22](https://go.dev/doc/devel/release#go1.22.0) or above.

### Installing

Use `go install`:

```sh
go install github.com/shtirlic/knotidx@latest
```

Arch AUR package and homebrew coming soon.

### Building and Usage

Use `go build`:

```sh

# clone
git clone https://github.com/shtirlic/knotidx
cd knotidx

# build adn run daemon with sample config
go build ./cmd/knotidx && ./knotidx --daemon -config configs/knotidx.sample.toml

#grpc client

# interactive input
./knotidx --client

# pipe input and json output
echo "some file" | ./knotidx --client --json

#with jq
echo "some file" | ./knotidx --client --json| jq '.[]."key"'
```

### Example config file knotidx.toml

```toml
interval = 5 # default 5

[grpc]
server = true # default false
# type = "tcp"  # default unix
# path ="knotidx.sock"  # default XDG_RUNTIME_DIR/knotidx.sock
# host = "localhost" # default
# port = 5319   # default 5319

[store]
type = "badger" # default "badger"
# path = "store.knot" for disk storage, default in memory

[[indexer]]
type = "fs"
notify = true
paths = ["/tmp"]
```


## Supported Stores and Indexers

Badger https://github.com/dgraph-io/badger (disk and in-memory modes)
File System indexer with FSNotify watcher


## knotidx Architecture

### Daemon

### Indexer

### Store

### GRPC
