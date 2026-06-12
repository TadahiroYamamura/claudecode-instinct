package main

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// fakeClone はリモートにチームブランチが存在するケース（cloneパス）をシミュレートする
func fakeClone(_ context.Context, dataDir, _, _, _ string) error {
	return setupDB(context.Background(), dataDir)
}

// fakeCloneFail はリモートにチームブランチが存在しないケース（initパス）をシミュレートする
func fakeCloneFail(_ context.Context, _ string, _, _, _ string) error {
	return fmt.Errorf("remote team branch not found")
}

func fakePush(_ context.Context, _ *sql.Conn, _, _ string) error { return nil }

// setup実行後に.instinct-db/data/が作成される
func TestSetup_CreatesDoltDBDirectory(t *testing.T) {
	dir := t.TempDir()
	gitInitWithRemote(t, dir)

	if err := execSetup(dir, setupParams{Yes: true}, nil, io.Discard, fakeCloneFail, fakePush); err != nil {
		t.Fatalf("execSetup: %v", err)
	}

	dataDir := filepath.Join(dir, ".instinct-db", "data")
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		t.Error(".instinct-db/data/ was not created")
	}
}

// setup実行後に.instinct-db/.gitignoreが作成され、ランタイムファイルとconfig.user.ymlが除外される
func TestSetup_CreatesGitignoreInInstinctDb(t *testing.T) {
	dir := t.TempDir()
	gitInitWithRemote(t, dir)

	if err := execSetup(dir, setupParams{Yes: true}, nil, io.Discard, fakeCloneFail, fakePush); err != nil {
		t.Fatalf("execSetup: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".instinct-db", ".gitignore"))
	if err != nil {
		t.Fatalf(".instinct-db/.gitignore not created: %v", err)
	}
	content := string(data)
	for _, entry := range []string{"data/", "observations.jsonl", ".observer.pid", "config.user.yml"} {
		if !strings.Contains(content, entry) {
			t.Errorf(".gitignore missing %q, got:\n%s", entry, content)
		}
	}
}

// initパス: config.user.ymlにdolt.branchが設定される
func TestSetup_InitPath_ConfigUserYmlContainsBranch(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "myproject")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	gitInitWithRemote(t, dir)
	if err := execSetup(dir, setupParams{Yes: true}, nil, io.Discard, fakeCloneFail, fakePush); err != nil {
		t.Fatalf("execSetup: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, ".instinct-db", "config.user.yml"))
	if err != nil {
		t.Fatalf("read config.user.yml: %v", err)
	}
	if !strings.Contains(string(data), "branch:") {
		t.Errorf("config.user.yml does not contain branch:, got:\n%s", data)
	}
}

// initパス: config.team.ymlにdolt.team_branchがmainに設定される
func TestSetup_InitPath_ConfigTeamYmlContainsTeamBranch(t *testing.T) {
	dir := t.TempDir()
	gitInitWithRemote(t, dir)
	if err := execSetup(dir, setupParams{Yes: true}, nil, io.Discard, fakeCloneFail, fakePush); err != nil {
		t.Fatalf("execSetup: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, ".instinct-db", "config.team.yml"))
	if err != nil {
		t.Fatalf("read config.team.yml: %v", err)
	}
	if !strings.Contains(string(data), "team_branch: main") {
		t.Errorf("config.team.yml does not contain team_branch: main, got:\n%s", data)
	}
}

// initパス: config.team.ymlにdolt.remote_urlがorigin remoteから設定される
func TestSetup_InitPath_ConfigTeamYmlContainsRemoteURL(t *testing.T) {
	dir := t.TempDir()
	mustRun(t, "git", "-C", dir, "init")
	mustRun(t, "git", "-C", dir, "remote", "add", "origin", "https://github.com/test/repo.git")

	if err := execSetup(dir, setupParams{Yes: true}, nil, io.Discard, fakeCloneFail, fakePush); err != nil {
		t.Fatalf("execSetup: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, ".instinct-db", "config.team.yml"))
	if err != nil {
		t.Fatalf("read config.team.yml: %v", err)
	}
	if !strings.Contains(string(data), "remote_url: https://github.com/test/repo.git") {
		t.Errorf("config.team.yml does not contain remote_url, got:\n%s", data)
	}
}

// initパス: config.team.ymlにはbranchが含まれない
func TestSetup_InitPath_ConfigTeamYmlDoesNotContainBranch(t *testing.T) {
	dir := t.TempDir()
	gitInitWithRemote(t, dir)
	if err := execSetup(dir, setupParams{Yes: true}, nil, io.Discard, fakeCloneFail, fakePush); err != nil {
		t.Fatalf("execSetup: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, ".instinct-db", "config.team.yml"))
	if err != nil {
		t.Fatalf("read config.team.yml: %v", err)
	}
	if strings.Contains(string(data), "\n  branch:") {
		t.Errorf("config.team.yml should NOT contain branch:, got:\n%s", data)
	}
}

func mustRun(t *testing.T, name string, args ...string) {
	t.Helper()
	if out, err := exec.Command(name, args...).CombinedOutput(); err != nil {
		t.Fatalf("%s %v: %v\n%s", name, args, err, out)
	}
}

func gitInitWithRemote(t *testing.T, dir string) {
	t.Helper()
	mustRun(t, "git", "-C", dir, "init")
	mustRun(t, "git", "-C", dir, "remote", "add", "origin", "https://github.com/test/repo.git")
}

// --yes フラグでプロンプトなしにデフォルト値でセットアップできる
func TestSetup_YesFlagSkipsPrompts(t *testing.T) {
	dir := t.TempDir()
	gitInitWithRemote(t, dir)

	if err := execSetup(dir, setupParams{Yes: true}, nil, io.Discard, fakeCloneFail, fakePush); err != nil {
		t.Fatalf("execSetup --yes: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".instinct-db", "data")); os.IsNotExist(err) {
		t.Error(".instinct-db/data/ was not created")
	}
}

// 対話入力でブランチ名を変更できる
func TestSetup_UsesInteractiveInputForBranch(t *testing.T) {
	dir := t.TempDir()
	// branch=custombranch、残りはデフォルト（team_branch=main、remote_url=空）
	in := strings.NewReader("custombranch\nmain\nhttps://github.com/test/repo.git\n")

	if err := execSetup(dir, setupParams{}, in, io.Discard, fakeCloneFail, fakePush); err != nil {
		t.Fatalf("execSetup: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, ".instinct-db", "config.user.yml"))
	if err != nil {
		t.Fatalf("read config.user.yml: %v", err)
	}
	if !strings.Contains(string(data), "branch: custombranch") {
		t.Errorf("config.user.yml should have custombranch, got:\n%s", data)
	}
}

// config.team.ymlのrefsがディレクトリ名から自動推定される
func TestSetup_ConfigTeamYmlContainsRefsBasedOnDirName(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "myproject")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	gitInitWithRemote(t, dir)

	if err := execSetup(dir, setupParams{Yes: true}, nil, io.Discard, fakeCloneFail, fakePush); err != nil {
		t.Fatalf("execSetup: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".instinct-db", "config.team.yml"))
	if err != nil {
		t.Fatalf("read config.team.yml: %v", err)
	}
	if !strings.Contains(string(data), "refs/dolt/myproject") {
		t.Errorf("config.team.yml does not contain refs/dolt/myproject, got:\n%s", data)
	}
}

// cloneパス: config.user.ymlは生成されるがconfig.team.ymlは生成されない
func TestSetup_ClonePath_WritesOnlyUserConfig(t *testing.T) {
	dir := t.TempDir()
	gitInitWithRemote(t, dir)

	if err := execSetup(dir, setupParams{Yes: true}, nil, io.Discard, fakeClone, fakePush); err != nil {
		t.Fatalf("execSetup: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, ".instinct-db", "config.user.yml")); os.IsNotExist(err) {
		t.Error("config.user.yml should exist in clone path")
	}
	if _, err := os.Stat(filepath.Join(dir, ".instinct-db", "config.team.yml")); !os.IsNotExist(err) {
		t.Error("config.team.yml should NOT be generated in clone path (it is Git-managed)")
	}
}

// sanitizeBranchNameはスペースをハイフンに置換する（Doltブランチ名はスペース不可）
func TestSanitizeBranchName_ReplacesSpacesWithHyphens(t *testing.T) {
	got := sanitizeBranchName("Tadahiro Yamamura")
	if got != "Tadahiro-Yamamura" {
		t.Errorf("expected Tadahiro-Yamamura, got %q", got)
	}
}

// remote_urlが空のままsetupを実行するとエラーになる
func TestSetup_ErrorWhenRemoteURLEmpty(t *testing.T) {
	dir := t.TempDir()
	// remote_urlを空にするため git remote を設定しない

	err := execSetup(dir, setupParams{Yes: true}, nil, io.Discard, fakeCloneFail, fakePush)
	if err == nil {
		t.Error("expected error when remote_url is empty")
	}
}

// フラグで全値を明示指定すれば--yesなしでもin=nilでセットアップできる
func TestSetup_ExplicitFlagsSkipAllPrompts(t *testing.T) {
	dir := t.TempDir()
	gitInitWithRemote(t, dir)

	params := setupParams{
		Branch:     "my-branch",
		TeamBranch: "team-main",
		RemoteURL:  "git@github.com:org/repo.git",
	}
	if err := execSetup(dir, params, nil, io.Discard, fakeCloneFail, fakePush); err != nil {
		t.Fatalf("execSetup: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".instinct-db", "config.user.yml"))
	if err != nil {
		t.Fatalf("read config.user.yml: %v", err)
	}
	if !strings.Contains(string(data), "branch: my-branch") {
		t.Errorf("expected branch: my-branch, got:\n%s", data)
	}
}

// 一部のフラグを指定すれば指定分だけプロンプトをスキップできる
func TestSetup_PartialFlagsPromptOnlyMissing(t *testing.T) {
	dir := t.TempDir()
	gitInitWithRemote(t, dir)

	// branch と remote_url だけ指定 → team_branch はプロンプト
	params := setupParams{
		Branch:    "my-branch",
		RemoteURL: "git@github.com:org/repo.git",
	}
	in := strings.NewReader("custom-team\n")
	if err := execSetup(dir, params, in, io.Discard, fakeCloneFail, fakePush); err != nil {
		t.Fatalf("execSetup: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".instinct-db", "config.team.yml"))
	if err != nil {
		t.Fatalf("read config.team.yml: %v", err)
	}
	if !strings.Contains(string(data), "team_branch: custom-team") {
		t.Errorf("expected team_branch: custom-team, got:\n%s", data)
	}
}
