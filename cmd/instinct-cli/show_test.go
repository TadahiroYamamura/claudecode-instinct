package main

import (
	"context"
	_ "embed"
	"strings"
	"testing"

	doltrepo "github.com/TadahiroYamamura/claudecode-instinct/cmd/instinct-cli/internal/dolt"
)

//go:embed testdata/show_golden.txt
var showGolden string

const fixedID = "aa000000-0000-0000-0000-000000000000"
const fixedContent = "ああああああああああああああああああああああああああああああああああああああああああ" // 41文字

// execShowはRepositoryからinstinctを取得して全フィールドを出力する
func TestExecShow_OutputsInstinctFromRepository(t *testing.T) {
	repo := &stubRepository{
		getInstinct: func(_ context.Context, shortID string) (*InstinctRow, error) {
			return &InstinctRow{
				ID:               "aa000000-0000-0000-0000-000000000000",
				Content:          "テスト前に仕様を確認",
				TriggerDesc:      "テスト実行時",
				Domain:           "testing",
				ObservationCount: 3,
				Scope:            "project",
			}, nil
		},
	}
	var buf strings.Builder
	if err := execShow(context.Background(), repo, "aa000000", &buf); err != nil {
		t.Fatalf("execShow: %v", err)
	}
	if !strings.Contains(buf.String(), "テスト前に仕様を確認") {
		t.Errorf("expected content in output, got: %s", buf.String())
	}
}

// 存在しないIDを指定したときexecShowはエラーを返す
func TestExecShow_ReturnsErrorForUnknownID(t *testing.T) {
	ctx, conn := setupTestDB(t)
	var buf strings.Builder
	err := execShow(ctx, doltRepoFn(conn), "nonexistent", &buf)
	if err == nil {
		t.Error("expected error for unknown ID")
	}
}

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
	if err := execShow(ctx, doltrepo.NewRepository(conn), fixedID[:shortIDLen], &buf); err != nil {
		t.Fatalf("execShow: %v", err)
	}

	if buf.String() != showGolden {
		t.Errorf("output mismatch:\n--- got ---\n%s\n--- want ---\n%s", buf.String(), showGolden)
	}
}
