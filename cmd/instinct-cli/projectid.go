package main

import (
	"crypto/sha256"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

func resolveProjectID(dir string) (string, error) {
	remoteURL, err := gitOutput(dir, "remote", "get-url", "origin")
	if err != nil {
		return "", fmt.Errorf("git remote get-url: %w", err)
	}

	gitRoot, err := gitOutput(dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", fmt.Errorf("git rev-parse: %w", err)
	}

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	relPath, err := filepath.Rel(gitRoot, absDir)
	if err != nil {
		return "", err
	}

	hashInput := remoteURL + "#" + relPath
	hash := sha256.Sum256([]byte(hashInput))
	return fmt.Sprintf("%x", hash[:6]), nil
}

func gitOutput(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
