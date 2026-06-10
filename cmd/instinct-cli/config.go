package main

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

type InstinctConfig struct {
	Dolt struct {
		Refs       string `yaml:"refs"`
		Branch     string `yaml:"branch"`
		TeamBranch string `yaml:"team_branch"`
		RemoteURL  string `yaml:"remote_url"`
	} `yaml:"dolt"`
}

func loadConfig(dbDir string) (*InstinctConfig, error) {
	data, err := os.ReadFile(filepath.Join(dbDir, "config.yml"))
	if err != nil {
		return nil, fmt.Errorf("read config.yml: %w", err)
	}
	var cfg InstinctConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config.yml: %w", err)
	}
	return &cfg, nil
}
