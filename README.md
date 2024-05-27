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
go install github.com/shtirlic/knotidx/cmd/knotidx@latest
```

Use `yay` for Arch AUR package:

```sh
yay knotidx

```

### Building

Use `go build`:

```sh

# clone the repo
git clone https://github.com/shtirlic/knotidx
cd knotidx

# build the binary (same for daemon and cli)
go build ./cmd/knotidx
```

### Usage

```sh

 #run daemon with sample config
./knotidx --daemon --config configs/knotidx.sample.toml

#grpc client

# Run the CLI in interactive mode.
./knotidx --client

# Pipe input data and json output
echo "some file" | ./knotidx --client --json

# Use jq to process output (e.g., retrieve keys):
echo "some file" | ./knotidx --client --json | jq '.[]."key"'
```

### Example config file `knotidx.toml`

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

## Features

- [x] Fast file indexing and key search
- [x] Disk or in-memory storage for faster search
- [x] GRPC protocol server
- [x] System Idle detection for background indexing
- [x] [FS] fsnotify watchers
- [x] Hot reload on SIGHUP or via GRPC
- [ ] [FS] xattr attributes support https://en.wikipedia.org/wiki/Extended_file_attributes
- [ ] Git Indexer
- [ ] sysfs Indexer
- [ ] S3 Indexer
- [ ] Metainfo extraction (e-books, images, audio, video)
- [ ] D-BUS interface
- [ ] KDE Baloo drop-in replacement
- [ ] Events and callbacks
- [ ] Testing

## Supported Stores and Indexers

- Badger https://github.com/dgraph-io/badger (disk and in-memory modes)
- File System indexer with FSNotify watcher


## knotidx Architecture

### Daemon

### Indexer

### Store

### GRPC


## License

MIT License

Copyright (c) 2024 Serg Podtynnyi
