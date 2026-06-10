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

func execPush(ctx context.Context, conn *sql.Conn, cfg *InstinctConfig, push doltPushFunc, w io.Writer) error {
	if cfg.Dolt.RemoteURL == "" {
		return fmt.Errorf("dolt.remote_url is not configured in config.yml")
	}
	// 既存リモートがあってもエラーにしない
	conn.ExecContext(ctx, "CALL dolt_remote('add', '--ref', ?, 'origin', ?)", //nolint
		cfg.Dolt.Refs, cfg.Dolt.RemoteURL)
	branch := cfg.Dolt.Branch
	if branch == "" {
		branch = "main"
	}
	if err := push(ctx, conn, "origin", branch); err != nil {
		return fmt.Errorf("push: %w", err)
	}
	fmt.Fprintf(w, "pushed %s to %s\n", branch, cfg.Dolt.RemoteURL)
	return nil
}
