package main

import (
	"context"
	"strings"
	"testing"
)

// execDedupはinstinctが0件のとき0ペアをチェックしたと報告する
func TestExecDedup_EmptyInstinctsReportsZeroPairs(t *testing.T) {
	ctx, conn := setupTestDB(t)

	var buf strings.Builder
	judge := func(_ context.Context, _, _ InstinctRow) (DedupDecision, error) {
		return DedupDecision{}, nil
	}
	if err := execDedup(ctx, conn, judge, &buf); err != nil {
		t.Fatalf("execDedup: %v", err)
	}
	if !strings.Contains(buf.String(), "0") {
		t.Errorf("expected output to mention 0, got: %q", buf.String())
	}
}

// execDedupはduplicateと判定されたペアをdedup_decisionsに記録する
func TestExecDedup_DuplicateDecisionIsRecorded(t *testing.T) {
	ctx, conn := setupTestDB(t)

	// 意味的に同一だが表現が異なるinstinctを2件挿入
	for _, params := range []InsertParams{
		{Content: "テスト前にlintを通す", TriggerDesc: "テスト実行時", Domain: "testing", Scope: "project", ObservationCount: 3, ProjectID: "abc"},
		{Content: "lintエラーを解消してからテストを走らせる", TriggerDesc: "テスト実行時", Domain: "testing", Scope: "project", ObservationCount: 2, ProjectID: "abc"},
	} {
		if _, err := insertInstinct(ctx, conn, params); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}

	judge := func(_ context.Context, _, _ InstinctRow) (DedupDecision, error) {
		return DedupDecision{Decision: "duplicate", Reasoning: "同じ知見の言い換え", Similarity: 0.85}, nil
	}

	var buf strings.Builder
	if err := execDedup(ctx, conn, judge, &buf); err != nil {
		t.Fatalf("execDedup: %v", err)
	}

	var count int
	if err := conn.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM dedup_decisions WHERE decision = 'duplicate'",
	).Scan(&count); err != nil {
		t.Fatalf("query dedup_decisions: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 duplicate decision in dedup_decisions, got %d", count)
	}
}
