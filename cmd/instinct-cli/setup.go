package main

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

//go:embed templates/gitignore.tmpl
var instinctDbGitignore []byte

func gitUserName() (string, error) {
	out, err := exec.Command("git", "config", "user.name").Output()
	if err != nil {
		return "", fmt.Errorf("git config user.name: %w", err)
	}
	name := strings.TrimSpace(string(out))
	if name == "" {
		return "", fmt.Errorf("git config user.name is empty")
	}
	return name, nil
}

var resolveGitUserName = gitUserName

func runSetup(projectDir string) error {
	if err := setupDB(context.Background(), instinctDataDir(projectDir)); err != nil {
		return err
	}

	branch, err := resolveGitUserName()
	if err != nil {
		return err
	}

	projectName := filepath.Base(projectDir)
	dbDir := instinctDbDir(projectDir)

	configPath := filepath.Join(dbDir, "config.yml")
	config := fmt.Sprintf("dolt:\n  refs: refs/dolt/%s/\n  branch: %s\n", projectName, branch)
	if err := os.WriteFile(configPath, []byte(config), 0o644); err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(dbDir, ".gitignore"), instinctDbGitignore, 0o644)
}
