//go:build e2e

package main

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"text/template"
)

const (
	e2eRemoteURL     = "git@github.com:TadahiroYamamura/claudecode-tdd.git"
	e2eRefsNamespace = "refs/dolt/e2e-instinct-test"
)

// TestE2E_FullSyncFlow は2ユーザー間の同期フローを検証する。
//
//	1人目: init→connect→insert×3→commit→push→list→show
//	2人目: connect(clone)→pull→list
//
// 実行: INSTINCT_E2E=1 go test -tags e2e -v -run TestE2E . (cmd/instinct-cli/ から)
func TestE2E_FullSyncFlow(t *testing.T) {
	if os.Getenv("INSTINCT_E2E") == "" {
		t.Skip("set INSTINCT_E2E=1 to run E2E tests")
	}

	dir := t.TempDir()
	mustRun(t, "git", "-C", dir, "init")
	mustRun(t, "git", "-C", dir, "remote", "add", "origin", e2eRemoteURL)

	t.Cleanup(func() {
		script := filepath.Join("..", "..", "scripts", "cleanup-dolt-remote.sh")
		cmd := exec.Command("bash", script, e2eRemoteURL, e2eRefsNamespace)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			t.Logf("cleanup warning: %v", err)
		}
	})

	t.Log("=== init ===")
	if err := dispatch([]string{"init", "--branch", "alice", "-y"}, dir, nil, os.Stdout); err != nil {
		t.Fatalf("init: %v", err)
	}

	t.Log("=== connect ===")
	if err := dispatch([]string{"connect",
		"--remote-url", e2eRemoteURL,
		"--refs", e2eRefsNamespace,
		"-y",
	}, dir, nil, os.Stdout); err != nil {
		t.Fatalf("connect: %v", err)
	}

	t.Log("=== insert x3 ===")
	testInstincts := []struct{ content, trigger, domain string }{
		{"TDDでテストを先に書く", "実装開始時", "development"},
		{"コミット前に全テストを通す", "git commit時", "git"},
		{"コードレビュー前にlintを通す", "PR作成時", "quality"},
	}
	for _, ins := range testInstincts {
		if err := dispatch([]string{"insert",
			"--content", ins.content,
			"--trigger", ins.trigger,
			"--domain", ins.domain,
			"--count", "1",
		}, dir, nil, io.Discard); err != nil {
			t.Fatalf("insert %q: %v", ins.content, err)
		}
	}

	t.Log("=== commit ===")
	if err := dispatch([]string{"commit", "-m", "e2e: 3つの知見を追加"}, dir, nil, io.Discard); err != nil {
		t.Fatalf("commit: %v", err)
	}

	t.Log("=== push ===")
	var pushBuf strings.Builder
	if err := dispatch([]string{"push"}, dir, nil, &pushBuf); err != nil {
		t.Fatalf("push: %v", err)
	}
	t.Log(pushBuf.String())
	assertWithTemplate(t, pushBuf.String(), "testdata/e2e_push.tmpl", struct{ Branch, RemoteURL string }{
		Branch:    "alice",
		RemoteURL: "git+ssh://git@github.com/TadahiroYamamura/claudecode-tdd.git",
	}, nil)

	t.Log("=== list ===")
	var listBuf strings.Builder
	if err := dispatch([]string{"list"}, dir, nil, &listBuf); err != nil {
		t.Fatalf("list: %v", err)
	}
	t.Log(listBuf.String())

	assertWithTemplate(t, listBuf.String(), "testdata/e2e_list.tmpl", struct{ TDD, Commit, Review string }{
		TDD:    findShortID(listBuf.String(), testInstincts[0].content),
		Commit: findShortID(listBuf.String(), testInstincts[1].content),
		Review: findShortID(listBuf.String(), testInstincts[2].content),
	}, sortDataRows)

	t.Log("=== show ===")
	shortID := findShortID(listBuf.String(), testInstincts[0].content)
	if shortID == "" {
		t.Fatalf("could not find shortID for %q in list output", testInstincts[0].content)
	}

	var showBuf strings.Builder
	if err := dispatch([]string{"show", shortID}, dir, nil, &showBuf); err != nil {
		t.Fatalf("show %s: %v", shortID, err)
	}
	t.Log(showBuf.String())

	assertWithTemplate(t, showBuf.String(), "testdata/e2e_show.tmpl", struct{ ID string }{
		ID: extractShowID(showBuf.String()),
	}, nil)

	// 2人目: config.team.yml を共有した状態で別マシンからアクセスするシナリオ
	t.Log("=== 2nd user: connect (clone) ===")
	dir2 := t.TempDir()
	mustRun(t, "git", "-C", dir2, "init")
	mustRun(t, "git", "-C", dir2, "remote", "add", "origin", e2eRemoteURL)
	mustCopyFile(t,
		filepath.Join(dir, ".instinct-db", "config.team.yml"),
		filepath.Join(dir2, ".instinct-db", "config.team.yml"),
	)
	if err := dispatch([]string{"connect", "--branch", "alice", "-y"}, dir2, nil, os.Stdout); err != nil {
		t.Fatalf("2nd user connect: %v", err)
	}

	t.Log("=== 2nd user: pull ===")
	var pullBuf strings.Builder
	if err := dispatch([]string{"pull"}, dir2, nil, &pullBuf); err != nil {
		t.Fatalf("2nd user pull: %v", err)
	}
	t.Log(pullBuf.String())
	assertWithTemplate(t, pullBuf.String(), "testdata/e2e_pull.tmpl", struct{ TeamBranch, Branch, RemoteURL string }{
		TeamBranch: "main",
		Branch:     "alice",
		RemoteURL:  "git+ssh://git@github.com/TadahiroYamamura/claudecode-tdd.git",
	}, nil)

	t.Log("=== 2nd user: list ===")
	var list2Buf strings.Builder
	if err := dispatch([]string{"list"}, dir2, nil, &list2Buf); err != nil {
		t.Fatalf("2nd user list: %v", err)
	}
	t.Log(list2Buf.String())
	assertWithTemplate(t, list2Buf.String(), "testdata/e2e_list.tmpl", struct{ TDD, Commit, Review string }{
		TDD:    findShortID(list2Buf.String(), testInstincts[0].content),
		Commit: findShortID(list2Buf.String(), testInstincts[1].content),
		Review: findShortID(list2Buf.String(), testInstincts[2].content),
	}, sortDataRows)
}

