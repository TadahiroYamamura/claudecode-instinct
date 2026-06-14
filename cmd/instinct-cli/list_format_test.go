package main

import (
	_ "embed"
	"flag"
	"os"
	"strings"
	"testing"

	doltrepo "github.com/TadahiroYamamura/claudecode-instinct/cmd/instinct-cli/internal/dolt"
)

//go:embed testdata/list_golden.txt
var listGolden string

var update = flag.Bool("update", false, "update golden files")

// list出力のフォーマットをゴールデンファイルで検証する
// （ID・全フィールド・短縮ID・新しい順すべてをカバー）
func TestCLI_ListCommand_Format(t *testing.T) {
	ctx, conn := setupTestDB(t)

	// 固定UUIDと明示的なcreated_atで動的な値を排除する
	for _, row := range []struct {
		id        string
		content   string
		trigger   string
		domain    string
		scope     string
		obs       int
		createdAt string
	}{
		{"bb000000-0000-0000-0000-000000000000", "run tests before pushing", "on git push", "git", "global", 7, "2026-06-10 10:00:00"},
		{"aa000000-0000-0000-0000-000000000000", "check spec before writing test", "on test run", "testing", "project", 5, "2026-06-01 10:00:00"},
	} {
		_, err := conn.ExecContext(ctx,
			`INSERT INTO instincts (id, content, trigger_desc, domain, scope, observation_count, project_id, created_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			row.id, row.content, row.trigger, row.domain, row.scope, row.obs, "abc123def456", row.createdAt,
		)
		if err != nil {
			t.Fatalf("insert %s: %v", row.id, err)
		}
	}

	var buf strings.Builder
	if err := execList(ctx, doltrepo.NewRepository(conn), &buf); err != nil {
		t.Fatalf("execList: %v", err)
	}

	if *update {
		if err := os.WriteFile("testdata/list_golden.txt", []byte(buf.String()), 0644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		return
	}

	if buf.String() != listGolden {
		t.Errorf("output mismatch:\n--- got ---\n%s\n--- want ---\n%s", buf.String(), listGolden)
	}
}
