package main

import (
	"testing"

	doltrepo "github.com/TadahiroYamamura/claudecode-instinct/cmd/instinct-cli/internal/dolt"
)

// working setが空のときcommitはエラーを返さず成功する（observer-loop.shが冪等に呼べる）
func TestExecCommit_NothingToCommit_SucceedsSilently(t *testing.T) {
	ctx, conn := setupTestDB(t)
	repo := doltRepoFn(conn)

	// DDLをコミットしてworking setをクリーンにする
	if err := execCommit(ctx, repo, "init"); err != nil {
		t.Fatalf("initial commit: %v", err)
	}

	// working setが空 → エラーにならない
	if err := execCommit(ctx, repo, "should be no-op"); err != nil {
		t.Errorf("expected no error when nothing to commit, got: %v", err)
	}
}

// execCommitはカスタムメッセージをdolt_logに記録する
func TestExecCommit_StoresCustomMessage(t *testing.T) {
	ctx, conn := setupTestDB(t)

	if err := execCommit(ctx, doltRepoFn(conn), "my custom message"); err != nil {
		t.Fatalf("execCommit: %v", err)
	}

	var msg string
	if err := conn.QueryRowContext(ctx, "SELECT message FROM dolt_log ORDER BY date DESC LIMIT 1").Scan(&msg); err != nil {
		t.Fatalf("query dolt_log: %v", err)
	}
	if msg != "my custom message" {
		t.Errorf("expected message 'my custom message', got %q", msg)
	}
}

// execCommitはRepositoryを通じてworking setをdoltコミットとして記録する
func TestExecCommit_CreatesDoltCommitViaRepository(t *testing.T) {
	ctx, conn := setupTestDB(t)

	if _, err := insertInstinct(ctx, conn, InsertParams{
		Content: "テスト前にlintを通す", TriggerDesc: "テスト実行時",
		Domain: "testing", Scope: "project", ObservationCount: 1, ProjectID: "abc",
	}); err != nil {
		t.Fatalf("insert: %v", err)
	}

	var before int
	if err := conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM dolt_log").Scan(&before); err != nil {
		t.Fatalf("dolt_log before: %v", err)
	}

	if err := execCommit(ctx, doltrepo.NewRepository(conn), "observer: 1 instinct"); err != nil {
		t.Fatalf("execCommit: %v", err)
	}

	var after int
	if err := conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM dolt_log").Scan(&after); err != nil {
		t.Fatalf("dolt_log after: %v", err)
	}
	if after != before+1 {
		t.Errorf("expected dolt_log to grow by 1, before=%d after=%d", before, after)
	}
}

