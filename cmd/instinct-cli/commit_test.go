package main

import (
	"testing"

	doltrepo "github.com/TadahiroYamamura/claudecode-instinct/cmd/instinct-cli/internal/dolt"
)

// execCommitはRepositoryを通じてworking setをdoltコミットとして記録する
func TestExecCommit_CreatesDoltCommitViaRepository(t *testing.T) {
	ctx, conn := setupTestDB(t)

	if _, err := insertInstinct(ctx, conn, InsertParams{
		Content: "テスト前にlintを通す", TriggerDesc: "テスト実行時",
		Domain: "testing", Scope: "project", ObservationCount: 1, ProjectID: "abc",
	}); err != nil {
		t.Fatalf("insert: %v", err)
	}

	var before int
	if err := conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM dolt_log").Scan(&before); err != nil {
		t.Fatalf("dolt_log before: %v", err)
	}

	if err := execCommit(ctx, doltrepo.NewRepository(conn), "observer: 1 instinct"); err != nil {
		t.Fatalf("execCommit: %v", err)
	}

	var after int
	if err := conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM dolt_log").Scan(&after); err != nil {
		t.Fatalf("dolt_log after: %v", err)
	}
	if after != before+1 {
		t.Errorf("expected dolt_log to grow by 1, before=%d after=%d", before, after)
	}
}

