package main

import (
	"context"
	"database/sql"
	"fmt"
	"io"

	"github.com/google/uuid"

	doltrepo "github.com/TadahiroYamamura/claudecode-instinct/cmd/instinct-cli/internal/dolt"
)

const (
	decisionDuplicate = "duplicate"
	decisionDistinct  = "distinct"
)

type DedupJudge func(ctx context.Context, a, b InstinctRow) (DedupDecision, error)

func insertDedupDecision(ctx context.Context, conn *sql.Conn, a, b InstinctRow, d DedupDecision, scores SimilarityScores) error {
	_, err := conn.ExecContext(ctx, `INSERT INTO dedup_decisions
		(id, instinct_id_a, instinct_id_b, content_a, content_b, trigger_a, trigger_b, decision, reasoning, sim_bigram, sim_trigram, sim_overlap)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		uuid.New().String(), a.ID, b.ID, a.Content, b.Content, a.TriggerDesc, b.TriggerDesc,
		d.Decision, d.Reasoning, scores.Bigram, scores.Trigram, scores.Overlap,
	)
	return err
}

func mergeAndDelete(ctx context.Context, conn *sql.Conn, winner, loser InstinctRow) error {
	if _, err := conn.ExecContext(ctx,
		"UPDATE instincts SET observation_count = observation_count + ? WHERE id = ?",
		loser.ObservationCount, winner.ID,
	); err != nil {
		return err
	}
	_, err := conn.ExecContext(ctx, "DELETE FROM instincts WHERE id = ?", loser.ID)
	return err
}

func execDedup(ctx context.Context, conn *sql.Conn, judge DedupJudge, threshold float64, w io.Writer) error {
	instincts, err := doltrepo.NewRepository(conn).ListInstincts(ctx)
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
			if err := insertDedupDecision(ctx, conn, a, b, d, scores); err != nil {
				return fmt.Errorf("record decision: %w", err)
			}
			if d.Decision == decisionDuplicate {
				if err := mergeAndDelete(ctx, conn, a, b); err != nil {
					return fmt.Errorf("merge duplicate (%s, %s): %w", a.ID, b.ID, err)
				}
			}
			pairs++
		}
	}

	if pairs > 0 {
		msg := fmt.Sprintf("dedup: %d pairs checked", pairs)
		if _, err := conn.ExecContext(ctx, "CALL dolt_commit('-Am', ?)", msg); err != nil {
			return fmt.Errorf("dolt_commit: %w", err)
		}
	}

	fmt.Fprintf(w, "%d pairs checked\n", pairs)
	return nil
}
