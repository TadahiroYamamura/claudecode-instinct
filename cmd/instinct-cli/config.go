package main

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

type DoltConfig struct {
	Refs       string `yaml:"refs"`
	Branch     string `yaml:"branch"`
	TeamBranch string `yaml:"team_branch"`
	RemoteURL  string `yaml:"remote_url"`
}

type UserDoltConfig struct {
	Branch string `yaml:"branch"`
}

type UserConfig struct {
	Dolt UserDoltConfig `yaml:"dolt"`
}

type ObserverConfig struct {
	Enabled      bool   `yaml:"enabled"`
	TriggerEvery int    `yaml:"trigger_every"`
	ActiveHours  string `yaml:"active_hours"`
}

type ConfidenceThresholds struct {
	Low    int `yaml:"low"`
	Medium int `yaml:"medium"`
	High   int `yaml:"high"`
}

type ConfidenceConfig struct {
	Thresholds ConfidenceThresholds `yaml:"thresholds"`
}

type DedupConfig struct {
	AutoRunBeforePush   bool    `yaml:"auto_run_before_push"`
	SimilarityThreshold float64 `yaml:"similarity_threshold"`
}

type InstinctConfig struct {
	Observer   ObserverConfig   `yaml:"observer"`
	Confidence ConfidenceConfig `yaml:"confidence"`
	Dedup      DedupConfig      `yaml:"dedup"`
	Dolt       DoltConfig       `yaml:"dolt"`
}

func loadConfig(dbDir string) (*InstinctConfig, error) {
	data, err := os.ReadFile(filepath.Join(dbDir, "config.team.yml"))
	if err != nil {
		return nil, fmt.Errorf("read config.team.yml: %w", err)
	}
	var cfg InstinctConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config.team.yml: %w", err)
	}
	return &cfg, nil
}

func loadUserConfig(dbDir string) (*UserConfig, error) {
	data, err := os.ReadFile(filepath.Join(dbDir, "config.user.yml"))
	if err != nil {
		return nil, fmt.Errorf("read config.user.yml: %w", err)
	}
	var cfg UserConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config.user.yml: %w", err)
	}
	return &cfg, nil
}
