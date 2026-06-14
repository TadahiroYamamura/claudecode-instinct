package main

import (
	"context"
	"fmt"
)

func execCommit(ctx context.Context, repo Repository, message string) error {
	if message == "" {
		message = "observer: batch commit"
	}
	if err := repo.Commit(ctx, message); err != nil {
		return fmt.Errorf("dolt_commit: %w", err)
	}
	return nil
}
