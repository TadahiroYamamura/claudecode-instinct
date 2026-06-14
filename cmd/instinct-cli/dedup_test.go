package main

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	doltrepo "github.com/TadahiroYamamura/claudecode-instinct/cmd/instinct-cli/internal/dolt"
)

// InsertDedupDecisionはdedup_decisionsテーブルにレコードを挿入する
func TestRepository_InsertDedupDecision(t *testing.T) {
	ctx, conn := setupTestDB(t)
	a := InstinctRow{ID: "aaaa0000-0000-0000-0000-000000000000", Content: "知見A", TriggerDesc: "trigger"}
	b := InstinctRow{ID: "bbbb0000-0000-0000-0000-000000000000", Content: "知見B", TriggerDesc: "trigger"}
	scores := SimilarityScores{Bigram: 0.8, Trigram: 0.7, Overlap: 0.9}
	d := DedupDecision{Decision: decisionDistinct, Reasoning: "similar but distinct"}

	repo := doltrepo.NewRepository(conn)
	if err := repo.InsertDedupDecision(ctx, a, b, d, scores); err != nil {
		t.Fatalf("InsertDedupDecision: %v", err)
	}
	var count int
	if err := conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM dedup_decisions").Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 record, got %d", count)
	}
}

// MergeAndDeleteはloserの観察数をwinnerに加算してloserを削除する
func TestRepository_MergeAndDelete(t *testing.T) {
	ctx, conn := setupTestDB(t)
	winnerID, _ := insertInstinct(ctx, conn, InsertParams{Content: "winner", TriggerDesc: "t", Domain: "d", Scope: "project", ObservationCount: 3, ProjectID: "abc"})
	loserID, _ := insertInstinct(ctx, conn, InsertParams{Content: "loser", TriggerDesc: "t", Domain: "d", Scope: "project", ObservationCount: 2, ProjectID: "abc"})

	winner := InstinctRow{ID: winnerID, ObservationCount: 3}
	loser := InstinctRow{ID: loserID, ObservationCount: 2}

	repo := doltrepo.NewRepository(conn)
	if err := repo.MergeAndDelete(ctx, winner, loser); err != nil {
		t.Fatalf("MergeAndDelete: %v", err)
	}

	var obsCount int
	if err := conn.QueryRowContext(ctx, "SELECT observation_count FROM instincts WHERE id = ?", winnerID).Scan(&obsCount); err != nil {
		t.Fatalf("query: %v", err)
	}
	if obsCount != 5 {
		t.Errorf("expected observation_count=5, got %d", obsCount)
	}
	var remaining int
	if err := conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM instincts WHERE id = ?", loserID).Scan(&remaining); err != nil {
		t.Fatalf("query: %v", err)
	}
	if remaining != 0 {
		t.Error("expected loser to be deleted")
	}
}

func insertLintInstincts(t *testing.T, ctx context.Context, conn *sql.Conn) {
	t.Helper()
	for _, params := range []InsertParams{
		{Content: "テスト前にlintを通す", TriggerDesc: "テスト実行時", Domain: "testing", Scope: "project", ObservationCount: 3, ProjectID: "abc"},
		{Content: "lintエラーを解消してからテストを走らせる", TriggerDesc: "テスト実行時", Domain: "testing", Scope: "project", ObservationCount: 2, ProjectID: "abc"},
	} {
		if _, err := insertInstinct(ctx, conn, params); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}
}

// execDedupは類似度が閾値未満のペアをHaikuに送らない
func TestExecDedup_SkipsPairsBelowThreshold(t *testing.T) {
	ctx, conn := setupTestDB(t)
	insertLintInstincts(t, ctx, conn)

	callCount := 0
	judge := func(_ context.Context, _, _ InstinctRow) (DedupDecision, error) {
		callCount++
		return DedupDecision{Decision: decisionDistinct}, nil
	}

	var buf strings.Builder
	// threshold=1.0 なら完全一致以外はすべてスキップ
	if err := execDedup(ctx, doltrepo.NewRepository(conn), judge, 1.0, &buf); err != nil {
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
	if err := execDedup(ctx, doltrepo.NewRepository(conn), judge, 0.0, &buf); err != nil {
		t.Fatalf("execDedup: %v", err)
	}
	if !strings.Contains(buf.String(), "0") {
		t.Errorf("expected output to mention 0, got: %q", buf.String())
	}
}

// execDedupはduplicateと判定されたinstinctをマージして重複を1件に削除する
func TestExecDedup_DuplicateMergesObservationCountAndDeletesOne(t *testing.T) {
	ctx, conn := setupTestDB(t)
	insertLintInstincts(t, ctx, conn)

	judge := func(_ context.Context, _, _ InstinctRow) (DedupDecision, error) {
		return DedupDecision{Decision: decisionDuplicate, Reasoning: "同じ知見の言い換え"}, nil
	}

	var buf strings.Builder
	if err := execDedup(ctx, doltrepo.NewRepository(conn), judge, 0.0, &buf); err != nil {
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

// execDedupは3つのモデルのスコアをすべてdedup_decisionsに記録する
func TestExecDedup_AllModelScoresAreRecorded(t *testing.T) {
	ctx, conn := setupTestDB(t)
	insertLintInstincts(t, ctx, conn)

	judge := func(_ context.Context, _, _ InstinctRow) (DedupDecision, error) {
		return DedupDecision{Decision: decisionDistinct}, nil
	}

	var buf strings.Builder
	if err := execDedup(ctx, doltrepo.NewRepository(conn), judge, 0.0, &buf); err != nil {
		t.Fatalf("execDedup: %v", err)
	}

	var simBigram, simTrigram, simOverlap float64
	if err := conn.QueryRowContext(ctx,
		"SELECT sim_bigram, sim_trigram, sim_overlap FROM dedup_decisions LIMIT 1",
	).Scan(&simBigram, &simTrigram, &simOverlap); err != nil {
		t.Fatalf("query scores: %v", err)
	}
	if simBigram <= 0 {
		t.Errorf("expected sim_bigram > 0, got %f", simBigram)
	}
	if simTrigram <= 0 {
		t.Errorf("expected sim_trigram > 0, got %f", simTrigram)
	}
	if simOverlap <= 0 {
		t.Errorf("expected sim_overlap > 0, got %f", simOverlap)
	}
}

// execDedupはduplicateと判定されたペアをdedup_decisionsに記録する
func TestExecDedup_DuplicateDecisionIsRecorded(t *testing.T) {
	ctx, conn := setupTestDB(t)
	insertLintInstincts(t, ctx, conn)

	judge := func(_ context.Context, _, _ InstinctRow) (DedupDecision, error) {
		return DedupDecision{Decision: decisionDuplicate, Reasoning: "同じ知見の言い換え"}, nil
	}

	var buf strings.Builder
	if err := execDedup(ctx, doltrepo.NewRepository(conn), judge, 0.0, &buf); err != nil {
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
