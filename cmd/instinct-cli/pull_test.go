package main

import (
	"context"
	"database/sql"
	"strings"
	"testing"
)

var nopPull doltPullFunc = func(_ context.Context, _ *sql.Conn, _, _ string) error {
	return nil
}

// execPullはremote_urlが未設定のときエラーを返す
func TestExecPull_FailsWhenRemoteURLEmpty(t *testing.T) {
	ctx, conn := setupTestDB(t)

	var buf strings.Builder
	if err := execPull(ctx, conn, &InstinctConfig{}, nopPull, &buf); err == nil {
		t.Fatal("expected error when remote_url is empty, got nil")
	}
}

// execPullはteam_branchが未設定のときエラーを返す
func TestExecPull_FailsWhenTeamBranchEmpty(t *testing.T) {
	ctx, conn := setupTestDB(t)

	cfg := &InstinctConfig{Dolt: DoltConfig{RemoteURL: "git@github.com:org/repo.git"}}
	var buf strings.Builder
	if err := execPull(ctx, conn, cfg, nopPull, &buf); err == nil {
		t.Fatal("expected error when team_branch is empty, got nil")
	}
}

// execPullはdolt_remoteにoriginを登録する
func TestExecPull_RegistersRemote(t *testing.T) {
	ctx, conn := setupTestDB(t)

	cfg := &InstinctConfig{
		Dolt: DoltConfig{
			Refs:       "refs/dolt/myproject/",
			TeamBranch: "main",
			RemoteURL:  "git@github.com:org/repo.git",
		},
	}

	var buf strings.Builder
	_ = execPull(ctx, conn, cfg, nopPull, &buf)

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

// execPullはpullFnに正しいremoteとteam_branchを渡す
func TestExecPull_CallsPullWithCorrectArgs(t *testing.T) {
	ctx, conn := setupTestDB(t)

	cfg := &InstinctConfig{
		Dolt: DoltConfig{
			Refs:       "refs/dolt/myproject/",
			TeamBranch: "main",
			RemoteURL:  "git@github.com:org/repo.git",
		},
	}

	var gotRemote, gotBranch string
	capturePull := func(_ context.Context, _ *sql.Conn, remote, branch string) error {
		gotRemote, gotBranch = remote, branch
		return nil
	}

	var buf strings.Builder
	if err := execPull(ctx, conn, cfg, capturePull, &buf); err != nil {
		t.Fatalf("execPull: %v", err)
	}
	if gotRemote != "origin" {
		t.Errorf("expected remote %q, got %q", "origin", gotRemote)
	}
	if gotBranch != "main" {
		t.Errorf("expected branch %q, got %q", "main", gotBranch)
	}
}
