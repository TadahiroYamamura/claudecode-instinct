package main

import (
	"testing"
)

func TestListInstincts_ReturnsInsertedRecord(t *testing.T) {
	ctx, conn := setupTestDB(t)

	if err := insertInstinct(ctx, conn, InsertParams{
		Content:          "テスト前に仕様を確認する",
		TriggerDesc:      "テスト実行時",
		Domain:           "testing",
		Scope:            "project",
		ObservationCount: 5,
		ProjectID:        "abc123def456",
	}); err != nil {
		t.Fatalf("insertInstinct: %v", err)
	}

	rows, err := listInstincts(ctx, conn)
	if err != nil {
		t.Fatalf("listInstincts: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0].Content != "テスト前に仕様を確認する" {
		t.Errorf("content = %q", rows[0].Content)
	}
}

func TestListInstincts_ReturnsTriggerDesc(t *testing.T) {
	ctx, conn := setupTestDB(t)

	if err := insertInstinct(ctx, conn, InsertParams{
		Content:          "コミット前にlintを実行する",
		TriggerDesc:      "git commit時",
		Domain:           "git",
		Scope:            "project",
		ObservationCount: 3,
		ProjectID:        "abc123def456",
	}); err != nil {
		t.Fatalf("insertInstinct: %v", err)
	}

	rows, err := listInstincts(ctx, conn)
	if err != nil {
		t.Fatalf("listInstincts: %v", err)
	}
	if rows[0].TriggerDesc != "git commit時" {
		t.Errorf("trigger_desc = %q", rows[0].TriggerDesc)
	}
}