// mustCopyFile はファイルをコピーし、宛先ディレクトリを必要に応じて作成する。
func mustCopyFile(t *testing.T, src, dst string) {
	t.Helper()
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read %s: %v", src, err)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(dst), err)
	}
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		t.Fatalf("write %s: %v", dst, err)
	}
}

// findShortID はリスト出力から特定コンテンツを含む行の shortID を返す。
func findShortID(listOutput, content string) string {
	for _, line := range strings.Split(listOutput, "\n") {
		if strings.Contains(line, content) {
			if fields := strings.Fields(line); len(fields) > 0 {
				return fields[0]
			}
		}
	}
	return ""
}

// extractShowID は show 出力の "id: <uuid>" 行からフル UUID を取り出す。
func extractShowID(showOutput string) string {
	for _, line := range strings.Split(showOutput, "\n") {
		if strings.HasPrefix(line, "id: ") {
			return strings.TrimPrefix(line, "id: ")
		}
	}
	return ""
}

// sortDataRows はヘッダー行を保持しつつデータ行をソートする。
// created_at が同一秒内で非決定的なため、比較を順序非依存にする。
func sortDataRows(s string) string {
	lines := strings.Split(s, "\n")
	if len(lines) <= 1 {
		return s
	}
	header := lines[0]
	var dataLines []string
	for _, line := range lines[1:] {
		if strings.TrimSpace(line) != "" {
			dataLines = append(dataLines, line)
		}
	}
	sort.Strings(dataLines)
	return strings.Join(append([]string{header}, dataLines...), "\n")
}

