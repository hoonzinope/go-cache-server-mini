package config

import (
	"fmt"
	"os"

	"github.com/goccy/go-yaml"
)

type Config struct {
	Persistent struct {
		Type string `yaml:"type"`
	} `yaml:"persistent"`

	TTL struct {
		Default int64 `yaml:"default"`
		Max     int64 `yaml:"max"`
	} `yaml:"ttl"`

	HTTP struct {
		Enabled bool   `yaml:"enabled"`
		Address string `yaml:"address"`
	} `yaml:"http"`
}

func LoadConfig() (*Config, error) {
	yamlFile, err := os.ReadFile("config.yml")
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
