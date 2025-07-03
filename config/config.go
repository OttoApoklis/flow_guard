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

type LogConfig struct {
	Level      string `yaml:"level"`
	File       string `yaml:"file"`
	MaxSize    int    `yaml:"max_size"`
	MaxBackups int    `yaml:"max_backups"`
	MaxAge     int    `yaml:"max_age"`
}

type GalileoConfig struct {
	AppID  string `yaml:"app_id"`
	Token  string `yaml:"token"`
	APIURL string `yaml:"apiurl"`
}

type Redis struct {
	IsCluster     bool     `yaml:"is_cluster"`
	RedisAddrs    []string `yaml:"redis_addrs"`
	RedisPassword string   `yaml:"redis_password"`
}

type FlowGuardConfig struct {
	Redis     Redis         `yaml:"redis"`
	Rules     []Rule        `yaml:"rules"`
	LogConfig LogConfig     `yaml:"log"`
	Galileo   GalileoConfig `yaml:"galileo"`
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
