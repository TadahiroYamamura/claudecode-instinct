package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"gopkg.in/yaml.v2"
)

//go:embed templates/gitignore.tmpl
var instinctDbGitignore []byte

//go:embed templates/config_team.tmpl
var configTeamTemplate string

//go:embed templates/config_user.tmpl
var configUserTemplate string

var configTeamTmpl = template.Must(template.New("config_team").Parse(configTeamTemplate))
var configUserTmpl = template.Must(template.New("config_user").Parse(configUserTemplate))

const defaultTeamBranch = "main"

type teamConfigData struct {
	Refs       string
	TeamBranch string
	RemoteURL  string
}

type userConfigData struct {
	Branch string
}

func writeTeamConfig(dbDir, refs, teamBranch, remoteURL string) error {
	var buf bytes.Buffer
	if err := configTeamTmpl.Execute(&buf, teamConfigData{
		Refs:       refs,
		TeamBranch: teamBranch,
		RemoteURL:  remoteURL,
	}); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dbDir, "config.team.yml"), buf.Bytes(), 0o644)
}

func sanitizeBranchName(name string) string {
	var out []rune
	for _, r := range name {
		if r == ' ' || r == '/' || r == '\\' {
			out = append(out, '-')
		} else {
			out = append(out, r)
		}
	}
	return string(out)
}

func writeUserConfig(dbDir, branch string) error {
	var buf bytes.Buffer
	if err := configUserTmpl.Execute(&buf, userConfigData{Branch: branch}); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dbDir, "config.user.yml"), buf.Bytes(), 0o644)
}

type DoltConfig struct {
	Refs       string `yaml:"refs"`
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

type ConfidenceConfig struct {
	ReviewMin int `yaml:"review_min"`
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
