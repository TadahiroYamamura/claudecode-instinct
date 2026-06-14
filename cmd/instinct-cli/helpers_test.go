package main

import (
	"context"
	"database/sql"
	"fmt"
	"os/exec"
	"testing"

	doltrepo "github.com/TadahiroYamamura/claudecode-instinct/cmd/instinct-cli/internal/dolt"
)

func insertInstinct(ctx context.Context, conn *sql.Conn, p InsertParams) (string, error) {
	return doltrepo.NewRepository(conn).InsertInstinct(ctx, p)
}

func listInstincts(ctx context.Context, conn *sql.Conn) ([]InstinctRow, error) {
	return doltrepo.NewRepository(conn).ListInstincts(ctx)
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

// fakeCloneFail はリモートにチームブランチが存在しないケース（initパス）をシミュレートする
func fakeCloneFail(_ context.Context, _ string, _, _, _ string) error {
	return fmt.Errorf("remote team branch not found")
}

func fakePush(_ context.Context, _ *sql.Conn, _, _ string) error { return nil }

type stubRepository struct {
	insertInstinct func(ctx context.Context, p InsertParams) (string, error)
	listInstincts  func(ctx context.Context) ([]InstinctRow, error)
}

func (s *stubRepository) InsertInstinct(ctx context.Context, p InsertParams) (string, error) {
	if s.insertInstinct != nil {
		return s.insertInstinct(ctx, p)
	}
	return "stub-id", nil
}

func (s *stubRepository) ListInstincts(ctx context.Context) ([]InstinctRow, error) {
	if s.listInstincts != nil {
		return s.listInstincts(ctx)
	}
	return nil, nil
}
