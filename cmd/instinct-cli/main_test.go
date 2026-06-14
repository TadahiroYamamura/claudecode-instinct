package main

import (
	"database/sql"
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
	if err.Error() == ".instinct-db not found in any parent directory" {
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
	want := "remote URL is not set: run 'git remote add origin <url>' to configure a remote"
	if err == nil || err.Error() != want {
		t.Errorf("expected %q, got: %v", want, err)
	}
}

// "instinct review" はサブコマンド(list/approve)が必須
func TestDispatch_ReviewWithoutSubcommand_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	gitInitWithRemote(t, dir)
	if err := execInit(dir, initParams{Yes: true}, nil, nil, doltRepoFn); err != nil {
		t.Fatalf("execInit: %v", err)
	}

	err := dispatch([]string{"review"}, dir, nil, io.Discard)
	if err == nil {
		t.Error("expected error when 'review' called without subcommand (list or approve required)")
	}
}

// dispatch("insert") → dispatch("list") でレコードが表示される
func TestDispatch_InsertThenList_RecordAppears(t *testing.T) {
	dir := t.TempDir()
	gitInitWithRemote(t, dir)
	if err := execInit(dir, initParams{Yes: true}, nil, nil, doltRepoFn); err != nil {
		t.Fatalf("execInit: %v", err)
	}

	if err := dispatch([]string{"insert",
		"--content", "TDDでテストを先に書く",
		"--trigger", "実装開始時",
		"--domain", "testing",
		"--count", "5",
	}, dir, nil, io.Discard); err != nil {
		t.Fatalf("dispatch insert: %v", err)
	}

	var buf strings.Builder
	if err := dispatch([]string{"list"}, dir, nil, &buf); err != nil {
		t.Fatalf("dispatch list: %v", err)
	}
	if !strings.Contains(buf.String(), "TDDでテストを先に書く") {
		t.Errorf("expected inserted content in list output, got:\n%s", buf.String())
	}
}

// dispatch("insert") → dispatch("commit") でパーソナルブランチのコミット数が増える
func TestDispatch_CommitCommand_IncreasesCommitCount(t *testing.T) {
	dir := t.TempDir()
	gitInitWithRemote(t, dir)
	if err := execInit(dir, initParams{Yes: true}, nil, nil, doltRepoFn); err != nil {
		t.Fatalf("execInit: %v", err)
	}

	if err := dispatch([]string{"insert",
		"--content", "コミット前に全テストを通す",
		"--trigger", "コミット時",
		"--domain", "git",
		"--count", "3",
	}, dir, nil, io.Discard); err != nil {
		t.Fatalf("dispatch insert: %v", err)
	}

	// パーソナルブランチでの件数を確認（openProjectConnが checkout する）
	countOnPersonalBranch := func() int {
		t.Helper()
		var capturedConn *sql.Conn
		_, _, cleanup, err := openProjectConn(dir, func(conn *sql.Conn) Repository {
			capturedConn = conn
			return doltRepoFn(conn)
		})
		if err != nil {
			t.Fatalf("openProjectConn: %v", err)
		}
		defer cleanup()
		var n int
		if err := capturedConn.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM dolt_log").Scan(&n); err != nil {
			t.Fatalf("dolt_log: %v", err)
		}
		return n
	}

	before := countOnPersonalBranch()

	if err := dispatch([]string{"commit", "-m", "test batch"}, dir, nil, io.Discard); err != nil {
		t.Fatalf("dispatch commit: %v", err)
	}

	after := countOnPersonalBranch()
	if after != before+1 {
		t.Errorf("expected commit count +1, before=%d after=%d", before, after)
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
