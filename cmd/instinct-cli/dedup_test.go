package main

import (
	"context"
	"strings"
	"testing"
)

// execDedupはinstinctが0件のとき0ペアをチェックしたと報告する
func TestExecDedup_EmptyInstinctsReportsZeroPairs(t *testing.T) {
	ctx, conn := setupTestDB(t)

	var buf strings.Builder
	judge := func(_ context.Context, _, _ InstinctRow) (DedupDecision, error) {
		return DedupDecision{}, nil
	}
	if err := execDedup(ctx, conn, judge, &buf); err != nil {
		t.Fatalf("execDedup: %v", err)
	}
	if !strings.Contains(buf.String(), "0") {
		t.Errorf("expected output to mention 0, got: %q", buf.String())
	}
}
