package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// dispatch(["init"])がexecInitにルーティングされ.instinct-db/dataを作成する
func TestDispatch_InitCommand_CreatesInstinctDb(t *testing.T) {
	dir := t.TempDir()
	mustRun(t, "git", "-C", dir, "init")

	if err := dispatch([]string{"init", "-y"}, dir, nil, io.Discard); err != nil {
		t.Fatalf("dispatch init: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, ".instinct-db", "data")); os.IsNotExist(err) {
		t.Error(".instinct-db/data/ was not created")
	}
}

// initはgit remoteが未設定でも.instinct-db/dataを作成できる
func TestInit_CreatesDoltDBWithoutRemote(t *testing.T) {
	dir := t.TempDir()
	mustRun(t, "git", "-C", dir, "init")

	if err := execInit(dir, initParams{Yes: true}, nil, io.Discard); err != nil {
		t.Fatalf("execInit: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, ".instinct-db", "data")); os.IsNotExist(err) {
		t.Error(".instinct-db/data/ was not created")
	}
}

// initはconfig.team.ymlを生成する
// リモート関連のdolt.refsとdolt.remote_urlは空、その他の設定はデフォルト値で埋まる
func TestInit_WritesTeamConfig(t *testing.T) {
	dir := t.TempDir()
	mustRun(t, "git", "-C", dir, "init")

	if err := execInit(dir, initParams{Yes: true}, nil, io.Discard); err != nil {
		t.Fatalf("execInit: %v", err)
	}

	cfg, err := loadConfig(filepath.Join(dir, ".instinct-db"))
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}

	if cfg.Observer.Enabled != true {
		t.Errorf("observer.enabled: got %v, want true", cfg.Observer.Enabled)
	}
	if cfg.Observer.TriggerEvery != 20 {
		t.Errorf("observer.trigger_every: got %d, want 20", cfg.Observer.TriggerEvery)
	}
	if cfg.Observer.ActiveHours != "800-2300" {
		t.Errorf("observer.active_hours: got %q, want 800-2300", cfg.Observer.ActiveHours)
	}
	if cfg.Confidence.ReviewMin != 6 {
		t.Errorf("confidence.review_min: got %d, want 6", cfg.Confidence.ReviewMin)
	}
	if cfg.Dedup.AutoRunBeforePush != false {
		t.Errorf("dedup.auto_run_before_push: got %v, want false", cfg.Dedup.AutoRunBeforePush)
	}
	if cfg.Dedup.SimilarityThreshold != 0.15 {
		t.Errorf("dedup.similarity_threshold: got %v, want 0.15", cfg.Dedup.SimilarityThreshold)
	}
	if cfg.Dolt.Refs != "" {
		t.Errorf("dolt.refs should be empty, got %q", cfg.Dolt.Refs)
	}
	if cfg.Dolt.TeamBranch != "main" {
		t.Errorf("dolt.team_branch: got %q, want main", cfg.Dolt.TeamBranch)
	}
	if cfg.Dolt.RemoteURL != "" {
		t.Errorf("dolt.remote_url should be empty, got %q", cfg.Dolt.RemoteURL)
	}
}

// initは.instinct-db/.gitignoreを生成する
func TestInit_CreatesGitignore(t *testing.T) {
	dir := t.TempDir()
	mustRun(t, "git", "-C", dir, "init")

	if err := execInit(dir, initParams{Yes: true}, nil, io.Discard); err != nil {
		t.Fatalf("execInit: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".instinct-db", ".gitignore"))
	if err != nil {
		t.Fatalf(".instinct-db/.gitignore not created: %v", err)
	}
	if string(data) != string(instinctDbGitignore) {
		t.Errorf(".gitignore content mismatch\n got:  %q\n want: %q", data, instinctDbGitignore)
	}
}

// initはconfig.user.ymlにdolt.branchを書き込む
func TestInit_WritesUserConfig(t *testing.T) {
	dir := t.TempDir()
	mustRun(t, "git", "-C", dir, "init")

	if err := execInit(dir, initParams{Branch: "alice", Yes: true}, nil, io.Discard); err != nil {
		t.Fatalf("execInit: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".instinct-db", "config.user.yml"))
	if err != nil {
		t.Fatalf("read config.user.yml: %v", err)
	}
	if !strings.Contains(string(data), "branch: alice") {
		t.Errorf("config.user.yml should contain branch: alice, got:\n%s", data)
	}
}
