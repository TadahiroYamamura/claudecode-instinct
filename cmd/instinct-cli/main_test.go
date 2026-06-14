package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	doltrepo "github.com/TadahiroYamamura/claudecode-instinct/cmd/instinct-cli/internal/dolt"
)


// siblingワークツリーはsetup前にメインの.instinct-dbを誤発見しない
func TestFindProjectDirFrom_SiblingWorktree_NotFoundBeforeSetup(t *testing.T) {
	mainDir := t.TempDir()
	mustRun(t, "git", "-C", mainDir, "init")
	mustRun(t, "git", "-C", mainDir, "config", "user.email", "test@test.com")
	mustRun(t, "git", "-C", mainDir, "config", "user.name", "Test")
	mustRun(t, "git", "-C", mainDir, "commit", "--allow-empty", "-m", "init")
	if err := os.MkdirAll(filepath.Join(mainDir, ".instinct-db"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// sibling: mainDirの外に作成
	siblingDir := filepath.Join(t.TempDir(), "project-feature")
	mustRun(t, "git", "-C", mainDir, "worktree", "add", "-b", "feature", siblingDir)

	_, err := findProjectDirFrom(siblingDir)
	if err == nil {
		t.Error("sibling worktree should not find main worktree's .instinct-db before setup")
	}
}

// siblingワークツリーはsetup後に自分の.instinct-dbを発見する
func TestFindProjectDirFrom_SiblingWorktree_FindsOwnAfterSetup(t *testing.T) {
	mainDir := t.TempDir()
	mustRun(t, "git", "-C", mainDir, "init")
	mustRun(t, "git", "-C", mainDir, "config", "user.email", "test@test.com")
	mustRun(t, "git", "-C", mainDir, "config", "user.name", "Test")
	mustRun(t, "git", "-C", mainDir, "commit", "--allow-empty", "-m", "init")

	siblingDir := filepath.Join(t.TempDir(), "project-feature")
	mustRun(t, "git", "-C", mainDir, "worktree", "add", "-b", "feature", siblingDir)

	// setup後: siblingに固有の.instinct-dbを作成
	if err := os.MkdirAll(filepath.Join(siblingDir, ".instinct-db"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	found, err := findProjectDirFrom(siblingDir)
	if err != nil {
		t.Fatalf("sibling worktree should find own .instinct-db after setup: %v", err)
	}
	if found != siblingDir {
		t.Errorf("expected %s, got %s", siblingDir, found)
	}
}

// in-treeワークツリーからgit境界を越えて親の.instinct-dbを誤発見しない
func TestFindProjectDirFrom_DoesNotCrossIntoParentWorktree(t *testing.T) {
	mainDir := t.TempDir()
	mustRun(t, "git", "-C", mainDir, "init")
	mustRun(t, "git", "-C", mainDir, "config", "user.email", "test@test.com")
	mustRun(t, "git", "-C", mainDir, "config", "user.name", "Test")
	mustRun(t, "git", "-C", mainDir, "commit", "--allow-empty", "-m", "init")

	if err := os.MkdirAll(filepath.Join(mainDir, ".instinct-db"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// in-treeワークツリー: mainDir 配下に作成
	worktreeDir := filepath.Join(mainDir, "worktrees", "feature")
	mustRun(t, "git", "-C", mainDir, "worktree", "add", "-b", "feature", worktreeDir)

	// worktree 内から探索しても親の .instinct-db は見えてはいけない
	_, err := findProjectDirFrom(worktreeDir)
	if err == nil {
		t.Error("should not find parent worktree's .instinct-db across git boundary")
	}
}



// サブコマンドなしのとき.instinct-db探索エラーではなく使用法エラーを返す
func TestDispatch_NoArgs_ReturnsUsageErrorNotProjectDirError(t *testing.T) {
	dir := t.TempDir() // .instinct-dbが存在しないディレクトリ

	err := dispatch([]string{}, dir, strings.NewReader(""), io.Discard)

	if err == nil {
		t.Fatal("expected error for no args")
	}
	if strings.Contains(err.Error(), ".instinct-db") {
		t.Errorf("should not search for .instinct-db when no subcommand given, got: %v", err)
	}
}

// dispatchはdedupサブコマンドをexecDedupにルーティングする（instinctが0件なのでjudgeは呼ばれない）
func TestDispatch_DedupCommand_ZeroPairsWhenEmpty(t *testing.T) {
	dir := t.TempDir()
	mustRun(t, "git", "-C", dir, "init")
	if err := execInit(dir, initParams{Yes: true}, nil, nil, doltRepoFn); err != nil {
		t.Fatalf("execInit: %v", err)
	}

	var buf strings.Builder
	if err := dispatch([]string{"dedup"}, dir, nil, &buf); err != nil {
		t.Fatalf("dedup: %v", err)
	}
	if !strings.Contains(buf.String(), "0") {
		t.Errorf("expected '0 pairs' in output, got: %q", buf.String())
	}
}

// dispatch(["connect"])が"connect"コマンドにルーティングされる（--remote-url未指定でエラー）
func TestDispatch_ConnectCommand_RoutesToExecConnect(t *testing.T) {
	dir := t.TempDir()
	mustRun(t, "git", "-C", dir, "init")
	if err := execInit(dir, initParams{Yes: true}, nil, nil, doltRepoFn); err != nil {
		t.Fatalf("execInit: %v", err)
	}

	err := dispatch([]string{"connect", "--refs", "refs/dolt/myproject"}, dir, nil, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "git remote") {
		t.Errorf("expected git remote error, got: %v", err)
	}
}

// execInsert(insertFlags)がDBにレコードを保存する
func TestCLI_InsertCommand_StoresRecord(t *testing.T) {
	ctx, conn := setupTestDB(t)

	err := execInsert(ctx, doltrepo.NewRepository(conn), insertFlags{
		Content: "テスト前に仕様を確認する",
		Trigger: "テスト実行時",
		Domain:  "testing",
		Count:   2,
		Scope:   "project",
	}, func(string) (string, error) { return "abc123def456", nil })
	if err != nil {
		t.Fatalf("execInsert: %v", err)
	}

	var count int
	if err := conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM instincts").Scan(&count); err != nil {
		t.Fatalf("query: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 record, got %d", count)
	}
}
