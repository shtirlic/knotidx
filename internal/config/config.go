package config

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/BurntSushi/toml"
)

// GRPCServerType represents the type of the gRPC server, either TCP or Unix.
type GRPCServerType string

// Default values for various configuration parameters.
const (
	defaultConfigFile     = "configs/knotidx.toml"
	defaultGrpcPort       = 5319
	defaultGrpcHost       = "localhost"
	defaultGrpcSocketPath = "knotidx.sock"
	defaultInterval       = 5 // seconds
	defaultStoreType      = "badger"

	// GRPCServerTcpType represents the TCP type for the gRPC server.
	GrpcServerTcpType GRPCServerType = "tcp"
	// GrpcServerUnixType represents the Unix type for the gRPC server.
	GrpcServerUnixType GRPCServerType = "unix"
)

// IndexerConfig represents the configuration for an indexer.
type IndexerConfig struct {
	Type               string   // Type of the indexer.
	Paths              []string // List of paths to index.
	Notify             bool     // Enable/disable file system notifications.
	ExcludeDirFilters  []string // List of directory filters to exclude during indexing.
	ExcludeFileFilters []string // List of file filters to exclude during indexing.
}

// StoreConfig represents the configuration for the data store.
type StoreConfig struct {
	Type string // Type of the data store.
	Path string // Path for the data store.
}

// GRPCConfig represents the configuration for the gRPC server.
type GRPCConfig struct {
	Server bool           // Enable/disable the gRPC server.
	Port   int            // Port on which the gRPC server listens.
	Type   GRPCServerType // Type of the gRPC server (TCP or Unix).
	Path   string         // Path for Unix socket (if applicable).
	Host   string         // Host for TCP server.
}

// Config represents the overall application configuration.
type Config struct {
	Interval int             // Interval for indexing.
	GRPC     GRPCConfig      // gRPC server configuration.
	Store    StoreConfig     // Data store configuration.
	Indexer  []IndexerConfig // List of indexer configurations.
}

// DefaultConfig returns the default configuration for the application.
func DefaultConfig() Config {
	// TODO: make nice slash handling( clear path)
	defaultBaseSocketPath := os.Getenv("XDG_RUNTIME_DIR")
	if defaultBaseSocketPath != "" {
		defaultBaseSocketPath = defaultBaseSocketPath + "/"
	}
	// Create and return the default configuration.
	conf := Config{
		Interval: defaultInterval, // Default interval for indexing.
		Store: StoreConfig{
			Type: defaultStoreType, // Default type for the data store.
		},
		GRPC: GRPCConfig{
			Server: true,                                          // Enable the gRPC server by default.
			Type:   GrpcServerUnixType,                            // Default type for the gRPC server (Unix socket).
			Port:   defaultGrpcPort,                               // Default port for the gRPC server (TCP).
			Host:   defaultGrpcHost,                               // Default host for the gRPC server (TCP).
			Path:   defaultBaseSocketPath + defaultGrpcSocketPath, // Default path for the Unix socket.
		},
	}
	return conf
}

// Load reads configuration data from the specified file path in TOML format and updates
// the current configuration with the loaded values. If the file path is empty, the
// defaultConfigFile constant is used. The method returns the updated configuration and
// any encountered error during file reading or decoding.
func (c Config) Load(path string) (Config, error) {
	if path == "" {
		path = defaultConfigFile
	}
	slog.Info("Config Load", "path", path)
	configData, err := os.ReadFile(path)
	if err != nil {
		return c, fmt.Errorf("error reading config file: %w", err)
	}
	_, err = toml.Decode(string(configData), &c)
	if err != nil {
		return c, fmt.Errorf("error decoding config data: %w", err)

	}
	// slog.Debug("Config load", "meta", meta)
	slog.Debug("Config Load", "config", c)
	return c, nil
}
