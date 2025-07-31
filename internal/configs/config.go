package configs

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

type Config struct {
	TargetDomains []TargetDomain `yaml:"target_domains"`
}

func NewConfig(filePath string) (Config, error) {
	var (
		config = Config{}
		err    error
	)

	if filePath == "" {
		return Config{}, fmt.Errorf("file path is not empty")
	}

	configBytes, err := os.ReadFile(filePath)
	if err != nil {
		return Config{}, fmt.Errorf("failed to read YAML file: %w", err)
	}

	err = yaml.Unmarshal(configBytes, &config)
	if err != nil {
		return Config{}, fmt.Errorf("failed to unmarshal YAML: %w", err)
	}

	return config, nil
}


