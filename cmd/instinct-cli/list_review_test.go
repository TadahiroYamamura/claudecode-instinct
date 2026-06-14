package main

import (
	"strings"
	"testing"

	doltrepo "github.com/TadahiroYamamura/claudecode-instinct/cmd/instinct-cli/internal/dolt"
)


// listReviewInstincts は個人ブランチにのみ存在するinstinctを返す
func TestListReviewInstincts_ShowsPendingInstincts(t *testing.T) {
	ctx, conn := setupTestDB(t)

	if _, err := insertInstinct(ctx, conn, InsertParams{
		Content: "already on main", TriggerDesc: "always",
		Domain: "git", Scope: "global", ObservationCount: 6, ProjectID: "abc123def456",
	}); err != nil {
		t.Fatalf("insert on main: %v", err)
	}
	if _, err := conn.ExecContext(ctx, `CALL dolt_commit('-Am', 'test: main instinct')`); err != nil {
		t.Fatalf("commit on main: %v", err)
	}
	if _, err := conn.ExecContext(ctx, `CALL dolt_checkout('-b', 'personal')`); err != nil {
		t.Fatalf("checkout personal: %v", err)
	}
	if _, err := insertInstinct(ctx, conn, InsertParams{
		Content: "pending review instinct", TriggerDesc: "sometimes",
		Domain: "testing", Scope: "project", ObservationCount: 6, ProjectID: "abc123def456",
	}); err != nil {
		t.Fatalf("insert on personal: %v", err)
	}

	rows, err := listReviewInstincts(ctx, conn, "main", 6)
	if err != nil {
		t.Fatalf("listReviewInstincts: %v", err)
	}
	found := false
	for _, r := range rows {
		if strings.Contains(r.Content, "pending review instinct") {
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

// listReviewInstincts は observation_count が閾値未満のinstinctを除外する
func TestListReviewInstincts_ExcludesBelowThreshold(t *testing.T) {
	ctx, conn := setupTestDB(t)

	if _, err := conn.ExecContext(ctx, `CALL dolt_commit('-Am', 'test: init')`); err != nil {
		t.Fatalf("commit: %v", err)
	}
	if _, err := conn.ExecContext(ctx, `CALL dolt_checkout('-b', 'personal')`); err != nil {
		t.Fatalf("checkout: %v", err)
	}

	insertInstinct(ctx, conn, InsertParams{
		Content: "tentative instinct", TriggerDesc: "rarely",
		Domain: "testing", Scope: "project", ObservationCount: 3, ProjectID: "abc123def456",
	})
	insertInstinct(ctx, conn, InsertParams{
		Content: "strong instinct", TriggerDesc: "often",
		Domain: "testing", Scope: "project", ObservationCount: 6, ProjectID: "abc123def456",
	})

	rows, err := listReviewInstincts(ctx, conn, "main", 6)
	if err != nil {
		t.Fatalf("listReviewInstincts: %v", err)
	}
	for _, r := range rows {
		if strings.Contains(r.Content, "tentative instinct") {
			t.Error("tentative instinct (obs<6) should be excluded")
		}
	}
	found := false
	for _, r := range rows {
		if strings.Contains(r.Content, "strong instinct") {
			found = true
		}
	}
	if !found {
		t.Error("expected strong instinct (obs>=6) in results")
	}
}

// execNominateList は候補を一覧表示する
func TestExecNominateList_ShowsCandidates(t *testing.T) {
	ctx, conn := setupTestDB(t)

	conn.ExecContext(ctx, `CALL dolt_commit('-Am', 'test: init')`)   //nolint:errcheck
	conn.ExecContext(ctx, `CALL dolt_checkout('-b', 'personal')`)    //nolint:errcheck
	id, _ := insertInstinct(ctx, conn, InsertParams{
		Content: "TDDを実践する", TriggerDesc: "実装開始時",
		Domain: "development", Scope: "project", ObservationCount: 6, ProjectID: "abc123def456",
	})

	cfg := &InstinctConfig{Confidence: ConfidenceConfig{ReviewMin: 6}}
	var buf strings.Builder
	if err := execNominateList(ctx, doltrepo.NewRepository(conn), cfg, &buf); err != nil {
		t.Fatalf("execNominateList: %v", err)
	}

	if !strings.Contains(buf.String(), "TDDを実践する") {
		t.Errorf("expected content in output, got %q", buf.String())
	}
	if !strings.Contains(buf.String(), id[:8]) {
		t.Errorf("expected short ID in output, got %q", buf.String())
	}
}

// execNominateList は候補が0件のとき0件メッセージを出力する
func TestExecNominateList_ZeroItemsMessage(t *testing.T) {
	ctx, conn := setupTestDB(t)

	conn.ExecContext(ctx, `CALL dolt_commit('-Am', 'test: init')`)   //nolint:errcheck
	conn.ExecContext(ctx, `CALL dolt_checkout('-b', 'personal')`)    //nolint:errcheck

	cfg := &InstinctConfig{}
	var buf strings.Builder
	if err := execNominateList(ctx, doltrepo.NewRepository(conn), cfg, &buf); err != nil {
		t.Fatalf("execNominateList: %v", err)
	}
	if !strings.Contains(buf.String(), "0") {
		t.Errorf("expected 0-items message, got: %s", buf.String())
	}
}

// listReviewInstincts はconfigで指定したチームブランチを参照する
func TestListReviewInstincts_UsesConfiguredTeamBranch(t *testing.T) {
	ctx, conn := setupTestDB(t)

	if _, err := conn.ExecContext(ctx, `CALL dolt_commit('-Am', 'test: init')`); err != nil {
		t.Fatalf("commit: %v", err)
	}
	if _, err := conn.ExecContext(ctx, `CALL dolt_checkout('-b', 'team')`); err != nil {
		t.Fatalf("checkout team: %v", err)
	}
	if _, err := conn.ExecContext(ctx, `CALL dolt_checkout('-b', 'personal')`); err != nil {
		t.Fatalf("checkout personal: %v", err)
	}
	insertInstinct(ctx, conn, InsertParams{
		Content: "personal only", TriggerDesc: "sometimes",
		Domain: "testing", Scope: "project", ObservationCount: 6, ProjectID: "abc123def456",
	})

	rows, err := listReviewInstincts(ctx, conn, "team", 6)
	if err != nil {
		t.Fatalf("listReviewInstincts: %v", err)
	}
	found := false
	for _, r := range rows {
		if strings.Contains(r.Content, "personal only") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected personal instinct in results")
	}
}
