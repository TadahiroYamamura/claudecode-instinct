package main

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
)

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

func TestInsert_StoresInstinct(t *testing.T) {
	ctx, conn := setupTestDB(t)

	err := insertInstinct(ctx, conn, InsertParams{
		Content:          "テスト実行前に仕様を確認する",
		TriggerDesc:      "テスト実行時",
		Domain:           "testing",
		Scope:            "project",
		ObservationCount: 5,
		ProjectID:        "abc123def456",
	})
	if err != nil {
		t.Fatalf("insertInstinct: %v", err)
	}

	var count int
	err = conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM instincts WHERE content = ?",
		"テスト実行前に仕様を確認する").Scan(&count)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 record, got %d", count)
	}
}
