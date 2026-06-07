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
	configPath := filepath.Join(projectDir, ".instinct-db", "config.yml")
	content := fmt.Sprintf("dolt:\n  refs: refs/dolt/%s/\n", projectName)
	return os.WriteFile(configPath, []byte(content), 0o644)
}
