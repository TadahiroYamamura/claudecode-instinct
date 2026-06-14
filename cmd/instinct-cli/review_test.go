package main

import (
	"strings"
	"testing"

	doltrepo "github.com/TadahiroYamamura/claudecode-instinct/cmd/instinct-cli/internal/dolt"
)

// execReviewList はreview_queueの内容を一覧表示する
func TestExecReviewList_ShowsQueueItems(t *testing.T) {
	ctx, conn := setupTestDB(t)

	id, _ := insertInstinct(ctx, conn, InsertParams{
		Content: "promote me", TriggerDesc: "always", Domain: "git",
		Scope: "project", ObservationCount: 6, ProjectID: "abc123def456",
	})
	conn.ExecContext(ctx, `CALL dolt_commit('-Am', 'test: instinct')`) //nolint:errcheck
	conn.ExecContext(ctx, `CALL dolt_checkout('-b', 'personal')`)      //nolint:errcheck

	submitToReviewQueue(ctx, conn, "main", []InstinctRow{
		{ID: id, Content: "promote me", TriggerDesc: "always", Domain: "git",
			Scope: "project", ObservationCount: 6, ProjectID: "abc123def456"},
	}, "personal", "alice") //nolint:errcheck

	cfg := &InstinctConfig{}
	var buf strings.Builder
	if err := execReviewList(ctx, doltrepo.NewRepository(conn), cfg, &buf); err != nil {
		t.Fatalf("execReviewList: %v", err)
	}

	if !strings.Contains(buf.String(), "promote me") {
		t.Errorf("expected content in output, got %q", buf.String())
	}
	if !strings.Contains(buf.String(), id[:shortIDLen]) {
		t.Errorf("expected short ID in output, got %q", buf.String())
	}
}

// execReviewList はreview_queueが空のとき0件メッセージを出力する
func TestExecReviewList_ZeroItemsMessage(t *testing.T) {
	ctx, conn := setupTestDB(t)

	conn.ExecContext(ctx, `CALL dolt_commit('-Am', 'test: init')`) //nolint:errcheck

	cfg := &InstinctConfig{}
	var buf strings.Builder
	if err := execReviewList(ctx, doltrepo.NewRepository(conn), cfg, &buf); err != nil {
		t.Fatalf("execReviewList: %v", err)
	}
	if !strings.Contains(buf.String(), "0") {
		t.Errorf("expected 0-items message, got %q", buf.String())
	}
}

// execReviewApprove は指定IDをteamブランチに昇格させる
func TestExecReviewApprove_ByIDs_PromotesToTeam(t *testing.T) {
	ctx, conn := setupTestDB(t)

	id, _ := insertInstinct(ctx, conn, InsertParams{
		Content: "approve me", TriggerDesc: "always", Domain: "git",
		Scope: "project", ObservationCount: 6, ProjectID: "abc123def456",
	})
	conn.ExecContext(ctx, `CALL dolt_commit('-Am', 'test: instinct')`) //nolint:errcheck
	conn.ExecContext(ctx, `CALL dolt_checkout('-b', 'personal')`)      //nolint:errcheck

	submitToReviewQueue(ctx, conn, "main", []InstinctRow{
		{ID: id, Content: "approve me", TriggerDesc: "always", Domain: "git",
			Scope: "project", ObservationCount: 6, ProjectID: "abc123def456"},
	}, "personal", "alice") //nolint:errcheck

	cfg := &InstinctConfig{}
	var buf strings.Builder
	if err := execReviewApprove(ctx, doltrepo.NewRepository(conn), cfg, "personal", "bob", []string{id[:shortIDLen]}, &buf); err != nil {
		t.Fatalf("execReviewApprove: %v", err)
	}

	conn.ExecContext(ctx, `CALL dolt_checkout('main')`) //nolint:errcheck
	var count int
	conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM instincts WHERE id = ?", id).Scan(&count) //nolint:errcheck
	if count != 1 {
		t.Errorf("expected instinct promoted to main, got count=%d", count)
	}
	if !strings.Contains(buf.String(), "promoted") {
		t.Errorf("expected 'promoted' in output, got %q", buf.String())
	}
}
