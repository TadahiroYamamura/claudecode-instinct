package main

import (
	"crypto/sha256"
	"fmt"
	"os/exec"
	"testing"
)

func initRepo(t *testing.T, remote string) string {
	t.Helper()
	dir := t.TempDir()
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if err := cmd.Run(); err != nil {
			t.Fatalf("git %v: %v", args, err)
		}
	}
	run("init")
	run("remote", "add", "origin", remote)
	return dir
}

// resolveProjectIDはgit remoteにdir自身のリモートを使う（CWDではなく）
func TestResolveProjectID_UsesRemoteOfGivenDir(t *testing.T) {
	const remote = "git@github.com:test/unique-xyz.git"
	dir := initRepo(t, remote)

	id, err := resolveProjectID(dir)
	if err != nil {
		t.Fatalf("resolveProjectID: %v", err)
	}

	// dir直下なのでrel_pathは"."
	hash := sha256.Sum256([]byte(remote + "#."))
	expected := fmt.Sprintf("%x", hash[:6])
	if id != expected {
		t.Errorf("project_id = %q, want %q (expected to use test repo remote, not CWD remote)", id, expected)
	}
}

func TestResolveProjectID_ReturnsHashOfRemoteAndPath(t *testing.T) {
	dir := initRepo(t, "git@github.com:test/repo.git")

	id, err := resolveProjectID(dir)
	if err != nil {
		t.Fatalf("resolveProjectID: %v", err)
	}
	if len(id) != 12 {
		t.Errorf("expected 12-char project_id, got %q (len=%d)", id, len(id))
	}
}
