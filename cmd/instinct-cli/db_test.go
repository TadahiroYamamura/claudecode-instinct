package main

import (
	"io"
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

// openProjectConnはconfig.user.ymlが存在しない場合エラーを返す（setup未実施を意味する）
func TestOpenProjectConn_ErrorWhenUserConfigAbsent(t *testing.T) {
	dir := t.TempDir()
	gitInitWithRemote(t, dir)
	// config.user.yml を作成せずDBだけ初期化する（setup未実施相当）
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

// openProjectConnはsetup後にDBへの接続に成功する
func TestOpenProjectConn_SucceedsAfterSetup(t *testing.T) {
	dir := t.TempDir()
	gitInitWithRemote(t, dir)
	if err := execSetup(dir, setupParams{Yes: true}, nil, io.Discard, fakeCloneFail, fakePush); err != nil {
		t.Fatalf("execSetup: %v", err)
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
	gitInitWithRemote(t, dir)
	dbDir := filepath.Join(dir, ".instinct-db")
	if err := execSetup(dir, setupParams{Yes: true}, nil, io.Discard, fakeCloneFail, fakePush); err != nil {
		t.Fatalf("execSetup: %v", err)
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
