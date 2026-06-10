package main

import (
	_ "embed"
	"strings"
	"testing"
)

//go:embed testdata/show_golden.txt
var showGolden string

const fixedID = "aa000000-0000-0000-0000-000000000000"
const fixedContent = "ああああああああああああああああああああああああああああああああああああああああああ" // 41文字

// show <id> はMarkdown風セクション形式で全フィールドを出力する
func TestCLI_ShowCommand_MarkdownFormat(t *testing.T) {
	ctx, conn := setupTestDB(t)

	_, err := conn.ExecContext(ctx,
		`INSERT INTO instincts (id, content, trigger_desc, domain, scope, observation_count, project_id)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		fixedID, fixedContent, "git push時", "git", "project", 3, "abc123def456",
	)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	var buf strings.Builder
	if err := execShow(ctx, conn, fixedID[:shortIDLen], &buf); err != nil {
		t.Fatalf("execShow: %v", err)
	}

	if buf.String() != showGolden {
		t.Errorf("output mismatch:\n--- got ---\n%s\n--- want ---\n%s", buf.String(), showGolden)
	}
}
