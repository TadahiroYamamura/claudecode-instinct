package main

import (
	"context"
	"database/sql"
	"strings"
	"testing"
)

var nopPush doltPushFunc = func(_ context.Context, _ *sql.Conn, _, _ string) error {
	return nil
}

// execPushはremote_urlが未設定のときエラーを返す
func TestExecPush_FailsWhenRemoteURLEmpty(t *testing.T) {
	ctx, conn := setupTestDB(t)

	var buf strings.Builder
	err := execPush(ctx, conn, &InstinctConfig{}, nopPush, &buf)
	if err == nil {
		t.Fatal("expected error when remote_url is empty, got nil")
	}
}
