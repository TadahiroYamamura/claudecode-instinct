package main

import (
	"io"
	"os"
	"path/filepath"
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
