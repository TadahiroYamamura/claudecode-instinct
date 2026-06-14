package main

import (
	"os"
	"path/filepath"
	"testing"
)

// loadConfigがconfig.team.ymlのobserver設定を返す
func TestLoadConfig_ReturnsObserverConfig(t *testing.T) {
	dir := t.TempDir()
	yml := "observer:\n  trigger_every: 30\n"
	if err := os.WriteFile(filepath.Join(dir, "config.team.yml"), []byte(yml), 0o644); err != nil {
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

// loadConfigがconfig.team.ymlのteam_branchを返す
func TestLoadConfig_ReturnsTeamBranch(t *testing.T) {
	dir := t.TempDir()
	yml := "dolt:\n  team_branch: staging\n"
	if err := os.WriteFile(filepath.Join(dir, "config.team.yml"), []byte(yml), 0o644); err != nil {
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

// loadUserConfigがconfig.user.ymlのdolt.branchを返す
func TestLoadUserConfig_ReturnsBranch(t *testing.T) {
	dir := t.TempDir()
	yml := "dolt:\n  branch: tadahiro\n"
	if err := os.WriteFile(filepath.Join(dir, "config.user.yml"), []byte(yml), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := loadUserConfig(dir)
	if err != nil {
		t.Fatalf("loadUserConfig: %v", err)
	}
	if cfg.Dolt.Branch != "tadahiro" {
		t.Errorf("expected Branch=tadahiro, got %q", cfg.Dolt.Branch)
	}
}

func TestSanitizeBranchName(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"alice", "alice"},
		{"Alice", "alice"},
		{"Tadahiro Yamamura", "tadahiro_yamamura"},
		{"John O'Brien", "john_obrien"},
		{"user@example.com", "userexamplecom"},
		{"Bob123", "bob123"},
		{"--dash--", "--dash--"},
		{"my_user", "my_user"},
		{"山田 太郎", "_"},
	}
	for _, c := range cases {
		t.Run(c.input, func(t *testing.T) {
			if got := sanitizeBranchName(c.input); got != c.want {
				t.Errorf("sanitizeBranchName(%q) = %q, want %q", c.input, got, c.want)
			}
		})
	}
}

// config.user.ymlが存在しない場合はエラーを返す
func TestLoadUserConfig_ErrorWhenAbsent(t *testing.T) {
	dir := t.TempDir()

	_, err := loadUserConfig(dir)
	if err == nil {
		t.Error("expected error when config.user.yml is absent")
	}
}
