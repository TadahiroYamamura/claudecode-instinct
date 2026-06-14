package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// git configに存在しないキーはエラーを返す
func TestGitConfigValue_ReturnsErrorWhenKeyNotSet(t *testing.T) {
	_, err := gitConfigValue("nonexistent.key.xyz")
	if err == nil {
		t.Error("expected error for missing git config key")
	}
}

// doltDSNに渡したname/emailがDSNに含まれる
func TestDoltDSN_IncludesCommitNameAndEmail(t *testing.T) {
	dsn := doltDSN("/tmp/data", "Taro Yamada", "taro@example.com")

	if !strings.Contains(dsn, "Taro") {
		t.Errorf("DSN does not include commit name, got: %s", dsn)
	}
	if !strings.Contains(dsn, "taro@example.com") || strings.Contains(dsn, "instinct@local") {
		t.Errorf("DSN does not include commit email, got: %s", dsn)
	}
}

// openConnはinstincts DBが存在しない場合エラーを返す
func TestOpenConn_ErrorWhenDatabaseAbsent(t *testing.T) {
	dir := t.TempDir()
	mustRun(t, "git", "-C", dir, "init")
	// doltファイルは作るがCREATE DATABASEは実行しない
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	db, err := openDoltDB(dir)
	if err != nil {
		t.Fatalf("openDoltDB: %v", err)
	}
	db.Close()

	_, cleanup, err := openConn(t.Context(), dir)
	if cleanup != nil {
		defer cleanup()
	}
	if err == nil {
		t.Error("expected error when instincts database is absent")
	}
}

// openProjectConnはconfig.user.ymlが存在しない場合エラーを返す（init未実施を意味する）
func TestOpenProjectConn_ErrorWhenUserConfigAbsent(t *testing.T) {
	dir := t.TempDir()
	mustRun(t, "git", "-C", dir, "init")
	// config.user.yml を作成せずDBだけ初期化する（init未実施相当）
	if err := setupDB(t.Context(), instinctDataDir(dir)); err != nil {
		t.Fatalf("setupDB: %v", err)
	}

	_, _, cleanup, err := openProjectConn(dir)
	if cleanup != nil {
		defer cleanup()
	}
	if err == nil {
		t.Error("expected error when config.user.yml is absent")
	}
}

// openProjectConnはinit後にDBへの接続に成功する
func TestOpenProjectConn_SucceedsAfterInit(t *testing.T) {
	dir := t.TempDir()
	mustRun(t, "git", "-C", dir, "init")
	if err := execInit(dir, initParams{Yes: true}, nil, nil); err != nil {
		t.Fatalf("execInit: %v", err)
	}

	conn, projectDir, cleanup, err := openProjectConn(dir)
	if err != nil {
		t.Fatalf("openProjectConn: %v", err)
	}
	defer cleanup()

	if conn == nil {
		t.Error("expected non-nil conn")
	}
	if projectDir == "" {
		t.Error("expected non-empty projectDir")
	}
}

// openProjectConnは指定ブランチにcheckoutする
func TestOpenProjectConn_CheckoutsPersonalBranch(t *testing.T) {
	dir := t.TempDir()
	mustRun(t, "git", "-C", dir, "init")
	dbDir := filepath.Join(dir, ".instinct-db")
	if err := execInit(dir, initParams{Yes: true}, nil, nil); err != nil {
		t.Fatalf("execInit: %v", err)
	}

	// "testuser" ブランチをDBに作成してからconfig.user.ymlを上書き
	setupConn, cleanup, err := openConn(t.Context(), instinctDataDir(dir))
	if err != nil {
		t.Fatalf("openConn: %v", err)
	}
	if _, err := setupConn.ExecContext(t.Context(), "CALL dolt_checkout('-b', 'testuser')"); err != nil {
		cleanup()
		t.Fatalf("create testuser branch: %v", err)
	}
	cleanup()

	if err := writeUserConfig(dbDir, "testuser"); err != nil {
		t.Fatalf("writeUserConfig: %v", err)
	}

	conn, _, connCleanup, err := openProjectConn(dir)
	if err != nil {
		t.Fatalf("openProjectConn: %v", err)
	}
	defer connCleanup()

	var branch string
	if err := conn.QueryRowContext(t.Context(), "SELECT active_branch()").Scan(&branch); err != nil {
		t.Fatalf("active_branch: %v", err)
	}
	if branch != "testuser" {
		t.Errorf("expected branch testuser, got %q", branch)
	}
}
