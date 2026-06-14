package main

import (
	"context"
	"strings"
	"testing"

	doltrepo "github.com/TadahiroYamamura/claudecode-instinct/cmd/instinct-cli/internal/dolt"
)

// execPushはRepositoryのUploadを通じてリモートにpushする
func TestExecPush_UploadViaRepository(t *testing.T) {
	var gotRemote, gotBranch string
	repo := &stubRepository{
		upload: func(_ context.Context, remote, branch string) error {
			gotRemote, gotBranch = remote, branch
			return nil
		},
	}
	cfg := &InstinctConfig{Dolt: DoltConfig{
		Refs:      "refs/dolt/myproject/",
		RemoteURL: "git@github.com:org/repo.git",
	}}
	var buf strings.Builder
	if err := execPush(context.Background(), repo, cfg, "tadahiro", &buf); err != nil {
		t.Fatalf("execPush: %v", err)
	}
	if gotRemote != "origin" || gotBranch != "tadahiro" {
		t.Errorf("Upload called with remote=%q branch=%q", gotRemote, gotBranch)
	}
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
	_ = execPush(ctx, doltrepo.NewRepository(conn), cfg, "tadahiro", &buf)

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

// execPushはbranchが未設定のときエラーを返す（main へのフォールバック禁止）
func TestExecPush_FailsWhenBranchEmpty(t *testing.T) {
	cfg := &InstinctConfig{
		Dolt: DoltConfig{RemoteURL: "git@github.com:org/repo.git"},
	}
	var buf strings.Builder
	if err := execPush(context.Background(), &stubRepository{}, cfg, "", &buf); err == nil {
		t.Fatal("expected error when branch is empty, got nil")
	}
}

// execPushはremote_urlが未設定のときエラーを返す
func TestExecPush_FailsWhenRemoteURLEmpty(t *testing.T) {
	var buf strings.Builder
	err := execPush(context.Background(), &stubRepository{}, &InstinctConfig{}, "tadahiro", &buf)
	if err == nil {
		t.Fatal("expected error when remote_url is empty, got nil")
	}
}
