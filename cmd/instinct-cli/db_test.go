package main

import (
	"os"
	"path/filepath"
	"testing"

	doltrepo "github.com/TadahiroYamamura/claudecode-instinct/cmd/instinct-cli/internal/dolt"
)

// git configに存在しないキーはエラーを返す
func TestGitConfigValue_ReturnsErrorWhenKeyNotSet(t *testing.T) {
	_, err := gitConfigValue("nonexistent.key.xyz")
	if err == nil {
		t.Error("expected error for missing git config key")
	}
}

// openConnはinstincts DBが存在しない場合エラーを返す
func TestOpenConn_ErrorWhenDatabaseAbsent(t *testing.T) {
	dir := t.TempDir()
	mustRun(t, "git", "-C", dir, "init")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	db, err := doltrepo.OpenDB(dir, "Test", "test@test.com")
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
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
