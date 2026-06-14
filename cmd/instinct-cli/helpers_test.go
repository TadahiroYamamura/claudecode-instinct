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

func listReviewInstincts(ctx context.Context, conn *sql.Conn, teamBranch string, minObservations int) ([]InstinctRow, error) {
	return doltrepo.NewRepository(conn).ListReviewInstincts(ctx, teamBranch, minObservations)
}

func submitToReviewQueue(ctx context.Context, conn *sql.Conn, teamBranch string, rows []InstinctRow, personalBranch, submittedBy string) error {
	return doltrepo.NewRepository(conn).SubmitToReviewQueue(ctx, teamBranch, rows, personalBranch, submittedBy)
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
	insertInstinct      func(ctx context.Context, p InsertParams) (string, error)
	listInstincts       func(ctx context.Context) ([]InstinctRow, error)
	getInstinct         func(ctx context.Context, shortID string) (*InstinctRow, error)
	listMergedInstincts  func(ctx context.Context, teamBranch string) ([]InstinctRow, error)
	listReviewInstincts  func(ctx context.Context, teamBranch string, minObservations int) ([]InstinctRow, error)
	insertDedupDecision  func(ctx context.Context, a, b InstinctRow, d DedupDecision, scores SimilarityScores) error
	mergeAndDelete       func(ctx context.Context, winner, loser InstinctRow) error
	commit               func(ctx context.Context, message string) error
	submitToReviewQueue  func(ctx context.Context, teamBranch string, rows []InstinctRow, personalBranch, submittedBy string) error
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

func (s *stubRepository) GetInstinct(ctx context.Context, shortID string) (*InstinctRow, error) {
	if s.getInstinct != nil {
		return s.getInstinct(ctx, shortID)
	}
	return nil, nil
}

func (s *stubRepository) ListMergedInstincts(ctx context.Context, teamBranch string) ([]InstinctRow, error) {
	if s.listMergedInstincts != nil {
		return s.listMergedInstincts(ctx, teamBranch)
	}
	return nil, nil
}

func (s *stubRepository) ListReviewInstincts(ctx context.Context, teamBranch string, minObservations int) ([]InstinctRow, error) {
	if s.listReviewInstincts != nil {
		return s.listReviewInstincts(ctx, teamBranch, minObservations)
	}
	return nil, nil
}

func (s *stubRepository) InsertDedupDecision(ctx context.Context, a, b InstinctRow, d DedupDecision, scores SimilarityScores) error {
	if s.insertDedupDecision != nil {
		return s.insertDedupDecision(ctx, a, b, d, scores)
	}
	return nil
}

func (s *stubRepository) MergeAndDelete(ctx context.Context, winner, loser InstinctRow) error {
	if s.mergeAndDelete != nil {
		return s.mergeAndDelete(ctx, winner, loser)
	}
	return nil
}

func (s *stubRepository) Commit(ctx context.Context, message string) error {
	if s.commit != nil {
		return s.commit(ctx, message)
	}
	return nil
}

func (s *stubRepository) SubmitToReviewQueue(ctx context.Context, teamBranch string, rows []InstinctRow, personalBranch, submittedBy string) error {
	if s.submitToReviewQueue != nil {
		return s.submitToReviewQueue(ctx, teamBranch, rows, personalBranch, submittedBy)
	}
	return nil
}
