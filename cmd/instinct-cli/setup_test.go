package main

import (
	"os"
	"path/filepath"
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
