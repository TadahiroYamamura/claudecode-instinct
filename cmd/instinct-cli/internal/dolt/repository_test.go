package dolt_test

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/dolthub/driver"

	doltrepo "github.com/TadahiroYamamura/claudecode-instinct/cmd/instinct-cli/internal/dolt"
	"github.com/TadahiroYamamura/claudecode-instinct/cmd/instinct-cli/internal/instincts"
)

func setupTestRepo(t *testing.T) (context.Context, *sql.Conn, *doltrepo.Repository) {
	t.Helper()
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	dsn := "file://" + dataDir + "?commitname=Test&commitemail=test@test.com"
	db, err := sql.Open("dolt", dsn)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}

	ctx := context.Background()
	conn, err := db.Conn(ctx)
	if err != nil {
		db.Close()
		t.Fatalf("db.Conn: %v", err)
	}

	for _, stmt := range append([]string{"CREATE DATABASE instincts", "USE instincts"}, doltrepo.Schema()...) {
		if _, err := conn.ExecContext(ctx, stmt); err != nil {
			conn.Close()
			db.Close()
			t.Fatalf("setup: %v", err)
		}
	}

	t.Cleanup(func() {
		conn.Close()
		db.Close()
	})

	return ctx, conn, doltrepo.NewRepository(conn)
}

// InsertDedupDecisionはdedup_decisionsテーブルにレコードを挿入する
func TestRepository_InsertDedupDecision(t *testing.T) {
	ctx, conn, repo := setupTestRepo(t)

	a := instincts.InstinctRow{ID: "aaaa0000-0000-0000-0000-000000000000", Content: "知見A", TriggerDesc: "trigger"}
	b := instincts.InstinctRow{ID: "bbbb0000-0000-0000-0000-000000000000", Content: "知見B", TriggerDesc: "trigger"}
	scores := instincts.SimilarityScores{Bigram: 0.8, Trigram: 0.7, Overlap: 0.9}
	d := instincts.DedupDecision{Decision: "distinct", Reasoning: "similar but distinct"}

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
	ctx, conn, repo := setupTestRepo(t)

	winnerID, err := repo.InsertInstinct(ctx, instincts.InsertParams{
		Content: "winner", TriggerDesc: "t", Domain: "d", Scope: "project", ObservationCount: 3, ProjectID: "abc",
	})
	if err != nil {
		t.Fatalf("insert winner: %v", err)
	}
	loserID, err := repo.InsertInstinct(ctx, instincts.InsertParams{
		Content: "loser", TriggerDesc: "t", Domain: "d", Scope: "project", ObservationCount: 2, ProjectID: "abc",
	})
	if err != nil {
		t.Fatalf("insert loser: %v", err)
	}

	winner := instincts.InstinctRow{ID: winnerID, ObservationCount: 3}
	loser := instincts.InstinctRow{ID: loserID, ObservationCount: 2}

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

// ListReviewInstinctsは個人ブランチにのみ存在するinstinctを返す
func TestRepository_ListReviewInstincts_ShowsPendingInstincts(t *testing.T) {
	ctx, conn, repo := setupTestRepo(t)

	if _, err := repo.InsertInstinct(ctx, instincts.InsertParams{
		Content: "already on main", TriggerDesc: "always",
		Domain: "git", Scope: "global", ObservationCount: 6, ProjectID: "abc123def456",
	}); err != nil {
		t.Fatalf("insert: %v", err)
	}
	if _, err := conn.ExecContext(ctx, `CALL dolt_commit('-Am', 'test: main instinct')`); err != nil {
		t.Fatalf("commit: %v", err)
	}
	if _, err := conn.ExecContext(ctx, `CALL dolt_checkout('-b', 'personal')`); err != nil {
		t.Fatalf("checkout: %v", err)
	}
	if _, err := repo.InsertInstinct(ctx, instincts.InsertParams{
		Content: "pending review", TriggerDesc: "sometimes",
		Domain: "testing", Scope: "project", ObservationCount: 6, ProjectID: "abc123def456",
	}); err != nil {
		t.Fatalf("insert: %v", err)
	}

	rows, err := repo.ListReviewInstincts(ctx, "main", 6)
	if err != nil {
		t.Fatalf("ListReviewInstincts: %v", err)
	}

	found := false
	for _, r := range rows {
		if strings.Contains(r.Content, "pending review") {
			found = true
		}
		if strings.Contains(r.Content, "already on main") {
			t.Error("already-merged instinct should not appear")
		}
	}
	if !found {
		t.Error("expected pending instinct in results")
	}
}

// SubmitToReviewQueueは選択したinstinctをチームブランチのreview_queueに挿入する
func TestRepository_SubmitToReviewQueue_InsertsOnTeamBranch(t *testing.T) {
	ctx, conn, repo := setupTestRepo(t)

	id, err := repo.InsertInstinct(ctx, instincts.InsertParams{
		Content: "submit via repo", TriggerDesc: "often",
		Domain: "testing", Scope: "project", ObservationCount: 6, ProjectID: "abc123def456",
	})
	if err != nil {
		t.Fatalf("insert: %v", err)
	}
	conn.ExecContext(ctx, `CALL dolt_commit('-Am', 'test: instinct')`)    //nolint:errcheck
	conn.ExecContext(ctx, `CALL dolt_checkout('-b', 'personal')`)          //nolint:errcheck

	row := instincts.InstinctRow{
		ID: id, Content: "submit via repo", TriggerDesc: "often",
		Domain: "testing", Scope: "project", ObservationCount: 6,
	}

	if err := repo.SubmitToReviewQueue(ctx, "main", []instincts.InstinctRow{row}, "personal", "Test User"); err != nil {
		t.Fatalf("SubmitToReviewQueue: %v", err)
	}

	conn.ExecContext(ctx, `CALL dolt_checkout('main')`) //nolint:errcheck
	var count int
	if err := conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM review_queue WHERE instinct_id = ?", id).Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 record in review_queue, got %d", count)
	}
}
