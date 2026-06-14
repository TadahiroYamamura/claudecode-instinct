package main

import (
	"io"
	"strings"
	"testing"

	doltrepo "github.com/TadahiroYamamura/claudecode-instinct/cmd/instinct-cli/internal/dolt"
)

func selectAllReviewQueueSelector(rows []ReviewQueueRow, _ io.Writer) ([]string, error) {
	ids := make([]string, len(rows))
	for i, r := range rows {
		ids[i] = r.InstinctID
	}
	return ids, nil
}

// execReview はreview_queueの全件を選択してteamブランチに昇格させる
func TestExecReview_PromotesSelectedInstincts(t *testing.T) {
	ctx, conn := setupTestDB(t)

	id, _ := insertInstinct(ctx, conn, InsertParams{
		Content: "promote me", TriggerDesc: "always", Domain: "git",
		Scope: "project", ObservationCount: 6, ProjectID: "abc123def456",
	})
	conn.ExecContext(ctx, `CALL dolt_commit('-Am', 'test: instinct on main')`) //nolint:errcheck
	conn.ExecContext(ctx, `CALL dolt_checkout('-b', 'personal')`)              //nolint:errcheck

	row := InstinctRow{
		ID: id, Content: "promote me", TriggerDesc: "always",
		Domain: "git", Scope: "project", ObservationCount: 6, ProjectID: "abc123def456",
	}
	submitToReviewQueue(ctx, conn, "main", []InstinctRow{row}, "personal", "alice") //nolint:errcheck

	cfg := &InstinctConfig{Confidence: ConfidenceConfig{ReviewMin: 6}}
	var buf strings.Builder
	if err := execReview(ctx, doltrepo.NewRepository(conn), cfg, "personal", "bob", selectAllReviewQueueSelector, &buf); err != nil {
		t.Fatalf("execReview: %v", err)
	}

	// チームブランチに昇格されている
	conn.ExecContext(ctx, `CALL dolt_checkout('main')`) //nolint:errcheck
	var count int
	conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM instincts WHERE id = ?", id).Scan(&count) //nolint:errcheck
	if count != 1 {
		t.Errorf("expected instinct promoted to main, got count=%d", count)
	}

	// review_queue から削除されている
	var qCount int
	conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM review_queue").Scan(&qCount) //nolint:errcheck
	if qCount != 0 {
		t.Errorf("expected review_queue empty after promotion, got count=%d", qCount)
	}

	if !strings.Contains(buf.String(), "promoted") {
		t.Errorf("expected 'promoted' in output, got %q", buf.String())
	}
}

// execReview はreview_queueが空のとき0件メッセージを出力する
func TestExecReview_ZeroItemsMessage(t *testing.T) {
	ctx, conn := setupTestDB(t)

	conn.ExecContext(ctx, `CALL dolt_commit('-Am', 'test: init')`) //nolint:errcheck
	conn.ExecContext(ctx, `CALL dolt_checkout('-b', 'personal')`)  //nolint:errcheck

	cfg := &InstinctConfig{}
	var buf strings.Builder
	noOpSelector := func(_ []ReviewQueueRow, _ io.Writer) ([]string, error) { return nil, nil }
	if err := execReview(ctx, doltrepo.NewRepository(conn), cfg, "personal", "bob", noOpSelector, &buf); err != nil {
		t.Fatalf("execReview: %v", err)
	}
	if !strings.Contains(buf.String(), "0") {
		t.Errorf("expected 0-items message, got %q", buf.String())
	}
}
