package main

import (
	"context"
	"strings"
	"testing"

	doltrepo "github.com/TadahiroYamamura/claudecode-instinct/cmd/instinct-cli/internal/dolt"
)

// execListはRepositoryからinstinctを取得して出力する
func TestExecList_OutputsInstinctsFromRepository(t *testing.T) {
	repo := &stubRepository{
		listInstincts: func(_ context.Context) ([]InstinctRow, error) {
			return []InstinctRow{
				{ID: "aaaa0000-0000-0000-0000-000000000000", Content: "テスト前に仕様を確認", TriggerDesc: "テスト時", Domain: "testing", ObservationCount: 3, Scope: "project"},
			}, nil
		},
	}
	var buf strings.Builder
	if err := execList(context.Background(), repo, &buf); err != nil {
		t.Fatalf("execList: %v", err)
	}
	if !strings.Contains(buf.String(), "テスト前に仕様を確認") {
		t.Errorf("expected content in output, got: %s", buf.String())
	}
}

// tabwriterによる整形後は生のタブ文字が出力に残らない
func TestExecList_AlignsColumns(t *testing.T) {
	ctx, conn := setupTestDB(t)

	for _, content := range []string{"short", "this is much longer content"} {
		if _, err := insertInstinct(ctx, conn, InsertParams{
			Content: content, TriggerDesc: "trigger", Domain: "test",
			Scope: "project", ObservationCount: 1, ProjectID: "abc123def456",
		}); err != nil {
			t.Fatalf("insertInstinct: %v", err)
		}
	}

	var buf strings.Builder
	if err := execList(ctx, doltrepo.NewRepository(conn), &buf); err != nil {
		t.Fatalf("execList: %v", err)
	}
	if strings.Contains(buf.String(), "\t") {
		t.Errorf("expected tabwriter to replace tabs with spaces, got:\n%s", buf.String())
	}
}

// 41文字超のcontentは40文字で打ち切られ "..." が付く
func TestExecList_TruncatesLongContent(t *testing.T) {
	ctx, conn := setupTestDB(t)

	longContent := strings.Repeat("あ", 41)

	if _, err := insertInstinct(ctx, conn, InsertParams{
		Content: longContent, TriggerDesc: "trigger", Domain: "test",
		Scope: "project", ObservationCount: 1, ProjectID: "abc123def456",
	}); err != nil {
		t.Fatalf("insertInstinct: %v", err)
	}

	var buf strings.Builder
	if err := execList(ctx, doltrepo.NewRepository(conn), &buf); err != nil {
		t.Fatalf("execList: %v", err)
	}
	if strings.Contains(buf.String(), longContent) {
		t.Error("expected content to be truncated, but full content appeared")
	}
	if !strings.Contains(buf.String(), "...") {
		t.Error("expected truncation marker '...'")
	}
}

