package config

import (
	"fmt"
	"os"

	"go.yaml.in/yaml/v4"
)


type Config struct {
	Server ServerConfig `yaml:"server"`
}

type ServerConfig struct {
	Listen int `yaml:"listen"`
	Workers int `yaml:"workers"`
	Upstreams []UpstreamConfig `yaml:"upstreams"`
	Paths []PathConfig `yaml:"paths"`
	Headers []HeaderConfig `yaml:"headers"`
}

type UpstreamConfig struct {
	ID string `yaml:"id"`
	URL string `yaml:"url"`
}

type PathConfig struct {
	Path string `yaml:"path"`
	Upstreams []string `yaml:"upstreams"`
}

type HeaderConfig struct {
	Key string `yaml:"key"`
	Value string `yaml:"value"`
}

func ParseConfig(path string) (*Config, error) {
	file, err := os.ReadFile(path)	 
	
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	config := &Config{}
	err = yaml.Unmarshal(file, config)

	if err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return  config, nil
}