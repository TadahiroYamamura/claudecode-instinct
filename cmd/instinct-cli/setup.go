package main

import (
	"context"
	"path/filepath"
)

func runSetup(projectDir string) error {
	dataDir := filepath.Join(projectDir, ".instinct-db", "data")
	return setupDB(context.Background(), dataDir)
}
