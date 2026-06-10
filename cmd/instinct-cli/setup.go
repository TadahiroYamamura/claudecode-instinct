package main

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
)

//go:embed templates/gitignore.tmpl
var instinctDbGitignore []byte

func runSetup(projectDir string) error {
	if err := setupDB(context.Background(), instinctDataDir(projectDir)); err != nil {
		return err
	}

	branch, err := gitConfigValue("user.name")
	if err != nil {
		return err
	}

	projectName := filepath.Base(projectDir)
	dbDir := instinctDbDir(projectDir)

	configPath := filepath.Join(dbDir, "config.yml")
	config := fmt.Sprintf("dolt:\n  refs: refs/dolt/%s/\n  branch: %s\n  team_branch: main\n", projectName, branch)
	if err := os.WriteFile(configPath, []byte(config), 0o644); err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(dbDir, ".gitignore"), instinctDbGitignore, 0o644)
}
