package main

import (
	"os"
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
