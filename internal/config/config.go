package config

import (
	"fmt"
	"os"

	"github.com/go-playground/validator/v10"
	"go.yaml.in/yaml/v4"
)


type Config struct {
	Server ServerConfig `yaml:"server" validate:"required"`
}

type ServerConfig struct {
	Listen int `yaml:"listen" validate:"gt=0,lte=65535"`
	Workers int `yaml:"workers"`
	Upstreams []UpstreamConfig `yaml:"upstreams" validate:"gt=0,dive,required"`
	Paths []PathConfig `yaml:"paths" validate:"gt=0,dive,required"`
	Headers []HeaderConfig `yaml:"headers" validate:"omitempty,dive"`
}

type UpstreamConfig struct {
	ID string `yaml:"id" validate:"required"`
	URL string `yaml:"url" validate:"required,url"`
}

type PathConfig struct {
	Path string `yaml:"path" validate:"required"`
	Upstreams []string `yaml:"upstreams" validate:"gt=0,dive,required"`
}

type HeaderConfig struct {
	Key string `yaml:"key" validate:"required"`
	Value string `yaml:"value" validate:"required"`
}

var validate = validator.New(validator.WithRequiredStructEnabled())

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

	err = ValidateConfig(config)

	if err != nil {
		return nil,  fmt.Errorf("validation failed: %v", err)
	}

	return  config, nil
}

func ValidateConfig(cfg *Config) error {
	return validate.Struct(cfg)
}