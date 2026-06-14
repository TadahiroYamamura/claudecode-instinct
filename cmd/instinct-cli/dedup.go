package main

import (
	"context"
	"fmt"
	"io"
)

const (
	decisionDuplicate = "duplicate"
	decisionDistinct  = "distinct"
)

type DedupJudge func(ctx context.Context, a, b InstinctRow) (DedupDecision, error)

func execDedup(ctx context.Context, repo Repository, judge DedupJudge, threshold float64, w io.Writer) error {
	instincts, err := repo.ListInstincts(ctx)
	if err != nil {
		return err
	}

	pairs := 0
	for i := 0; i < len(instincts); i++ {
		for j := i + 1; j < len(instincts); j++ {
			a, b := instincts[i], instincts[j]
			scores := computeAllScores(a.Content, b.Content)
			if !anyAbove(scores, threshold) {
				continue
			}
			d, err := judge(ctx, a, b)
			if err != nil {
				return fmt.Errorf("judge pair (%s, %s): %w", a.ID, b.ID, err)
			}
			if err := repo.InsertDedupDecision(ctx, a, b, d, scores); err != nil {
				return fmt.Errorf("record decision: %w", err)
			}
			if d.Decision == decisionDuplicate {
				if err := repo.MergeAndDelete(ctx, a, b); err != nil {
					return fmt.Errorf("merge duplicate (%s, %s): %w", a.ID, b.ID, err)
				}
			}
			pairs++
		}
	}

	if pairs > 0 {
		msg := fmt.Sprintf("dedup: %d pairs checked", pairs)
		if err := repo.Commit(ctx, msg); err != nil {
			return fmt.Errorf("dolt_commit: %w", err)
		}
	}

	fmt.Fprintf(w, "%d pairs checked\n", pairs)
	return nil
}
