package main

import (
	"context"
	"database/sql"
	"fmt"
	"io"
)

type doltPullFunc func(ctx context.Context, conn *sql.Conn, remote, branch string) error

var defaultDoltPull doltPullFunc = func(ctx context.Context, conn *sql.Conn, remote, branch string) error {
	_, err := conn.ExecContext(ctx, "CALL dolt_pull(?, ?)", remote, branch)
	return err
}

func execPull(ctx context.Context, conn *sql.Conn, cfg *InstinctConfig, localBranch string, pull doltPullFunc, w io.Writer) error {
	if cfg.Dolt.RemoteURL == "" {
		return fmt.Errorf("dolt.remote_url is not configured in config.yml")
	}
	if cfg.Dolt.TeamBranch == "" {
		return fmt.Errorf("dolt.team_branch is not configured in config.yml")
	}
	if localBranch == "" {
		return fmt.Errorf("local branch is not configured in config.user.yml")
	}

	ensureRemote(ctx, conn, cfg.Dolt.Refs, cfg.Dolt.RemoteURL)

	// チームブランチを更新してから個人ブランチを更新する
	_, _ = conn.ExecContext(ctx, "CALL dolt_checkout(?)", cfg.Dolt.TeamBranch)
	if err := pull(ctx, conn, "origin", cfg.Dolt.TeamBranch); err != nil {
		return fmt.Errorf("pull team branch: %w", err)
	}

	_, _ = conn.ExecContext(ctx, "CALL dolt_checkout(?)", localBranch)
	if err := pull(ctx, conn, "origin", localBranch); err != nil {
		return fmt.Errorf("pull personal branch: %w", err)
	}

	fmt.Fprintf(w, "pulled %s and %s from %s\n", cfg.Dolt.TeamBranch, localBranch, cfg.Dolt.RemoteURL)
	return nil
}
