package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// setup実行後に.instinct-db/data/が作成される
func TestSetup_CreatesDoltDBDirectory(t *testing.T) {
	dir := t.TempDir()

	if err := runSetup(dir); err != nil {
		t.Fatalf("runSetup: %v", err)
	}

	dataDir := filepath.Join(dir, ".instinct-db", "data")
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		t.Error(".instinct-db/data/ was not created")
	}
}

// setup実行後に.instinct-db/.gitignoreが作成され、ランタイムファイルが除外される
func TestSetup_CreatesGitignoreInInstinctDb(t *testing.T) {
	dir := t.TempDir()

	if err := runSetup(dir); err != nil {
		t.Fatalf("runSetup: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".instinct-db", ".gitignore"))
	if err != nil {
		t.Fatalf(".instinct-db/.gitignore not created: %v", err)
	}
	content := string(data)
	for _, entry := range []string{"data/", "observations.jsonl", ".observer.pid"} {
		if !strings.Contains(content, entry) {
			t.Errorf(".gitignore missing %q, got:\n%s", entry, content)
		}
	}
}

// setup実行後にconfig.ymlのdolt.branchが設定される
func TestSetup_ConfigYmlContainsBranch(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "myproject")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := runSetup(dir); err != nil {
		t.Fatalf("runSetup: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, ".instinct-db", "config.yml"))
	if err != nil {
		t.Fatalf("read config.yml: %v", err)
	}
	if !strings.Contains(string(data), "branch:") {
		t.Errorf("config.yml does not contain branch:, got:\n%s", data)
	}
}

// setup実行後にconfig.ymlのdolt.team_branchがmainに設定される
func TestSetup_ConfigYmlContainsTeamBranch(t *testing.T) {
	dir := t.TempDir()
	if err := runSetup(dir); err != nil {
		t.Fatalf("runSetup: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, ".instinct-db", "config.yml"))
	if err != nil {
		t.Fatalf("read config.yml: %v", err)
	}
	if !strings.Contains(string(data), "team_branch: main") {
		t.Errorf("config.yml does not contain team_branch: main, got:\n%s", data)
	}
}

// setup実行後にconfig.ymlのdolt.remote_urlがorigin remoteから設定される
func TestSetup_ConfigYmlContainsRemoteURL(t *testing.T) {
	dir := t.TempDir()
	mustRun(t, "git", "-C", dir, "init")
	mustRun(t, "git", "-C", dir, "remote", "add", "origin", "https://github.com/test/repo.git")

	if err := runSetup(dir); err != nil {
		t.Fatalf("runSetup: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, ".instinct-db", "config.yml"))
	if err != nil {
		t.Fatalf("read config.yml: %v", err)
	}
	if !strings.Contains(string(data), "remote_url: https://github.com/test/repo.git") {
		t.Errorf("config.yml does not contain remote_url, got:\n%s", data)
	}
}

func mustRun(t *testing.T, name string, args ...string) {
	t.Helper()
	if out, err := exec.Command(name, args...).CombinedOutput(); err != nil {
		t.Fatalf("%s %v: %v\n%s", name, args, err, out)
	}
}

// config.ymlのrefsがディレクトリ名から自動推定される
func TestSetup_ConfigYmlContainsRefsBasedOnDirName(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "myproject")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	if err := runSetup(dir); err != nil {
		t.Fatalf("runSetup: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".instinct-db", "config.yml"))
	if err != nil {
		t.Fatalf("read config.yml: %v", err)
	}
	if !strings.Contains(string(data), "refs/dolt/myproject/") {
		t.Errorf("config.yml does not contain refs/dolt/myproject/, got:\n%s", data)
	}
}
