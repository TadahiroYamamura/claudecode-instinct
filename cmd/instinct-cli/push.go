package main

import (
	"context"
	"database/sql"
	"fmt"
	"io"
)

type doltPushFunc func(ctx context.Context, conn *sql.Conn, remote, branch string) error

var defaultDoltPush doltPushFunc = func(ctx context.Context, conn *sql.Conn, remote, branch string) error {
	_, err := conn.ExecContext(ctx, "CALL dolt_push(?, ?)", remote, branch)
	return err
}

func ensureRemote(ctx context.Context, conn *sql.Conn, refs, remoteURL string) {
	conn.ExecContext(ctx, "CALL dolt_remote('add', '--ref', ?, 'origin', ?)", refs, remoteURL) //nolint
}

func execPush(ctx context.Context, conn *sql.Conn, cfg *InstinctConfig, push doltPushFunc, w io.Writer) error {
	if cfg.Dolt.RemoteURL == "" {
		return fmt.Errorf("dolt.remote_url is not configured in config.yml")
	}
	if cfg.Dolt.Branch == "" {
		return fmt.Errorf("dolt.branch is not configured in config.yml")
	}
	ensureRemote(ctx, conn, cfg.Dolt.Refs, cfg.Dolt.RemoteURL)
	if err := push(ctx, conn, "origin", cfg.Dolt.Branch); err != nil {
		return fmt.Errorf("push: %w", err)
	}
	fmt.Fprintf(w, "pushed %s to %s\n", cfg.Dolt.Branch, cfg.Dolt.RemoteURL)
	return nil
}
