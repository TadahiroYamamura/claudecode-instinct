package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"

	doltrepo "github.com/TadahiroYamamura/claudecode-instinct/cmd/instinct-cli/internal/dolt"
)

// 1人目ケース: -yフラグでgit remote originとプロジェクト名からのデフォルト値を使ってpushできる
func TestConnect_UsesDefaultsWithYesFlag(t *testing.T) {
	dir := t.TempDir()
	mustRun(t, "git", "-C", dir, "init")
	mustRun(t, "git", "-C", dir, "remote", "add", "origin", "git@github.com:test/repo.git")
	if err := execInit(dir, initParams{Yes: true}, nil, nil); err != nil {
		t.Fatalf("execInit: %v", err)
	}

	var uploaded bool
	repoFn := func(_ *sql.Conn) Repository {
		return &stubRepository{upload: func(_ context.Context, _, _ string) error {
			uploaded = true
			return nil
		}}
	}

	if err := execConnect(dir, connectParams{Yes: true}, nil, io.Discard, fakeCloneFail, repoFn); err != nil {
		t.Fatalf("execConnect: %v", err)
	}
	if !uploaded {
		t.Error("expected upload to be called with defaults")
	}
}

// 1人目ケース: 対話入力からremote-urlを受け取りconfig.team.ymlに保存する
func TestConnect_UsesInteractiveInputForRemoteURL(t *testing.T) {
	dir := t.TempDir()
	mustRun(t, "git", "-C", dir, "init")
	if err := execInit(dir, initParams{Yes: true}, nil, nil); err != nil {
		t.Fatalf("execInit: %v", err)
	}

	in := strings.NewReader("git@github.com:test/repo.git\n")
	if err := execConnect(dir, connectParams{}, in, io.Discard, fakeCloneFail, fakeRepoFn); err != nil {
		t.Fatalf("execConnect: %v", err)
	}

	cfg, err := loadConfig(instinctDbDir(dir))
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	if cfg.Dolt.RemoteURL != "git@github.com:test/repo.git" {
		t.Errorf("remote_url: got %q, want git@github.com:test/repo.git", cfg.Dolt.RemoteURL)
	}
}

// git remote originが未設定かつ--remote-url未指定（対話入力なし）の場合、git remoteが必要な旨のエラーを返す
func TestConnect_ErrorWhenRemoteURLMissingAndNoGitOrigin(t *testing.T) {
	dir := t.TempDir()
	mustRun(t, "git", "-C", dir, "init")
	if err := execInit(dir, initParams{Yes: true}, nil, nil); err != nil {
		t.Fatalf("execInit: %v", err)
	}

	err := execConnect(dir, connectParams{}, nil, io.Discard, fakeCloneFail, fakeRepoFn)
	const wantErr = "remote URL is not set: run 'git remote add origin <url>' to configure a remote"
	if err == nil || err.Error() != wantErr {
		t.Errorf("error: got %v, want %q", err, wantErr)
	}
}

