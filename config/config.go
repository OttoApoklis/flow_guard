package config

import (
	"gopkg.in/yaml.v3"
	"os"
)

type Rule struct {
	Path   string `yaml:"path"`
	Limit  int    `yaml:"limit"`
	Window int    `yaml:"window"`
}

type FlowGuardConfig struct {
	RedisAddr string `yaml:"redis_addr"`
	Rules     []Rule `yaml:"rules"`
}

type Config struct {
	FlowGuard FlowGuardConfig `yaml:"flow_guard"`
}

func LoadConfig(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	decoder := yaml.NewDecoder(f)
	var cfg Config
	err = decoder.Decode(&cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}