// TestE2E_ReviewFlow はnominate→reviewの全フローを検証する。
//
//  1人目(alice): init→connect→insert×3(count=6)→commit→nominate→push(alice)→push(main)
//  2人目(bob):   connect(clone)→pull→review(全選択)→push(main)
//
// 実行: INSTINCT_E2E=1 go test -tags e2e -v -run TestE2E_ReviewFlow . (cmd/instinct-cli/ から)
func TestE2E_ReviewFlow(t *testing.T) {
	if os.Getenv("INSTINCT_E2E") == "" {
		t.Skip("set INSTINCT_E2E=1 to run E2E tests")
	}
	ctx := context.Background()

	dir := t.TempDir()
	mustRun(t, "git", "-C", dir, "init")
	mustRun(t, "git", "-C", dir, "remote", "add", "origin", e2eRemoteURL)

	t.Cleanup(func() {
		script := filepath.Join("..", "..", "scripts", "cleanup-dolt-remote.sh")
		cmd := exec.Command("bash", script, e2eRemoteURL, e2eRefsNamespace)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			t.Logf("cleanup warning: %v", err)
		}
	})

	// 1人目: alice (コントリビューター)
	t.Log("=== init ===")
	if err := dispatch([]string{"init", "--branch", "alice", "-y"}, dir, nil, os.Stdout); err != nil {
		t.Fatalf("init: %v", err)
	}

	t.Log("=== connect ===")
	if err := dispatch([]string{"connect",
		"--remote-url", e2eRemoteURL,
		"--refs", e2eRefsNamespace,
		"-y",
	}, dir, nil, os.Stdout); err != nil {
		t.Fatalf("connect: %v", err)
	}

	t.Log("=== insert x3 (count=6) ===")
	nominateInstincts := []struct{ content, trigger, domain string }{
		{"TDDでテストを先に書く", "実装開始時", "development"},
		{"コミット前に全テストを通す", "git commit時", "git"},
		{"コードレビュー前にlintを通す", "PR作成時", "quality"},
	}
	for _, ins := range nominateInstincts {
		if err := dispatch([]string{"insert",
			"--content", ins.content, "--trigger", ins.trigger,
			"--domain", ins.domain, "--count", "6",
		}, dir, nil, io.Discard); err != nil {
			t.Fatalf("insert %q: %v", ins.content, err)
		}
	}

	t.Log("=== commit ===")
	if err := dispatch([]string{"commit", "-m", "e2e: 知見を追加"}, dir, nil, io.Discard); err != nil {
		t.Fatalf("commit: %v", err)
	}

	// nominate: TTYセレクターをバイパスして全件選択
	t.Log("=== nominate ===")
	repo, projectDir, repoCleanup, err := openProjectConn(dir, defaultRepoFn)
	if err != nil {
		t.Fatalf("openProjectConn: %v", err)
	}
	defer repoCleanup()

	cfg, err := loadConfig(instinctDbDir(projectDir))
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}

	candidates, err := repo.ListReviewInstincts(ctx, "main", cfg.Confidence.ReviewMin)
	if err != nil {
		t.Fatalf("ListReviewInstincts: %v", err)
	}
	allIDs := make([]string, len(candidates))
	for i, r := range candidates {
		allIDs[i] = r.ID[:shortIDLen]
	}

	var nominateBuf strings.Builder
	if err := execNominate(ctx, repo, cfg, "alice", "alice", allIDs, &nominateBuf); err != nil {
		t.Fatalf("nominate: %v", err)
	}
	t.Log(nominateBuf.String())
	assertWithTemplate(t, nominateBuf.String(), "testdata/e2e_nominate.tmpl",
		struct {
			Count      int
			TeamBranch string
		}{Count: 3, TeamBranch: "main"}, nil)

	t.Log("=== push alice ===")
	var pushAliceBuf strings.Builder
	if err := execPush(ctx, repo, cfg, "alice", &pushAliceBuf); err != nil {
		t.Fatalf("push alice: %v", err)
	}
	assertWithTemplate(t, pushAliceBuf.String(), "testdata/e2e_push.tmpl",
		struct{ Branch, RemoteURL string }{Branch: "alice", RemoteURL: "git+ssh://git@github.com/TadahiroYamamura/claudecode-tdd.git"}, nil)

	// review_queue を含むチームブランチを push
	t.Log("=== push main (review_queue) ===")
	var pushMain1Buf strings.Builder
	if err := execPush(ctx, repo, cfg, "main", &pushMain1Buf); err != nil {
		t.Fatalf("push main: %v", err)
	}
	assertWithTemplate(t, pushMain1Buf.String(), "testdata/e2e_push.tmpl",
		struct{ Branch, RemoteURL string }{Branch: "main", RemoteURL: "git+ssh://git@github.com/TadahiroYamamura/claudecode-tdd.git"}, nil)

	// 2人目: bob (チームレビュアー)
	t.Log("=== 2nd user: connect (clone) ===")
	dir2 := t.TempDir()
	mustRun(t, "git", "-C", dir2, "init")
	mustRun(t, "git", "-C", dir2, "remote", "add", "origin", e2eRemoteURL)
	mustCopyFile(t,
		filepath.Join(dir, ".instinct-db", "config.team.yml"),
		filepath.Join(dir2, ".instinct-db", "config.team.yml"),
	)
	if err := dispatch([]string{"connect", "--branch", "bob", "-y"}, dir2, nil, os.Stdout); err != nil {
		t.Fatalf("2nd user connect: %v", err)
	}

	t.Log("=== 2nd user: pull ===")
	if err := dispatch([]string{"pull"}, dir2, nil, io.Discard); err != nil {
		t.Fatalf("2nd user pull: %v", err)
	}

	// review: review_queue から team ブランチへ昇格
	t.Log("=== 2nd user: review ===")
	repo2, projectDir2, repo2Cleanup, err := openProjectConn(dir2, defaultRepoFn)
	if err != nil {
		t.Fatalf("2nd user openProjectConn: %v", err)
	}
	defer repo2Cleanup()

	cfg2, err := loadConfig(instinctDbDir(projectDir2))
	if err != nil {
		t.Fatalf("2nd user loadConfig: %v", err)
	}

	queueItems, err := repo2.ListReviewQueue(ctx, "main")
	if err != nil {
		t.Fatalf("2nd user ListReviewQueue: %v", err)
	}
	allQueueIDs := make([]string, len(queueItems))
	for i, r := range queueItems {
		allQueueIDs[i] = r.InstinctID[:shortIDLen]
	}

	var reviewBuf strings.Builder
	if err := execReviewApprove(ctx, repo2, cfg2, "bob", "bob", allQueueIDs, &reviewBuf); err != nil {
		t.Fatalf("2nd user review: %v", err)
	}
	t.Log(reviewBuf.String())
	assertWithTemplate(t, reviewBuf.String(), "testdata/e2e_review.tmpl",
		struct {
			Count      int
			TeamBranch string
		}{Count: 3, TeamBranch: "main"}, nil)

	// 昇格後の main を push
	t.Log("=== 2nd user: push main (promoted) ===")
	var pushMain2Buf strings.Builder
	if err := execPush(ctx, repo2, cfg2, "main", &pushMain2Buf); err != nil {
		t.Fatalf("2nd user push main: %v", err)
	}
	assertWithTemplate(t, pushMain2Buf.String(), "testdata/e2e_push.tmpl",
		struct{ Branch, RemoteURL string }{Branch: "main", RemoteURL: "git+ssh://git@github.com/TadahiroYamamura/claudecode-tdd.git"}, nil)

	// review_queue が空になっていることを確認
	remaining, err := repo2.ListReviewQueue(ctx, "main")
	if err != nil {
		t.Fatalf("2nd user ListReviewQueue after approve: %v", err)
	}
	if len(remaining) != 0 {
		t.Errorf("2nd user: expected review_queue empty after promotion, got %d items", len(remaining))
	}
}

// assertWithTemplate はテンプレートに data を注入して expected を生成し、actual と比較する。
// normalize が non-nil の場合、両辺に適用してから比較する（例: ソート）。
func assertWithTemplate(t *testing.T, actual, tmplPath string, data any, normalize func(string) string) {
	t.Helper()

	tmplBytes, err := os.ReadFile(tmplPath)
	if err != nil {
		t.Fatalf("read template %s: %v", tmplPath, err)
	}
	tmpl, err := template.New("").Parse(string(tmplBytes))
	if err != nil {
		t.Fatalf("parse template %s: %v", tmplPath, err)
	}
	var expectedBuf strings.Builder
	if err := tmpl.Execute(&expectedBuf, data); err != nil {
		t.Fatalf("execute template %s: %v", tmplPath, err)
	}

	want := strings.TrimRight(expectedBuf.String(), "\n")
	got := strings.TrimRight(actual, "\n")
	if normalize != nil {
		want = normalize(want)
		got = normalize(got)
	}
	if got != want {
		t.Errorf("output mismatch for %s\ngot:\n%s\nwant:\n%s", tmplPath, actual, expectedBuf.String())
	}
}