// config.team.ymlのteam_branchが未設定の場合エラーを返す
func TestConnect_ErrorWhenTeamBranchNotSet(t *testing.T) {
	dir := t.TempDir()
	mustRun(t, "git", "-C", dir, "init")
	dbDir := instinctDbDir(dir)
	if err := os.MkdirAll(dbDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := writeTeamConfig(dbDir, "", "", ""); err != nil {
		t.Fatalf("writeTeamConfig: %v", err)
	}

	err := execConnect(dir, connectParams{
		RemoteURL: "https://github.com/test/repo.git",
		Refs:      "refs/dolt/myproject",
	}, nil, io.Discard, fakeCloneFail, fakeRepoFn)
	if err == nil {
		t.Error("expected error when team_branch is not set")
	}
}

// 1人目ケース: init済みDBをリモートにpushする
func TestConnect_PushesTeamBranchOnFirstUser(t *testing.T) {
	dir := t.TempDir()
	mustRun(t, "git", "-C", dir, "init")
	if err := execInit(dir, initParams{Yes: true}, nil, nil); err != nil {
		t.Fatalf("execInit: %v", err)
	}

	var uploaded bool
	repoFn := func(_ *sql.Conn) Repository {
		return &stubRepository{upload: func(_ context.Context, _, _ string) error {
			uploaded = true
			return nil
		}}
	}

	if err := execConnect(dir, connectParams{
		RemoteURL: "https://github.com/test/repo.git",
		Refs:      "refs/dolt/myproject",
	}, nil, io.Discard, fakeCloneFail, repoFn); err != nil {
		t.Fatalf("execConnect: %v", err)
	}

	if !uploaded {
		t.Error("expected upload to be called")
	}
}

// 1人目ケース: push後にconfig.team.ymlのrefs/remote_urlが更新される
func TestConnect_UpdatesTeamConfigAfterPush(t *testing.T) {
	dir := t.TempDir()
	mustRun(t, "git", "-C", dir, "init")
	if err := execInit(dir, initParams{Yes: true}, nil, nil); err != nil {
		t.Fatalf("execInit: %v", err)
	}

	if err := execConnect(dir, connectParams{
		RemoteURL: "https://github.com/test/repo.git",
		Refs:      "refs/dolt/myproject",
	}, nil, io.Discard, fakeCloneFail, fakeRepoFn); err != nil {
		t.Fatalf("execConnect: %v", err)
	}

	cfg, err := loadConfig(instinctDbDir(dir))
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	if cfg.Dolt.RemoteURL != "https://github.com/test/repo.git" {
		t.Errorf("remote_url: got %q, want %q", cfg.Dolt.RemoteURL, "https://github.com/test/repo.git")
	}
	if cfg.Dolt.Refs != "refs/dolt/myproject" {
		t.Errorf("refs: got %q, want %q", cfg.Dolt.Refs, "refs/dolt/myproject")
	}
}

// 1人目ケース: 指定のrefs/remote_urlでdoltリモートが登録される
func TestConnect_RegistersRemoteWithCorrectRefsAndURL(t *testing.T) {
	dir := t.TempDir()
	mustRun(t, "git", "-C", dir, "init")
	if err := execInit(dir, initParams{Yes: true}, nil, nil); err != nil {
		t.Fatalf("execInit: %v", err)
	}

	const remoteURL = "git+ssh://git@github.com/test/repo.git"
	const refs = "refs/dolt/myproject"

	realRepoFn := func(conn *sql.Conn) Repository {
		real := doltrepo.NewRepository(conn)
		return &stubRepository{
			ensureRemote: func(ctx context.Context, r, u string) { real.EnsureRemote(ctx, r, u) },
		}
	}
	if err := execConnect(dir, connectParams{
		RemoteURL: remoteURL,
		Refs:      refs,
	}, nil, io.Discard, fakeCloneFail, realRepoFn); err != nil {
		t.Fatalf("execConnect: %v", err)
	}

	conn, cleanup, err := openConn(t.Context(), instinctDataDir(dir))
	if err != nil {
		t.Fatalf("openConn: %v", err)
	}
	defer cleanup()

	var name, url, params string
	if err := conn.QueryRowContext(t.Context(),
		"SELECT name, url, params FROM dolt_remotes WHERE name = 'origin'",
	).Scan(&name, &url, &params); err != nil {
		t.Fatalf("query dolt_remotes: %v", err)
	}
	if url != remoteURL {
		t.Errorf("url: got %q, want %q", url, remoteURL)
	}
	var remoteParams struct {
		GitRef string `json:"git_ref"`
	}
	if err := json.Unmarshal([]byte(params), &remoteParams); err != nil {
		t.Fatalf("parse params JSON: %v (raw: %q)", err, params)
	}
	if remoteParams.GitRef != refs {
		t.Errorf("git_ref: got %q, want %q", remoteParams.GitRef, refs)
	}
}

// 2人目ケース: 対話入力からbranchを受け取りconfig.user.ymlに保存する
func TestConnect_UsesInteractiveInputForBranch(t *testing.T) {
	dir := t.TempDir()
	mustRun(t, "git", "-C", dir, "init")
	dbDir := instinctDbDir(dir)
	if err := os.MkdirAll(dbDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := writeTeamConfig(dbDir, "refs/dolt/myproject", "main", "git@github.com:test/repo.git"); err != nil {
		t.Fatalf("writeTeamConfig: %v", err)
	}

	fakeCloneWithDB := func(ctx context.Context, dataDir, refs, branch, remoteURL string) error {
		return setupDB(ctx, dataDir)
	}

	in := strings.NewReader("bob\n")
	if err := execConnect(dir, connectParams{}, in, io.Discard, fakeCloneWithDB, fakeRepoFn); err != nil {
		t.Fatalf("execConnect: %v", err)
	}

	userCfg, err := loadUserConfig(dbDir)
	if err != nil {
		t.Fatalf("loadUserConfig: %v", err)
	}
	if userCfg.Dolt.Branch != "bob" {
		t.Errorf("dolt.branch: got %q, want bob", userCfg.Dolt.Branch)
	}
}

// 2人目ケース: ローカルDBがない場合にcloneFnを呼ぶ
func TestConnect_ClonesWhenLocalDBAbsent(t *testing.T) {
	dir := t.TempDir()
	mustRun(t, "git", "-C", dir, "init")
	dbDir := instinctDbDir(dir)
	if err := os.MkdirAll(dbDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := writeTeamConfig(dbDir, "refs/dolt/myproject", "main", "git+ssh://git@github.com/test/repo.git"); err != nil {
		t.Fatalf("writeTeamConfig: %v", err)
	}

	var cloned bool
	captureClone := func(ctx context.Context, dataDir, refs, branch, remoteURL string) error {
		cloned = true
		return setupDB(ctx, dataDir)
	}

	if err := execConnect(dir, connectParams{}, nil, io.Discard, captureClone, fakeRepoFn); err != nil {
		t.Fatalf("execConnect: %v", err)
	}
	if !cloned {
		t.Error("expected clone to be called")
	}
}

// 2人目ケース: clone後にconfig.user.ymlが書かれる
func TestConnect_WritesUserConfigAfterClone(t *testing.T) {
	dir := t.TempDir()
	mustRun(t, "git", "-C", dir, "init")
	dbDir := instinctDbDir(dir)
	if err := os.MkdirAll(dbDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := writeTeamConfig(dbDir, "refs/dolt/myproject", "main", "git+ssh://git@github.com/test/repo.git"); err != nil {
		t.Fatalf("writeTeamConfig: %v", err)
	}

	fakeCloneNoop := func(ctx context.Context, dataDir, refs, branch, remoteURL string) error {
		return setupDB(ctx, dataDir)
	}

	if err := execConnect(dir, connectParams{Branch: "alice"}, nil, io.Discard, fakeCloneNoop, fakeRepoFn); err != nil {
		t.Fatalf("execConnect: %v", err)
	}

	userCfg, err := loadUserConfig(dbDir)
	if err != nil {
		t.Fatalf("loadUserConfig: %v", err)
	}
	if userCfg.Dolt.Branch != "alice" {
		t.Errorf("dolt.branch: got %q, want alice", userCfg.Dolt.Branch)
	}
}

// 2人目ケース: clone後にDoltで個人ブランチが作成される
func TestConnect_CreatesPersonalBranchAfterClone(t *testing.T) {
	dir := t.TempDir()
	mustRun(t, "git", "-C", dir, "init")
	dbDir := instinctDbDir(dir)
	if err := os.MkdirAll(dbDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := writeTeamConfig(dbDir, "refs/dolt/myproject", "main", "git+ssh://git@github.com/test/repo.git"); err != nil {
		t.Fatalf("writeTeamConfig: %v", err)
	}

	fakeCloneWithDB := func(ctx context.Context, dataDir, refs, branch, remoteURL string) error {
		return setupDB(ctx, dataDir)
	}

	if err := execConnect(dir, connectParams{Branch: "alice"}, nil, io.Discard, fakeCloneWithDB, fakeRepoFn); err != nil {
		t.Fatalf("execConnect: %v", err)
	}

	conn, cleanup, err := openConn(t.Context(), instinctDataDir(dir))
	if err != nil {
		t.Fatalf("openConn: %v", err)
	}
	defer cleanup()

	var count int
	if err := conn.QueryRowContext(t.Context(),
		"SELECT COUNT(*) FROM dolt_branches WHERE name = ?", "alice",
	).Scan(&count); err != nil {
		t.Fatalf("query dolt_branches: %v", err)
	}
	if count != 1 {
		t.Errorf("personal branch 'alice' not found in dolt_branches")
	}
}

// config.team.ymlが存在しない場合エラーを返す（init未実施を意味する）
func TestConnect_ErrorWhenTeamConfigAbsent(t *testing.T) {
	dir := t.TempDir()
	mustRun(t, "git", "-C", dir, "init")

	err := execConnect(dir, connectParams{
		RemoteURL: "https://github.com/test/repo.git",
		Refs:      "refs/dolt/myproject",
	}, nil, io.Discard, fakeCloneFail, fakeRepoFn)
	if err == nil {
		t.Error("expected error when config.team.yml is absent")
	}
}
