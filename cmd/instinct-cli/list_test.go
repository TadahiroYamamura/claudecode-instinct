package main

import (
	"testing"
)

func TestListInstincts_ReturnsInsertedRecord(t *testing.T) {
	ctx, conn := setupTestDB(t)

	if _, err := insertInstinct(ctx, conn, InsertParams{
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

func TestListInstincts_OrderedNewestFirst(t *testing.T) {
	ctx, conn := setupTestDB(t)

	for _, row := range []struct {
		id        string
		content   string
		createdAt string
	}{
		{"id-old", "古い知見", "2026-01-01 00:00:00"},
		{"id-new", "新しい知見", "2026-06-01 00:00:00"},
	} {
		_, err := conn.ExecContext(ctx,
			`INSERT INTO instincts (id, content, trigger_desc, domain, scope, observation_count, project_id, created_at)
			 VALUES (?, ?, '', '', 'project', 1, 'abc123def456', ?)`,
			row.id, row.content, row.createdAt)
		if err != nil {
			t.Fatalf("insert %s: %v", row.id, err)
		}
	}

	rows, err := listInstincts(ctx, conn)
	if err != nil {
		t.Fatalf("listInstincts: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	if rows[0].Content != "新しい知見" {
		t.Errorf("expected newest first, got %q", rows[0].Content)
	}
}

func TestListInstincts_ReturnsAllFields(t *testing.T) {
	ctx, conn := setupTestDB(t)

	if _, err := insertInstinct(ctx, conn, InsertParams{
		Content:          "コミット前にlintを実行する",
		TriggerDesc:      "git commit時",
		Domain:           "git",
		Scope:            "global",
		ObservationCount: 7,
		ProjectID:        "abc123def456",
	}); err != nil {
		t.Fatalf("insertInstinct: %v", err)
	}

	rows, err := listInstincts(ctx, conn)
	if err != nil {
		t.Fatalf("listInstincts: %v", err)
	}
	r := rows[0]
	if r.TriggerDesc != "git commit時" {
		t.Errorf("trigger_desc = %q", r.TriggerDesc)
	}
	if r.Domain != "git" {
		t.Errorf("domain = %q", r.Domain)
	}
	if r.ObservationCount != 7 {
		t.Errorf("observation_count = %d", r.ObservationCount)
	}
	if r.Scope != "global" {
		t.Errorf("scope = %q", r.Scope)
	}
	if r.CreatedAt.IsZero() {
		t.Error("created_at is zero")
	}
}
