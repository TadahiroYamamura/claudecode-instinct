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

// InsertInstinctは同一内容を2回insertすると2レコードになる（dedup前）
func TestRepository_Insert_SameContentTwiceCreatesTwoRecords(t *testing.T) {
	ctx, conn, repo := setupTestRepo(t)

	params := instincts.InsertParams{
		Content: "git push前にテストを実行する", TriggerDesc: "git push時",
		Domain: "git", Scope: "global", ObservationCount: 2, ProjectID: "abc123def456",
	}
	if _, err := repo.InsertInstinct(ctx, params); err != nil {
		t.Fatalf("first insert: %v", err)
	}
	params.ObservationCount = 1
	if _, err := repo.InsertInstinct(ctx, params); err != nil {
		t.Fatalf("second insert: %v", err)
	}

	var totalCount, totalObs int
	if err := conn.QueryRowContext(ctx,
		"SELECT COUNT(*), SUM(observation_count) FROM instincts WHERE content = ?",
		"git push前にテストを実行する").Scan(&totalCount, &totalObs); err != nil {
		t.Fatalf("query: %v", err)
	}
	if totalCount != 2 {
		t.Errorf("expected 2 records before dedup, got %d", totalCount)
	}
	if totalObs != 3 {
		t.Errorf("expected observation_count sum = 3, got %d", totalObs)
	}
}

// InsertInstinctはUUID形式のIDを返す
func TestRepository_InsertInstinct_ReturnsGeneratedID(t *testing.T) {
	ctx, _, repo := setupTestRepo(t)

	id, err := repo.InsertInstinct(ctx, instincts.InsertParams{
		Content: "テスト実行前に仕様を確認する", TriggerDesc: "テスト実行時",
		Domain: "testing", Scope: "project", ObservationCount: 1, ProjectID: "abc123def456",
	})
	if err != nil {
		t.Fatalf("InsertInstinct: %v", err)
	}
	if id == "" {
		t.Error("expected non-empty ID")
	}
}

// InsertInstinctはinstinctsテーブルにレコードを保存する
func TestRepository_InsertInstinct_StoresRecord(t *testing.T) {
	ctx, conn, repo := setupTestRepo(t)

	if _, err := repo.InsertInstinct(ctx, instincts.InsertParams{
		Content: "テスト実行前に仕様を確認する", TriggerDesc: "テスト実行時",
		Domain: "testing", Scope: "project", ObservationCount: 5, ProjectID: "abc123def456",
	}); err != nil {
		t.Fatalf("InsertInstinct: %v", err)
	}

	var count int
	if err := conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM instincts WHERE content = ?",
		"テスト実行前に仕様を確認する").Scan(&count); err != nil {
		t.Fatalf("query: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 record, got %d", count)
	}
}

// ListInstinctsは挿入したレコードを返す
func TestRepository_ListInstincts_ReturnsInsertedRecord(t *testing.T) {
	ctx, _, repo := setupTestRepo(t)

	if _, err := repo.InsertInstinct(ctx, instincts.InsertParams{
		Content: "テスト前に仕様を確認する", TriggerDesc: "テスト実行時",
		Domain: "testing", Scope: "project", ObservationCount: 5, ProjectID: "abc123def456",
	}); err != nil {
		t.Fatalf("insert: %v", err)
	}

	rows, err := repo.ListInstincts(ctx)
	if err != nil {
		t.Fatalf("ListInstincts: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0].Content != "テスト前に仕様を確認する" {
		t.Errorf("content = %q", rows[0].Content)
	}
}

// ListInstinctsは新しい順に返す
func TestRepository_ListInstincts_OrderedNewestFirst(t *testing.T) {
	ctx, conn, _ := setupTestRepo(t)

	for _, row := range []struct {
		id        string
		content   string
		createdAt string
	}{
		{"id-old", "古い知見", "2026-01-01 00:00:00"},
		{"id-new", "新しい知見", "2026-06-01 00:00:00"},
	} {
		if _, err := conn.ExecContext(ctx,
			`INSERT INTO instincts (id, content, trigger_desc, domain, scope, observation_count, project_id, created_at)
			 VALUES (?, ?, '', '', 'project', 1, 'abc123def456', ?)`,
			row.id, row.content, row.createdAt); err != nil {
			t.Fatalf("insert %s: %v", row.id, err)
		}
	}

	repo := doltrepo.NewRepository(conn)
	rows, err := repo.ListInstincts(ctx)
	if err != nil {
		t.Fatalf("ListInstincts: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	if rows[0].Content != "新しい知見" {
		t.Errorf("expected newest first, got %q", rows[0].Content)
	}
}

// ListInstinctsは全フィールドを返す
func TestRepository_ListInstincts_ReturnsAllFields(t *testing.T) {
	ctx, _, repo := setupTestRepo(t)

	if _, err := repo.InsertInstinct(ctx, instincts.InsertParams{
		Content: "コミット前にlintを実行する", TriggerDesc: "git commit時",
		Domain: "git", Scope: "global", ObservationCount: 7, ProjectID: "abc123def456",
	}); err != nil {
		t.Fatalf("insert: %v", err)
	}

	rows, err := repo.ListInstincts(ctx)
	if err != nil {
		t.Fatalf("ListInstincts: %v", err)
	}
	r := rows[0]
	if r.TriggerDesc != "git commit時" {
		t.Errorf("trigger_desc = %q", r.TriggerDesc)
	}
	if r.Domain != "git" {
		t.Errorf("domain = %q", r.Domain)
	}
	if r.ObservationCount != 7 {
		t.Errorf("observation_count = %d", r.ObservationCount)
	}
	if r.Scope != "global" {
		t.Errorf("scope = %q", r.Scope)
	}
	if r.CreatedAt.IsZero() {
		t.Error("created_at is zero")
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
