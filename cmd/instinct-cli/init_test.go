package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// initはgit remoteが未設定でも.instinct-db/dataを作成できる
func TestInit_CreatesDoltDBWithoutRemote(t *testing.T) {
	dir := t.TempDir()
	mustRun(t, "git", "-C", dir, "init")

	if err := execInit(dir, initParams{Yes: true}, nil, io.Discard); err != nil {
		t.Fatalf("execInit: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, ".instinct-db", "data")); os.IsNotExist(err) {
		t.Error(".instinct-db/data/ was not created")
	}
}

// initはconfig.user.ymlにdolt.branchを書き込む
func TestInit_WritesUserConfig(t *testing.T) {
	dir := t.TempDir()
	mustRun(t, "git", "-C", dir, "init")

	if err := execInit(dir, initParams{Branch: "alice", Yes: true}, nil, io.Discard); err != nil {
		t.Fatalf("execInit: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".instinct-db", "config.user.yml"))
	if err != nil {
		t.Fatalf("read config.user.yml: %v", err)
	}
	if !strings.Contains(string(data), "branch: alice") {
		t.Errorf("config.user.yml should contain branch: alice, got:\n%s", data)
	}
}
