package main

import (
	"testing"
)

// run(["insert", ...])がDBにレコードを保存する
func TestCLI_InsertCommand_StoresRecord(t *testing.T) {
	ctx, conn := setupTestDB(t)

	err := run([]string{
		"insert",
		"--content", "テスト前に仕様を確認する",
		"--trigger", "テスト実行時",
		"--domain", "testing",
		"--count", "2",
	}, conn, func(string) (string, error) { return "abc123def456", nil })
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	var count int
	if err := conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM instincts").Scan(&count); err != nil {
		t.Fatalf("query: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 record, got %d", count)
	}
}
