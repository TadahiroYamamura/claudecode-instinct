package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// サブコマンドなしのとき.instinct-db探索エラーではなく使用法エラーを返す
func TestDispatch_NoArgs_ReturnsUsageErrorNotProjectDirError(t *testing.T) {
	dir := t.TempDir() // .instinct-dbが存在しないディレクトリ

	err := dispatch([]string{}, dir)

	if err == nil {
		t.Fatal("expected error for no args")
	}
	if strings.Contains(err.Error(), ".instinct-db") {
		t.Errorf("should not search for .instinct-db when no subcommand given, got: %v", err)
	}
}

// dispatch(["setup"], dir)が.instinct-db/data/を作成する
func TestCLI_SetupCommand_CreatesInstinctDb(t *testing.T) {
	dir := t.TempDir()

	if err := dispatch([]string{"setup"}, dir); err != nil {
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
