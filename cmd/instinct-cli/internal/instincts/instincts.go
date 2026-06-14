package instincts

import (
	"context"
	"time"
)

type InsertParams struct {
	Content          string
	TriggerDesc      string
	Domain           string
	Scope            string
	ObservationCount int
	ProjectID        string
}

type InstinctRow struct {
	ID               string
	Content          string
	TriggerDesc      string
	Domain           string
	ObservationCount int
	Scope            string
	CreatedAt        time.Time
}

type DedupDecision struct {
	Decision  string
	Reasoning string
}

type SimilarityScores struct {
	Bigram  float64
	Trigram float64
	Overlap float64
}

type Repository interface {
	InsertInstinct(ctx context.Context, p InsertParams) (string, error)
	ListInstincts(ctx context.Context) ([]InstinctRow, error)
	GetInstinct(ctx context.Context, shortID string) (*InstinctRow, error)
	ListMergedInstincts(ctx context.Context, teamBranch string) ([]InstinctRow, error)
	ListReviewInstincts(ctx context.Context, teamBranch string, minObservations int) ([]InstinctRow, error)
	InsertDedupDecision(ctx context.Context, a, b InstinctRow, d DedupDecision, scores SimilarityScores) error
	MergeAndDelete(ctx context.Context, winner, loser InstinctRow) error
	Commit(ctx context.Context, message string) error
	SubmitToReviewQueue(ctx context.Context, teamBranch string, rows []InstinctRow, personalBranch, submittedBy string) error
	Upload(ctx context.Context, remote, branch string) error
	Sync(ctx context.Context, remote, branch string) error
	EnsureRemote(ctx context.Context, refs, remoteURL string)
	Checkout(ctx context.Context, branch string) error
}
