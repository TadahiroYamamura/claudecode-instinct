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
}
