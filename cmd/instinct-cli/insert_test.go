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

// 同一内容を2回insertすると2レコードになる（dedup前）
// observation_countの合算はdedup時に行われる
func TestRunInsert_StoresRecordFromFlags(t *testing.T) {
	ctx, conn := setupTestDB(t)

	err := runInsert(ctx, conn, []string{
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

func TestInsert_SameContentTwiceCreatesTwoRecords(t *testing.T) {
	ctx, conn := setupTestDB(t)

	params := InsertParams{
		Content:          "git push前にテストを実行する",
		TriggerDesc:      "git push時",
		Domain:           "git",
		Scope:            "global",
		ObservationCount: 2,
		ProjectID:        "abc123def456",
	}
	if err := insertInstinct(ctx, conn, params); err != nil {
		t.Fatalf("first insert: %v", err)
	}
	params.ObservationCount = 1
	if err := insertInstinct(ctx, conn, params); err != nil {
		t.Fatalf("second insert: %v", err)
	}

	var totalCount, totalObs int
	err := conn.QueryRowContext(ctx,
		"SELECT COUNT(*), SUM(observation_count) FROM instincts WHERE content = ?",
		"git push前にテストを実行する").Scan(&totalCount, &totalObs)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if totalCount != 2 {
		t.Errorf("expected 2 records before dedup, got %d", totalCount)
	}
	if totalObs != 3 {
		t.Errorf("expected observation_count sum = 3, got %d", totalObs)
	}
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
