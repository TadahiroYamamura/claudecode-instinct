package main

import (
	"context"
	"io"
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

// insertはパーソナルブランチにレコードを追加し、チームブランチ(main)には追加しない
func TestDispatch_Insert_AddsRecordOnPersonalBranchOnly(t *testing.T) {
	dir := t.TempDir()
	gitInitWithRemote(t, dir)
	if err := execInit(dir, initParams{Branch: "alice", Yes: true}, nil, io.Discard, doltRepoFn); err != nil {
		t.Fatalf("execInit: %v", err)
	}

	if err := dispatch([]string{"insert",
		"--content", "テスト前に仕様を確認",
		"--trigger", "実装前",
		"--domain", "testing",
		"--count", "1",
	}, dir, nil, io.Discard); err != nil {
		t.Fatalf("dispatch insert: %v", err)
	}

	conn, cleanup, err := openConn(t.Context(), instinctDataDir(dir))
	if err != nil {
		t.Fatalf("openConn: %v", err)
	}
	defer cleanup()

	var aliceCount int
	if _, err := conn.ExecContext(t.Context(), "CALL dolt_checkout('alice')"); err != nil {
		t.Fatalf("checkout alice: %v", err)
	}
	if err := conn.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM instincts").Scan(&aliceCount); err != nil {
		t.Fatalf("count on alice: %v", err)
	}
	if aliceCount != 1 {
		t.Errorf("expected 1 record on alice branch, got %d", aliceCount)
	}

	var mainCount int
	if _, err := conn.ExecContext(t.Context(), "CALL dolt_checkout('main')"); err != nil {
		t.Fatalf("checkout main: %v", err)
	}
	if err := conn.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM instincts").Scan(&mainCount); err != nil {
		t.Fatalf("count on main: %v", err)
	}
	if mainCount != 0 {
		t.Errorf("expected 0 records on main branch, got %d", mainCount)
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

