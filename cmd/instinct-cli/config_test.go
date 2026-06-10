package main

import (
	"os"
	"path/filepath"
	"testing"
)

// loadConfigがconfig.ymlのobserver設定を返す
func TestLoadConfig_ReturnsObserverConfig(t *testing.T) {
	dir := t.TempDir()
	yml := "observer:\n  trigger_every: 30\n"
	if err := os.WriteFile(filepath.Join(dir, "config.yml"), []byte(yml), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := loadConfig(dir)
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	if cfg.Observer.TriggerEvery != 30 {
		t.Errorf("expected TriggerEvery=30, got %d", cfg.Observer.TriggerEvery)
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
