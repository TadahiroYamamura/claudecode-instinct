package main

import (
	"context"
	"database/sql"
	"fmt"
	"io"

	"github.com/google/uuid"
)

type DedupDecision struct {
	Decision   string
	Reasoning  string
	Similarity float64
}

type DedupJudge func(ctx context.Context, a, b InstinctRow) (DedupDecision, error)

func insertDedupDecision(ctx context.Context, conn *sql.Conn, a, b InstinctRow, d DedupDecision) error {
	_, err := conn.ExecContext(ctx, `INSERT INTO dedup_decisions
		(id, instinct_id_a, instinct_id_b, content_a, content_b, trigger_a, trigger_b, decision, reasoning, similarity)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		uuid.New().String(), a.ID, b.ID, a.Content, b.Content, a.TriggerDesc, b.TriggerDesc,
		d.Decision, d.Reasoning, d.Similarity,
	)
	return err
}

func execDedup(ctx context.Context, conn *sql.Conn, judge DedupJudge, w io.Writer) error {
	instincts, err := listInstincts(ctx, conn)
	if err != nil {
		return err
	}

	pairs := 0
	for i := 0; i < len(instincts); i++ {
		for j := i + 1; j < len(instincts); j++ {
			d, err := judge(ctx, instincts[i], instincts[j])
			if err != nil {
				return fmt.Errorf("judge pair (%s, %s): %w", instincts[i].ID, instincts[j].ID, err)
			}
			if err := insertDedupDecision(ctx, conn, instincts[i], instincts[j], d); err != nil {
				return fmt.Errorf("record decision: %w", err)
			}
			pairs++
		}
	}

	fmt.Fprintf(w, "%d pairs checked\n", pairs)
	return nil
}
