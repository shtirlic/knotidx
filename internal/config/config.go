package config

import (
	"log/slog"
	"os"

	"github.com/BurntSushi/toml"
)

const (
	configFile = "configs/knot.toml"
)

type IndexerConfig struct {
	Type  string
	Paths []string
}

type StoreConfig struct {
	Type string
	Path string
}

type GeneralConfig struct {
	Logging string
}

type KnotctlConfig struct {
}

type KnotdConfig struct {
	Interval int
	Store    StoreConfig
	Indexer  []IndexerConfig
}

type Config struct {
	General GeneralConfig
	Knotd   KnotdConfig
	Knotctl KnotctlConfig
}

func DefaultConfig() *Config {
	conf := Config{
		Knotd: KnotdConfig{
			Interval: 5,
			Store: StoreConfig{
				Type: "badger",
			},
		},
	}
	return &conf
}

func (c *Config) Load() (*Config, error) {
	slog.Info("Config Load", "path", configFile)
	configData, err := os.ReadFile(configFile)
	if err != nil {
		return nil, err
	}
	meta, err := toml.Decode(string(configData), c)
	if err != nil {
		return nil, err
	}
	slog.Debug("Config load", "meta", meta)
	slog.Debug("Config Load", "config", *c)
	return c, err
}
