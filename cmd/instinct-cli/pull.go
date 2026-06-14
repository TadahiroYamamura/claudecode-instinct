package main

import (
	"context"
	"fmt"
	"io"
)

func execPull(ctx context.Context, repo Repository, cfg *InstinctConfig, localBranch string, w io.Writer) error {
	if cfg.Dolt.RemoteURL == "" {
		return fmt.Errorf("dolt.remote_url is not configured in config.yml")
	}
	if cfg.Dolt.TeamBranch == "" {
		return fmt.Errorf("dolt.team_branch is not configured in config.yml")
	}
	if localBranch == "" {
		return fmt.Errorf("local branch is not configured in config.user.yml")
	}

	repo.EnsureRemote(ctx, cfg.Dolt.Refs, cfg.Dolt.RemoteURL)

	repo.Checkout(ctx, cfg.Dolt.TeamBranch) //nolint:errcheck
	if err := repo.Sync(ctx, "origin", cfg.Dolt.TeamBranch); err != nil {
		return fmt.Errorf("pull team branch: %w", err)
	}

	repo.Checkout(ctx, localBranch) //nolint:errcheck
	if err := repo.Sync(ctx, "origin", localBranch); err != nil {
		return fmt.Errorf("pull personal branch: %w", err)
	}

	fmt.Fprintf(w, "pulled %s and %s from %s\n", cfg.Dolt.TeamBranch, localBranch, cfg.Dolt.RemoteURL)
	return nil
}
