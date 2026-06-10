package main

import (
	"strings"
	"testing"
)

// show <id> はMarkdown風セクション形式で全フィールドを出力する
func TestCLI_ShowCommand_MarkdownFormat(t *testing.T) {
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
	out := buf.String()

	// content が先頭に全文で出力される
	if !strings.HasPrefix(out, fullContent) {
		t.Errorf("expected content at top, got:\n%s", out)
	}
	// セクションヘッダが存在する
	for _, header := range []string{"[trigger]", "[meta]"} {
		if !strings.Contains(out, header) {
			t.Errorf("expected section %q in output, got:\n%s", header, out)
		}
	}
	// trigger と meta の値が含まれる
	for _, want := range []string{"git push時", "domain: git", "scope: project"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in output, got:\n%s", want, out)
		}
	}
}
