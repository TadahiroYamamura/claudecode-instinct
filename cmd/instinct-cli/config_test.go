package main

import (
	"os"
	"path/filepath"
	"testing"
)

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
