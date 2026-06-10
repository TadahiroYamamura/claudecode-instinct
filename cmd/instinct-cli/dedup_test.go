package main

import (
	"context"
	"strings"
	"testing"
)

// execDedupは類似度が閾値未満のペアをHaikuに送らない
func TestExecDedup_SkipsPairsBelowThreshold(t *testing.T) {
	ctx, conn := setupTestDB(t)

	for _, params := range []InsertParams{
		{Content: "テスト前にlintを通す", TriggerDesc: "テスト実行時", Domain: "testing", Scope: "project", ObservationCount: 3, ProjectID: "abc"},
		{Content: "lintエラーを解消してからテストを走らせる", TriggerDesc: "テスト実行時", Domain: "testing", Scope: "project", ObservationCount: 2, ProjectID: "abc"},
	} {
		if _, err := insertInstinct(ctx, conn, params); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}

	callCount := 0
	judge := func(_ context.Context, _, _ InstinctRow) (DedupDecision, error) {
		callCount++
		return DedupDecision{Decision: decisionDistinct}, nil
	}

	var buf strings.Builder
	// threshold=1.0 なら完全一致以外はすべてスキップ
	if err := execDedup(ctx, conn, judge, bigramSimilarity, 1.0, &buf); err != nil {
		t.Fatalf("execDedup: %v", err)
	}
	if callCount != 0 {
		t.Errorf("expected judge not called, got %d calls", callCount)
	}
}

// execDedupはinstinctが0件のとき0ペアをチェックしたと報告する
func TestExecDedup_EmptyInstinctsReportsZeroPairs(t *testing.T) {
	ctx, conn := setupTestDB(t)

	var buf strings.Builder
	judge := func(_ context.Context, _, _ InstinctRow) (DedupDecision, error) {
		return DedupDecision{}, nil
	}
	if err := execDedup(ctx, conn, judge, bigramSimilarity, 0.0, &buf); err != nil {
		t.Fatalf("execDedup: %v", err)
	}
	if !strings.Contains(buf.String(), "0") {
		t.Errorf("expected output to mention 0, got: %q", buf.String())
	}
}

// execDedupはduplicateと判定されたinstinctをマージして重複を1件に削除する
func TestExecDedup_DuplicateMergesObservationCountAndDeletesOne(t *testing.T) {
	ctx, conn := setupTestDB(t)

	// 表現は異なるが意味的に同一なinstinctを2件挿入
	if _, err := insertInstinct(ctx, conn, InsertParams{
		Content: "テスト前にlintを通す", TriggerDesc: "テスト実行時",
		Domain: "testing", Scope: "project", ObservationCount: 3, ProjectID: "abc",
	}); err != nil {
		t.Fatalf("insert A: %v", err)
	}
	if _, err := insertInstinct(ctx, conn, InsertParams{
		Content: "lintエラーを解消してからテストを走らせる", TriggerDesc: "テスト実行時",
		Domain: "testing", Scope: "project", ObservationCount: 2, ProjectID: "abc",
	}); err != nil {
		t.Fatalf("insert B: %v", err)
	}

	judge := func(_ context.Context, _, _ InstinctRow) (DedupDecision, error) {
		return DedupDecision{Decision: decisionDuplicate, Reasoning: "同じ知見の言い換え", Similarity: 0.85}, nil
	}

	var buf strings.Builder
	if err := execDedup(ctx, conn, judge, bigramSimilarity, 0.0, &buf); err != nil {
		t.Fatalf("execDedup: %v", err)
	}

	var remaining int
	if err := conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM instincts").Scan(&remaining); err != nil {
		t.Fatalf("count instincts: %v", err)
	}
	if remaining != 1 {
		t.Errorf("expected 1 instinct after dedup, got %d", remaining)
	}

	var obsCount int
	if err := conn.QueryRowContext(ctx,
		"SELECT observation_count FROM instincts",
	).Scan(&obsCount); err != nil {
		t.Fatalf("query observation_count: %v", err)
	}
	if obsCount != 5 {
		t.Errorf("expected merged observation_count=5 (3+2), got %d", obsCount)
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
		return DedupDecision{Decision: decisionDuplicate, Reasoning: "同じ知見の言い換え", Similarity: 0.85}, nil
	}

	var buf strings.Builder
	if err := execDedup(ctx, conn, judge, bigramSimilarity, 0.0, &buf); err != nil {
		t.Fatalf("execDedup: %v", err)
	}

	var count int
	if err := conn.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM dedup_decisions WHERE decision = ?", decisionDuplicate,
	).Scan(&count); err != nil {
		t.Fatalf("query dedup_decisions: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 duplicate decision in dedup_decisions, got %d", count)
	}
}
