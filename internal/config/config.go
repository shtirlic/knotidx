package config

import (
	"log/slog"
	"os"

	"github.com/BurntSushi/toml"
)

const (
	configFile = "configs/knotidx.toml"
)

type IndexerConfig struct {
	Type   string
	Paths  []string
	Notify bool
}

type StoreConfig struct {
	Type string
	Path string
}

type GeneralConfig struct {
	Logging string
}

type KnotidxConfig struct {
}

type Config struct {
	Interval int
	Store    StoreConfig
	Indexer  []IndexerConfig
}

func DefaultConfig() Config {
	conf := Config{
		Interval: 5,
		Store: StoreConfig{
			Type: "badger",
		},
	}
	return conf
}

func (c Config) Load() (Config, error) {
	slog.Info("Config Load", "path", configFile)
	configData, err := os.ReadFile(configFile)
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
