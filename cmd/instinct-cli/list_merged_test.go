package main

import (
	"strings"
	"testing"
)

// list --merged は main ブランチの instinct を含む
func TestListMerged_IncludesMainBranchInstincts(t *testing.T) {
	ctx, conn := setupTestDB(t)

	// main ブランチに "shared" レコードを追加してコミット
	if _, err := insertInstinct(ctx, conn, InsertParams{
		Content: "shared instinct on main", TriggerDesc: "always",
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

	// personal ブランチに追加レコード（main にはない）
	if _, err := insertInstinct(ctx, conn, InsertParams{
		Content: "personal only instinct", TriggerDesc: "sometimes",
		Domain: "testing", Scope: "project", ObservationCount: 2, ProjectID: "abc123def456",
	}); err != nil {
		t.Fatalf("insert on personal: %v", err)
	}

	var buf strings.Builder
	if err := execListMerged(ctx, conn, &buf); err != nil {
		t.Fatalf("execListMerged: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "personal only instinct") {
		t.Error("expected personal branch instinct in output")
	}
	if !strings.Contains(out, "shared instinct on main") {
		t.Error("expected main branch instinct in output")
	}
}

// list --merged は同一 ID のレコードを重複して表示しない
func TestListMerged_DeduplicatesByID(t *testing.T) {
	ctx, conn := setupTestDB(t)

	// main に shared レコードを追加してコミット
	sharedID, err := insertInstinct(ctx, conn, InsertParams{
		Content: "shared instinct", TriggerDesc: "always",
		Domain: "git", Scope: "global", ObservationCount: 5, ProjectID: "abc123def456",
	})
	if err != nil {
		t.Fatalf("insert shared: %v", err)
	}
	if _, err := conn.ExecContext(ctx, `CALL dolt_commit('-Am', 'test: shared instinct')`); err != nil {
		t.Fatalf("commit on main: %v", err)
	}

	// personal ブランチを作成（main の shared レコードを引き継ぐ）
	if _, err := conn.ExecContext(ctx, `CALL dolt_checkout('-b', 'personal')`); err != nil {
		t.Fatalf("checkout personal: %v", err)
	}

	var buf strings.Builder
	if err := execListMerged(ctx, conn, &buf); err != nil {
		t.Fatalf("execListMerged: %v", err)
	}

	// shared レコードは1件だけ表示される
	count := strings.Count(buf.String(), sharedID[:shortIDLen])
	if count != 1 {
		t.Errorf("expected shared ID to appear once, got %d times:\n%s", count, buf.String())
	}
}
