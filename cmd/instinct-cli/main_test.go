package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// tabwriterによる整形後は生のタブ文字が出力に残らない
func TestCLI_ListCommand_AlignsColumns(t *testing.T) {
	ctx, conn := setupTestDB(t)

	for _, content := range []string{"short", "this is much longer content"} {
		if _, err := insertInstinct(ctx, conn, InsertParams{
			Content:          content,
			TriggerDesc:      "trigger",
			Domain:           "test",
			Scope:            "project",
			ObservationCount: 1,
			ProjectID:        "abc123def456",
		}); err != nil {
			t.Fatalf("insertInstinct: %v", err)
		}
	}

	var buf strings.Builder
	if err := execList(ctx, conn, &buf); err != nil {
		t.Fatalf("execList: %v", err)
	}
	if strings.Contains(buf.String(), "\t") {
		t.Errorf("expected tabwriter to replace tabs with spaces, got:\n%s", buf.String())
	}
}

// 41文字超のcontentは40文字で打ち切られ "..." が付く
func TestCLI_ListCommand_TruncatesLongContent(t *testing.T) {
	ctx, conn := setupTestDB(t)

	longContent := strings.Repeat("あ", 41) // 41文字

	if _, err := insertInstinct(ctx, conn, InsertParams{
		Content:          longContent,
		TriggerDesc:      "trigger",
		Domain:           "test",
		Scope:            "project",
		ObservationCount: 1,
		ProjectID:        "abc123def456",
	}); err != nil {
		t.Fatalf("insertInstinct: %v", err)
	}

	var buf strings.Builder
	if err := execList(ctx, conn, &buf); err != nil {
		t.Fatalf("execList: %v", err)
	}
	if strings.Contains(buf.String(), longContent) {
		t.Error("expected content to be truncated, but full content appeared")
	}
	if !strings.Contains(buf.String(), "...") {
		t.Error("expected truncation marker '...'")
	}
}

// サブコマンドなしのとき.instinct-db探索エラーではなく使用法エラーを返す
func TestDispatch_NoArgs_ReturnsUsageErrorNotProjectDirError(t *testing.T) {
	dir := t.TempDir() // .instinct-dbが存在しないディレクトリ

	err := dispatch([]string{}, dir, strings.NewReader(""), io.Discard)

	if err == nil {
		t.Fatal("expected error for no args")
	}
	if strings.Contains(err.Error(), ".instinct-db") {
		t.Errorf("should not search for .instinct-db when no subcommand given, got: %v", err)
	}
}

// dispatchはdedupサブコマンドをexecDedupにルーティングする（instinctが0件なのでjudgeは呼ばれない）
func TestDispatch_DedupCommand_ZeroPairsWhenEmpty(t *testing.T) {
	dir := t.TempDir()
	gitInitWithRemote(t, dir)
	if err := execSetup(dir, true, nil, io.Discard, fakeCloneFail, fakePush); err != nil {
		t.Fatalf("setup: %v", err)
	}

	var buf strings.Builder
	if err := dispatch([]string{"dedup"}, dir, nil, &buf); err != nil {
		t.Fatalf("dedup: %v", err)
	}
	if !strings.Contains(buf.String(), "0") {
		t.Errorf("expected '0 pairs' in output, got: %q", buf.String())
	}
}

// dispatch(["setup"], dir)が.instinct-db/data/を作成する（initパス）
func TestCLI_SetupCommand_CreatesInstinctDb(t *testing.T) {
	dir := t.TempDir()
	gitInitWithRemote(t, dir)

	if err := execSetup(dir, true, nil, io.Discard, fakeCloneFail, fakePush); err != nil {
		t.Fatalf("dispatch: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, ".instinct-db", "data")); os.IsNotExist(err) {
		t.Error(".instinct-db/data/ was not created")
	}
}

// execInsert(insertFlags)がDBにレコードを保存する
func TestCLI_InsertCommand_StoresRecord(t *testing.T) {
	ctx, conn := setupTestDB(t)

	err := execInsert(ctx, conn, insertFlags{
		Content: "テスト前に仕様を確認する",
		Trigger: "テスト実行時",
		Domain:  "testing",
		Count:   2,
		Scope:   "project",
	}, func(string) (string, error) { return "abc123def456", nil })
	if err != nil {
		t.Fatalf("execInsert: %v", err)
	}

	var count int
	if err := conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM instincts").Scan(&count); err != nil {
		t.Fatalf("query: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 record, got %d", count)
	}
}
