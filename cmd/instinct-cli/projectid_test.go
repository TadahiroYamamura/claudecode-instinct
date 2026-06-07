package main

import (
	"os/exec"
	"testing"
)

func TestResolveProjectID_ReturnsHashOfRemoteAndPath(t *testing.T) {
	dir := t.TempDir()

	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if err := cmd.Run(); err != nil {
			t.Fatalf("git %v: %v", args, err)
		}
	}
	run("init")
	run("remote", "add", "origin", "git@github.com:test/repo.git")

	id, err := resolveProjectID(dir)
	if err != nil {
		t.Fatalf("resolveProjectID: %v", err)
	}
	if len(id) != 12 {
		t.Errorf("expected 12-char project_id, got %q (len=%d)", id, len(id))
	}
}
