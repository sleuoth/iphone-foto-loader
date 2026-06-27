package config

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

type HelperConfig struct {
	Path string `toml:"path"`
}

type DeviceConfig struct {
	Name   string `toml:"name"`
	Target string `toml:"target"`
}

type Config struct {
	Helper  HelperConfig            `toml:"helper"`
	Devices map[string]DeviceConfig `toml:"devices"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if cfg.Devices == nil {
		cfg.Devices = make(map[string]DeviceConfig)
	}
	return &cfg, nil
}

func (c *Config) Save(path string) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create config: %w", err)
	}
	defer file.Close()

	encoder := toml.NewEncoder(file)
	if err := encoder.Encode(c); err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	return nil
}
