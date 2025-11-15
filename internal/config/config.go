package config

import (
	"fmt"
	"os"

	"github.com/goccy/go-yaml"
)

type PersistentConfig struct {
	Type string `yaml:"type"`
	Path string `yaml:"path"`
}

type TTLConfig struct {
	Default int64 `yaml:"default"`
	Max     int64 `yaml:"max"`
}

type HTTPConfig struct {
	Enabled bool   `yaml:"enabled"`
	Address string `yaml:"address"`
}

type Config struct {
	Persistent PersistentConfig `yaml:"persistent"`
	TTL        TTLConfig        `yaml:"ttl"`
	HTTP       HTTPConfig       `yaml:"http"`
}

func LoadConfig(configFilePath string) (*Config, error) {
	yamlFile, err := os.ReadFile(configFilePath)
	if err != nil {
		return nil, err
	}
	// os.ExpandEnv to replace environment variables in the YAML content
	expandedYaml := os.ExpandEnv(string(yamlFile))

	var config Config
	err = yaml.Unmarshal([]byte(expandedYaml), &config)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling config file: %w", err)
	}
	return &config, nil
}

func LoadTestConfig() *Config {
	return &Config{
		Persistent: PersistentConfig{
			Type: "aof",
			Path: "./persistent_data/",
		},
		TTL: TTLConfig{
			Default: 86400,
			Max:     604800,
		},
		HTTP: HTTPConfig{
			Enabled: true,
			Address: ":8080",
		},
	}
}
