package main

import (
	"strings"
	"testing"
)

// show <id> は content を打ち切らず全文を出力する
func TestCLI_ShowCommand_PrintsFullContent(t *testing.T) {
	ctx, conn := setupTestDB(t)

	fullContent := strings.Repeat("あ", 41) // list では打ち切られる長さ
	id, err := insertInstinct(ctx, conn, InsertParams{
		Content:          fullContent,
		TriggerDesc:      "git push時",
		Domain:           "git",
		Scope:            "project",
		ObservationCount: 3,
		ProjectID:        "abc123def456",
	})
	if err != nil {
		t.Fatalf("insertInstinct: %v", err)
	}

	var buf strings.Builder
	if err := execShow(ctx, conn, id[:shortIDLen], &buf); err != nil {
		t.Fatalf("execShow: %v", err)
	}
	if !strings.Contains(buf.String(), fullContent) {
		t.Errorf("expected full content in output, got:\n%s", buf.String())
	}
}
