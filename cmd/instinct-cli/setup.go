package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

func runSetup(projectDir string) error {
	if err := setupDB(context.Background(), instinctDataDir(projectDir)); err != nil {
		return err
	}

	projectName := filepath.Base(projectDir)
	dbDir := instinctDbDir(projectDir)

	configPath := filepath.Join(dbDir, "config.yml")
	if err := os.WriteFile(configPath, []byte(fmt.Sprintf("dolt:\n  refs: refs/dolt/%s/\n", projectName)), 0o644); err != nil {
		return err
	}

	gitignorePath := filepath.Join(dbDir, ".gitignore")
	gitignoreContent := "data/\nobservations.jsonl\nobservations.archive/\n.observer.pid\n.observer-signal-counter\n"
	return os.WriteFile(gitignorePath, []byte(gitignoreContent), 0o644)
}
