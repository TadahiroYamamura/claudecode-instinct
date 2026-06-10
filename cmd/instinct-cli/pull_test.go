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
	if err := execPull(ctx, conn, &InstinctConfig{}, "myuser", nopPull, &buf); err == nil {
		t.Fatal("expected error when remote_url is empty, got nil")
	}
}

// execPullはteam_branchが未設定のときエラーを返す
func TestExecPull_FailsWhenTeamBranchEmpty(t *testing.T) {
	ctx, conn := setupTestDB(t)

	cfg := &InstinctConfig{Dolt: DoltConfig{RemoteURL: "git@github.com:org/repo.git"}}
	var buf strings.Builder
	if err := execPull(ctx, conn, cfg, "myuser", nopPull, &buf); err == nil {
		t.Fatal("expected error when team_branch is empty, got nil")
	}
}

// execPullはlocalBranchが空のときエラーを返す
func TestExecPull_FailsWhenLocalBranchEmpty(t *testing.T) {
	ctx, conn := setupTestDB(t)

	cfg := &InstinctConfig{
		Dolt: DoltConfig{
			TeamBranch: "main",
			RemoteURL:  "git@github.com:org/repo.git",
		},
	}
	var buf strings.Builder
	if err := execPull(ctx, conn, cfg, "", nopPull, &buf); err == nil {
		t.Fatal("expected error when localBranch is empty, got nil")
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
	_ = execPull(ctx, conn, cfg, "myuser", nopPull, &buf)

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

// execPullはチームブランチと個人ブランチの両方をこの順番でpullする
func TestExecPull_PullsBothBranchesInOrder(t *testing.T) {
	ctx, conn := setupTestDB(t)

	// 個人ブランチをDB内に作成しておく
	if _, err := conn.ExecContext(ctx, "CALL dolt_checkout('-b', 'myuser')"); err != nil {
		t.Fatalf("create myuser branch: %v", err)
	}
	if _, err := conn.ExecContext(ctx, "CALL dolt_checkout('main')"); err != nil {
		t.Fatalf("checkout main: %v", err)
	}

	cfg := &InstinctConfig{
		Dolt: DoltConfig{
			Refs:       "refs/dolt/myproject/",
			TeamBranch: "main",
			RemoteURL:  "git@github.com:org/repo.git",
		},
	}

	var pulledBranches []string
	capturePull := func(_ context.Context, _ *sql.Conn, _, branch string) error {
		pulledBranches = append(pulledBranches, branch)
		return nil
	}

	var buf strings.Builder
	if err := execPull(ctx, conn, cfg, "myuser", capturePull, &buf); err != nil {
		t.Fatalf("execPull: %v", err)
	}

	if len(pulledBranches) != 2 {
		t.Fatalf("expected 2 pull calls, got %d: %v", len(pulledBranches), pulledBranches)
	}
	if pulledBranches[0] != "main" {
		t.Errorf("first pull should be team branch main, got %q", pulledBranches[0])
	}
	if pulledBranches[1] != "myuser" {
		t.Errorf("second pull should be personal branch myuser, got %q", pulledBranches[1])
	}
}

// execPull完了後、個人ブランチに留まる
func TestExecPull_StaysOnPersonalBranch(t *testing.T) {
	ctx, conn := setupTestDB(t)

	if _, err := conn.ExecContext(ctx, "CALL dolt_checkout('-b', 'myuser')"); err != nil {
		t.Fatalf("create myuser branch: %v", err)
	}
	if _, err := conn.ExecContext(ctx, "CALL dolt_checkout('main')"); err != nil {
		t.Fatalf("checkout main: %v", err)
	}

	cfg := &InstinctConfig{
		Dolt: DoltConfig{
			TeamBranch: "main",
			RemoteURL:  "git@github.com:org/repo.git",
		},
	}
	var buf strings.Builder
	if err := execPull(ctx, conn, cfg, "myuser", nopPull, &buf); err != nil {
		t.Fatalf("execPull: %v", err)
	}

	var branch string
	if err := conn.QueryRowContext(ctx, "SELECT active_branch()").Scan(&branch); err != nil {
		t.Fatalf("active_branch: %v", err)
	}
	if branch != "myuser" {
		t.Errorf("expected to stay on myuser branch, got %q", branch)
	}
}
