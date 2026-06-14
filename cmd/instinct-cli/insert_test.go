package main

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	doltrepo "github.com/TadahiroYamamura/claudecode-instinct/cmd/instinct-cli/internal/dolt"
)

// execInsertはRepositoryを通じてinstinctを保存する
func TestExecInsert_StoresInstinctViaRepository(t *testing.T) {
	var got InsertParams
	repo := &stubRepository{
		insertInstinct: func(_ context.Context, p InsertParams) (string, error) {
			got = p
			return "id", nil
		},
	}
	err := execInsert(context.Background(), repo, insertFlags{
		Content: "テスト前に仕様を確認する",
		Trigger: "テスト実行時",
		Domain:  "testing",
		Count:   2,
		Scope:   "project",
	}, func(string) (string, error) { return "abc123", nil })
	if err != nil {
		t.Fatalf("execInsert: %v", err)
	}
	if got.Content != "テスト前に仕様を確認する" {
		t.Errorf("content: got %q, want %q", got.Content, "テスト前に仕様を確認する")
	}
}

func setupTestDB(t *testing.T) (context.Context, *sql.Conn) {
	t.Helper()
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "data")
	ctx := context.Background()

	if err := setupDB(ctx, dataDir); err != nil {
		t.Fatalf("setupDB: %v", err)
	}
	conn, cleanup, err := openConn(ctx, dataDir)
	if err != nil {
		t.Fatalf("openConn: %v", err)
	}
	t.Cleanup(cleanup)
	return ctx, conn
}

// 同一内容を2回insertすると2レコードになる（dedup前）
// observation_countの合算はdedup時に行われる
func TestRunInsert_StoresRecordFromFlags(t *testing.T) {
	ctx, conn := setupTestDB(t)

	err := runInsert(ctx, doltrepo.NewRepository(conn), []string{
		"--content", "git push前にテストを実行する",
		"--trigger", "git push時",
		"--domain", "git",
		"--count", "3",
		"--scope", "global",
	}, func(string) (string, error) { return "abc123def456", nil })
	if err != nil {
		t.Fatalf("runInsert: %v", err)
	}

	var content, scope string
	var obsCount int
	err = conn.QueryRowContext(ctx,
		"SELECT content, scope, observation_count FROM instincts LIMIT 1",
	).Scan(&content, &scope, &obsCount)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if content != "git push前にテストを実行する" {
		t.Errorf("content = %q", content)
	}
	if scope != "global" {
		t.Errorf("scope = %q", scope)
	}
	if obsCount != 3 {
		t.Errorf("observation_count = %d", obsCount)
	}
}

func TestRunInsert_CountIsRequired(t *testing.T) {
	ctx, conn := setupTestDB(t)

	err := runInsert(ctx, doltrepo.NewRepository(conn), []string{
		"--content", "何か知見",
		"--trigger", "何かのとき",
	}, func(string) (string, error) { return "abc123def456", nil })

	if err == nil {
		t.Fatal("expected error when --count is omitted")
	}
}

