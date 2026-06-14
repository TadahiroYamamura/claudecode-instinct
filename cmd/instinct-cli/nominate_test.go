package main

import (
	"io"
	"strings"
	"testing"

	doltrepo "github.com/TadahiroYamamura/claudecode-instinct/cmd/instinct-cli/internal/dolt"
)

// fakeSelector は指定したIDだけを返す（TUIの代替）
func fakeSelector(ids []string) nominateSelector {
	return func(_ []InstinctRow, _ io.Writer) ([]string, error) {
		return ids, nil
	}
}

// selectAllSelector は全行のIDを返す
func selectAllSelector(rows []InstinctRow, _ io.Writer) ([]string, error) {
	ids := make([]string, len(rows))
	for i, r := range rows {
		ids[i] = r.ID
	}
	return ids, nil
}


// submitToReviewQueue は選択したinstinctをチームブランチのreview_queueに挿入する
func TestSubmitToReviewQueue_InsertsOnTeamBranch(t *testing.T) {
	ctx, conn := setupTestDB(t)

	id, err := insertInstinct(ctx, conn, InsertParams{
		Content: "submit this", TriggerDesc: "often",
		Domain: "testing", Scope: "project", ObservationCount: 6, ProjectID: "abc123def456",
	})
	if err != nil {
		t.Fatalf("insert: %v", err)
	}
	if _, err := conn.ExecContext(ctx, `CALL dolt_commit('-Am', 'test: instinct')`); err != nil {
		t.Fatalf("commit: %v", err)
	}
	if _, err := conn.ExecContext(ctx, `CALL dolt_checkout('-b', 'personal')`); err != nil {
		t.Fatalf("checkout personal: %v", err)
	}

	row := InstinctRow{
		ID: id, Content: "submit this", TriggerDesc: "often",
		Domain: "testing", Scope: "project", ObservationCount: 6,
	}
	if err := submitToReviewQueue(ctx, conn, "main", []InstinctRow{row}, "personal", "Test User"); err != nil {
		t.Fatalf("submitToReviewQueue: %v", err)
	}

	// 現在のブランチ（personal）に戻っていることを確認
	var branch string
	if err := conn.QueryRowContext(ctx, "SELECT active_branch()").Scan(&branch); err != nil {
		t.Fatalf("active_branch: %v", err)
	}
	if branch != "personal" {
		t.Errorf("expected to be back on personal branch, got %q", branch)
	}

	// チームブランチのreview_queueに挿入されていることを確認
	if _, err := conn.ExecContext(ctx, `CALL dolt_checkout('main')`); err != nil {
		t.Fatalf("checkout main: %v", err)
	}
	var count int
	if err := conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM review_queue WHERE instinct_id = ?", id).Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 record in review_queue, got %d", count)
	}
}

// submitToReviewQueue は同じinstinct_idを再投入してもDUPLICATE KEY でエラーにならない
func TestSubmitToReviewQueue_IdempotentOnDuplicate(t *testing.T) {
	ctx, conn := setupTestDB(t)

	id, _ := insertInstinct(ctx, conn, InsertParams{
		Content: "already submitted", TriggerDesc: "often",
		Domain: "testing", Scope: "project", ObservationCount: 6, ProjectID: "abc123def456",
	})
	conn.ExecContext(ctx, `CALL dolt_commit('-Am', 'test')`)
	conn.ExecContext(ctx, `CALL dolt_checkout('-b', 'personal')`)

	row := InstinctRow{ID: id, Content: "already submitted", TriggerDesc: "often",
		Domain: "testing", Scope: "project", ObservationCount: 6}

	if err := submitToReviewQueue(ctx, conn, "main", []InstinctRow{row}, "personal", "Test User"); err != nil {
		t.Fatalf("first submit: %v", err)
	}
	conn.ExecContext(ctx, `CALL dolt_checkout('personal')`)
	if err := submitToReviewQueue(ctx, conn, "main", []InstinctRow{row}, "personal", "Test User"); err != nil {
		t.Fatalf("second submit should not error: %v", err)
	}
}

// execNominate はセレクターで選んだinstinctをreview_queueに登録する
func TestExecNominate_SubmitsSelectedInstincts(t *testing.T) {
	ctx, conn := setupTestDB(t)

	id, _ := insertInstinct(ctx, conn, InsertParams{
		Content: "strong instinct", TriggerDesc: "often",
		Domain: "testing", Scope: "project", ObservationCount: 6, ProjectID: "abc123def456",
	})
	conn.ExecContext(ctx, `CALL dolt_commit('-Am', 'test: instinct on main')`)
	conn.ExecContext(ctx, `CALL dolt_checkout('-b', 'personal')`)

	// personal ブランチに新規レコード（review 対象）
	_, _ = insertInstinct(ctx, conn, InsertParams{
		Content: "personal pending", TriggerDesc: "sometimes",
		Domain: "code", Scope: "project", ObservationCount: 7, ProjectID: "abc123def456",
	})

	cfg := &InstinctConfig{Confidence: ConfidenceConfig{ReviewMin: 6}}
	var buf strings.Builder
	// selectAllSelector で全件を選択
	if err := execNominate(ctx, doltrepo.NewRepository(conn), cfg, "personal", "Test User", selectAllSelector, &buf); err != nil {
		t.Fatalf("execNominate: %v", err)
	}

	// チームブランチに review_queue レコードがあることを確認
	conn.ExecContext(ctx, `CALL dolt_checkout('main')`)
	var count int
	conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM review_queue").Scan(&count)
	if count == 0 {
		t.Error("expected records in review_queue after execNominate")
	}
	_ = id
}

// execNominate はセレクターが空を返した場合はreview_queueに何も挿入しない
func TestExecNominate_NoSubmitWhenNoneSelected(t *testing.T) {
	ctx, conn := setupTestDB(t)

	conn.ExecContext(ctx, `CALL dolt_commit('-Am', 'test: init')`)
	conn.ExecContext(ctx, `CALL dolt_checkout('-b', 'personal')`)
	insertInstinct(ctx, conn, InsertParams{
		Content: "pending", TriggerDesc: "often",
		Domain: "testing", Scope: "project", ObservationCount: 6, ProjectID: "abc123def456",
	})

	cfg := &InstinctConfig{Confidence: ConfidenceConfig{ReviewMin: 6}}
	emptySelector := func(_ []InstinctRow, _ io.Writer) ([]string, error) { return nil, nil }

	if err := execNominate(ctx, doltrepo.NewRepository(conn), cfg, "personal", "Test User", emptySelector, io.Discard); err != nil {
		t.Fatalf("execNominate: %v", err)
	}

	conn.ExecContext(ctx, `CALL dolt_checkout('main')`)
	var count int
	conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM review_queue").Scan(&count)
	if count != 0 {
		t.Errorf("expected 0 records in review_queue, got %d", count)
	}
}
