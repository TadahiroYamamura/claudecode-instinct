package main

import (
	"strings"
	"testing"
)

// review は個人ブランチにのみ存在する（チームブランチ未マージの）instinct を表示する
func TestReview_ShowsPendingInstincts(t *testing.T) {
	ctx, conn := setupTestDB(t)

	// main ブランチに shared レコードを追加してコミット
	if _, err := insertInstinct(ctx, conn, InsertParams{
		Content: "already on main", TriggerDesc: "always",
		Domain: "git", Scope: "global", ObservationCount: 5, ProjectID: "abc123def456",
	}); err != nil {
		t.Fatalf("insert on main: %v", err)
	}
	if _, err := conn.ExecContext(ctx, `CALL dolt_commit('-Am', 'test: main instinct')`); err != nil {
		t.Fatalf("commit on main: %v", err)
	}

	// personal ブランチを作成して切り替え（main の内容を引き継ぐ）
	if _, err := conn.ExecContext(ctx, `CALL dolt_checkout('-b', 'personal')`); err != nil {
		t.Fatalf("checkout personal: %v", err)
	}

	// personal ブランチのみに新規レコードを追加（medium 閾値以上）
	if _, err := insertInstinct(ctx, conn, InsertParams{
		Content: "pending review instinct", TriggerDesc: "sometimes",
		Domain: "testing", Scope: "project", ObservationCount: 6, ProjectID: "abc123def456",
	}); err != nil {
		t.Fatalf("insert on personal: %v", err)
	}

	cfg := &InstinctConfig{Confidence: ConfidenceConfig{ReviewMin: 6}}
	var buf strings.Builder
	if err := execReview(ctx, conn, cfg, &buf); err != nil {
		t.Fatalf("execReview: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "pending review instinct") {
		t.Error("expected pending instinct in review output")
	}
	if strings.Contains(out, "already on main") {
		t.Error("already-merged instinct should not appear in review output")
	}
}

// review は observation_count が medium 閾値未満の instinct を除外する
func TestReview_ExcludesBelowMediumThreshold(t *testing.T) {
	ctx, conn := setupTestDB(t)

	if _, err := conn.ExecContext(ctx, `CALL dolt_commit('-Am', 'test: init schema')`); err != nil {
		t.Fatalf("commit: %v", err)
	}
	if _, err := conn.ExecContext(ctx, `CALL dolt_checkout('-b', 'personal')`); err != nil {
		t.Fatalf("checkout: %v", err)
	}

	// medium=6 未満（観察が少なく仮説段階）
	if _, err := insertInstinct(ctx, conn, InsertParams{
		Content: "tentative instinct", TriggerDesc: "rarely",
		Domain: "testing", Scope: "project", ObservationCount: 3, ProjectID: "abc123def456",
	}); err != nil {
		t.Fatalf("insert tentative: %v", err)
	}
	// medium=6 以上（strong → レビュー対象）
	if _, err := insertInstinct(ctx, conn, InsertParams{
		Content: "strong instinct", TriggerDesc: "often",
		Domain: "testing", Scope: "project", ObservationCount: 6, ProjectID: "abc123def456",
	}); err != nil {
		t.Fatalf("insert strong: %v", err)
	}

	cfg := &InstinctConfig{Confidence: ConfidenceConfig{ReviewMin: 6}}
	var buf strings.Builder
	if err := execReview(ctx, conn, cfg, &buf); err != nil {
		t.Fatalf("execReview: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "strong instinct") {
		t.Error("expected strong instinct (obs>=6) in review output")
	}
	if strings.Contains(out, "tentative instinct") {
		t.Error("tentative instinct (obs<6) should be excluded from review")
	}
}

// review は全 instinct が既にチームブランチにある場合、0件を表示する
func TestReview_EmptyWhenAllMerged(t *testing.T) {
	ctx, conn := setupTestDB(t)

	if _, err := insertInstinct(ctx, conn, InsertParams{
		Content: "merged instinct", TriggerDesc: "always",
		Domain: "git", Scope: "global", ObservationCount: 1, ProjectID: "abc123def456",
	}); err != nil {
		t.Fatalf("insert: %v", err)
	}
	if _, err := conn.ExecContext(ctx, `CALL dolt_commit('-Am', 'test: merged')`); err != nil {
		t.Fatalf("commit on main: %v", err)
	}
	if _, err := conn.ExecContext(ctx, `CALL dolt_checkout('-b', 'personal')`); err != nil {
		t.Fatalf("checkout personal: %v", err)
	}

	var buf strings.Builder
	if err := execReview(ctx, conn, &InstinctConfig{}, &buf); err != nil {
		t.Fatalf("execReview: %v", err)
	}
	if !strings.Contains(buf.String(), "0") {
		t.Errorf("expected 0 pending instincts, got:\n%s", buf.String())
	}
}

// review はconfig で指定したチームブランチを参照する
func TestReview_UsesConfiguredTeamBranch(t *testing.T) {
	ctx, conn := setupTestDB(t)

	if _, err := conn.ExecContext(ctx, `CALL dolt_commit('-Am', 'test: init schema')`); err != nil {
		t.Fatalf("commit schema: %v", err)
	}
	if _, err := conn.ExecContext(ctx, `CALL dolt_checkout('-b', 'team')`); err != nil {
		t.Fatalf("checkout team: %v", err)
	}
	if _, err := conn.ExecContext(ctx, `CALL dolt_checkout('-b', 'personal')`); err != nil {
		t.Fatalf("checkout personal: %v", err)
	}

	if _, err := insertInstinct(ctx, conn, InsertParams{
		Content: "personal only", TriggerDesc: "sometimes",
		Domain: "testing", Scope: "project", ObservationCount: 6, ProjectID: "abc123def456",
	}); err != nil {
		t.Fatalf("insert on personal: %v", err)
	}

	cfg := &InstinctConfig{
		Confidence: ConfidenceConfig{ReviewMin: 6},
		Dolt:       DoltConfig{TeamBranch: "team"},
	}

	var buf strings.Builder
	if err := execReview(ctx, conn, cfg, &buf); err != nil {
		t.Fatalf("execReview: %v", err)
	}
	if !strings.Contains(buf.String(), "personal only") {
		t.Errorf("expected personal instinct in output, got:\n%s", buf.String())
	}
}
