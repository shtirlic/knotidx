package config

import (
	"log/slog"
	"os"

	"github.com/BurntSushi/toml"
)

type GrpcServerType string

const (
	defaultConfigFile     = "configs/knotidx.toml"
	defaultGrpcPort       = 5319
	defaultGrpcHost       = "localhost"
	defaultGrpcSocketPath = "knotidx.sock"
	defaultInterval       = 5 // seconds
	defaultStoreType      = "badger"

	GrpcServerTcpType  GrpcServerType = "tcp"
	GrpcServerUnixType GrpcServerType = "unix"
)

type IndexerConfig struct {
	Type               string
	Paths              []string
	Notify             bool
	ExcludeDirFilters  []string
	ExcludeFileFilters []string
}

type StoreConfig struct {
	Type string
	Path string
}

type GrpcConfig struct {
	Server bool
	Port   int
	Type   GrpcServerType
	Path   string
	Host   string
}

type Config struct {
	Interval int
	Grpc     GrpcConfig
	Store    StoreConfig
	Indexer  []IndexerConfig
}

func DefaultConfig() Config {
	// TODO: make nice slash handling( clear path)
	defaultBaseSocketPath := os.Getenv("XDG_RUNTIME_DIR")
	if defaultBaseSocketPath != "" {
		defaultBaseSocketPath = defaultBaseSocketPath + "/"
	}
	conf := Config{
		Interval: defaultInterval,
		Store: StoreConfig{
			Type: defaultStoreType,
		},
		Grpc: GrpcConfig{
			Server: true,
			Type:   GrpcServerUnixType,
			Port:   defaultGrpcPort,
			Host:   defaultGrpcHost,
			Path:   defaultBaseSocketPath + defaultGrpcSocketPath,
		},
	}
	return conf
}

func (c Config) Load(configPath string) (Config, error) {
	if configPath == "" {
		configPath = defaultConfigFile
	}
	slog.Info("Config Load", "path", configPath)
	configData, err := os.ReadFile(configPath)
	if err != nil {
		return c, err
	}
	_, err = toml.Decode(string(configData), &c)
	if err != nil {
		return c, err
	}
	// slog.Debug("Config load", "meta", meta)
	slog.Debug("Config Load", "config", c)
	return c, err
}
