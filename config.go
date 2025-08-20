package main

import (
	"os"

	"gopkg.in/yaml.v3"
)

type MetricConfig struct {
	Name   string `yaml:"name"`
	Path   string `yaml:"path"`
	Type   string `yaml:"type"`   // counter | gauge
	Derive string `yaml:"derive"` // none | rate_per_sec | delta
	Color  string `yaml:"color"`
}

type Config struct {
	URI             string         `yaml:"uri"`
	RefreshInterval string         `yaml:"refreshInterval"`
	Window          string         `yaml:"window"`
	Metrics         []MetricConfig `yaml:"metrics"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Config
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	return &c, nil
}
