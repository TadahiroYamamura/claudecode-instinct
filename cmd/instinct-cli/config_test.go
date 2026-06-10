package main

import (
	"os"
	"path/filepath"
	"testing"
)

// teamBranchFromConfigがconfig.ymlのteam_branchを返す
func TestTeamBranchFromConfig_ReturnsConfiguredValue(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".instinct-db"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	yml := "dolt:\n  team_branch: custom\n"
	if err := os.WriteFile(filepath.Join(dir, ".instinct-db", "config.yml"), []byte(yml), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if got := teamBranchFromConfig(dir); got != "custom" {
		t.Errorf("expected custom, got %q", got)
	}
}

// teamBranchFromConfigはconfig.ymlが存在しない場合mainを返す
func TestTeamBranchFromConfig_DefaultsToMain(t *testing.T) {
	if got := teamBranchFromConfig(t.TempDir()); got != "main" {
		t.Errorf("expected main, got %q", got)
	}
}

// loadConfigがconfig.ymlのteam_branchを返す
func TestLoadConfig_ReturnsTeamBranch(t *testing.T) {
	dir := t.TempDir()
	yml := "dolt:\n  team_branch: staging\n"
	if err := os.WriteFile(filepath.Join(dir, "config.yml"), []byte(yml), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := loadConfig(dir)
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	if cfg.Dolt.TeamBranch != "staging" {
		t.Errorf("expected TeamBranch=staging, got %q", cfg.Dolt.TeamBranch)
	}
}
