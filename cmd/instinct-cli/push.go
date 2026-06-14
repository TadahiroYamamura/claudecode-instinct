package main

import (
	"context"
	"fmt"
	"io"
)

func execPush(ctx context.Context, repo Repository, cfg *InstinctConfig, localBranch string, w io.Writer) error {
	if cfg.Dolt.RemoteURL == "" {
		return fmt.Errorf("dolt.remote_url is not configured in config.yml")
	}
	if localBranch == "" {
		return fmt.Errorf("dolt.branch is not configured in config.user.yml")
	}
	repo.EnsureRemote(ctx, cfg.Dolt.Refs, cfg.Dolt.RemoteURL)
	if err := repo.Upload(ctx, "origin", localBranch); err != nil {
		return fmt.Errorf("push: %w", err)
	}
	fmt.Fprintf(w, "pushed %s to %s\n", localBranch, cfg.Dolt.RemoteURL)
	return nil
}
