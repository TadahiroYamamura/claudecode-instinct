package main

import (
	"context"
	"database/sql"
	"strings"
	"testing"
)

var nopPush doltPushFunc = func(_ context.Context, _ *sql.Conn, _, _ string) error {
	return nil
}

// execPushはdolt_remoteにoriginを登録する
func TestExecPush_RegistersRemote(t *testing.T) {
	ctx, conn := setupTestDB(t)

	cfg := &InstinctConfig{
		Dolt: DoltConfig{
			Refs:      "refs/dolt/myproject/",
			RemoteURL: "git@github.com:org/repo.git",
		},
	}

	var buf strings.Builder
	_ = execPush(ctx, conn, cfg, "tadahiro", nopPush, &buf)

	var count int
	if err := conn.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM dolt_remotes WHERE name = 'origin'",
	).Scan(&count); err != nil {
		t.Fatalf("query dolt_remotes: %v", err)
	}
	if count != 1 {
		t.Errorf("expected origin remote to be registered, got count=%d", count)
	}
}

// execPushはpushFnに正しいremoteとbranchを渡す
func TestExecPush_CallsPushWithCorrectArgs(t *testing.T) {
	ctx, conn := setupTestDB(t)

	cfg := &InstinctConfig{
		Dolt: DoltConfig{
			Refs:      "refs/dolt/myproject/",
			RemoteURL: "git@github.com:org/repo.git",
		},
	}

	var gotRemote, gotBranch string
	capturePush := func(_ context.Context, _ *sql.Conn, remote, branch string) error {
		gotRemote, gotBranch = remote, branch
		return nil
	}

	var buf strings.Builder
	if err := execPush(ctx, conn, cfg, "tadahiro", capturePush, &buf); err != nil {
		t.Fatalf("execPush: %v", err)
	}
	if gotRemote != "origin" {
		t.Errorf("expected remote %q, got %q", "origin", gotRemote)
	}
	if gotBranch != "tadahiro" {
		t.Errorf("expected branch %q, got %q", "tadahiro", gotBranch)
	}
}

// execPushはbranchが未設定のときエラーを返す（main へのフォールバック禁止）
func TestExecPush_FailsWhenBranchEmpty(t *testing.T) {
	ctx, conn := setupTestDB(t)

	cfg := &InstinctConfig{
		Dolt: DoltConfig{
			RemoteURL: "git@github.com:org/repo.git",
		},
	}

	var buf strings.Builder
	if err := execPush(ctx, conn, cfg, "", nopPush, &buf); err == nil {
		t.Fatal("expected error when branch is empty, got nil")
	}
}

// execPushはremote_urlが未設定のときエラーを返す
func TestExecPush_FailsWhenRemoteURLEmpty(t *testing.T) {
	ctx, conn := setupTestDB(t)

	var buf strings.Builder
	err := execPush(ctx, conn, &InstinctConfig{}, "tadahiro", nopPush, &buf)
	if err == nil {
		t.Fatal("expected error when remote_url is empty, got nil")
	}
}
