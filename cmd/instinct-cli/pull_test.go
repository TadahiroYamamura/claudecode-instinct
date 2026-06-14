package main

import (
	"context"
	"strings"
	"testing"

	doltrepo "github.com/TadahiroYamamura/claudecode-instinct/cmd/instinct-cli/internal/dolt"
)

// execPullはremote_urlが未設定のときエラーを返す
func TestExecPull_FailsWhenRemoteURLEmpty(t *testing.T) {
	var buf strings.Builder
	if err := execPull(context.Background(), &stubRepository{}, &InstinctConfig{}, "myuser", &buf); err == nil {
		t.Fatal("expected error when remote_url is empty, got nil")
	}
}

// execPullはteam_branchが未設定のときエラーを返す
func TestExecPull_FailsWhenTeamBranchEmpty(t *testing.T) {
	cfg := &InstinctConfig{Dolt: DoltConfig{RemoteURL: "git@github.com:org/repo.git"}}
	var buf strings.Builder
	if err := execPull(context.Background(), &stubRepository{}, cfg, "myuser", &buf); err == nil {
		t.Fatal("expected error when team_branch is empty, got nil")
	}
}

// execPullはlocalBranchが空のときエラーを返す
func TestExecPull_FailsWhenLocalBranchEmpty(t *testing.T) {
	cfg := &InstinctConfig{Dolt: DoltConfig{
		TeamBranch: "main",
		RemoteURL:  "git@github.com:org/repo.git",
	}}
	var buf strings.Builder
	if err := execPull(context.Background(), &stubRepository{}, cfg, "", &buf); err == nil {
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
	_ = execPull(ctx, doltrepo.NewRepository(conn), cfg, "myuser", &buf)

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

// execPullはチームブランチと個人ブランチの両方をこの順番でsyncする
func TestExecPull_SyncsBothBranchesInOrder(t *testing.T) {
	ctx, conn := setupTestDB(t)

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

	var syncedBranches []string
	repo := &stubRepository{
		sync: func(_ context.Context, _, branch string) error {
			syncedBranches = append(syncedBranches, branch)
			return nil
		},
	}

	var buf strings.Builder
	if err := execPull(ctx, repo, cfg, "myuser", &buf); err != nil {
		t.Fatalf("execPull: %v", err)
	}

	if len(syncedBranches) != 2 {
		t.Fatalf("expected 2 sync calls, got %d: %v", len(syncedBranches), syncedBranches)
	}
	if syncedBranches[0] != "main" {
		t.Errorf("first sync should be team branch main, got %q", syncedBranches[0])
	}
	if syncedBranches[1] != "myuser" {
		t.Errorf("second sync should be personal branch myuser, got %q", syncedBranches[1])
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
	repo := &stubRepository{
		checkout: func(ctx context.Context, branch string) error {
			_, err := conn.ExecContext(ctx, "CALL dolt_checkout(?)", branch)
			return err
		},
	}
	var buf strings.Builder
	if err := execPull(ctx, repo, cfg, "myuser", &buf); err != nil {
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
