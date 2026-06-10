package main

import (
	"context"
	"database/sql"
	"fmt"
	"io"
)

type DedupDecision struct {
	Decision   string
	Reasoning  string
	Similarity float64
}

type DedupJudge func(ctx context.Context, a, b InstinctRow) (DedupDecision, error)

func execDedup(ctx context.Context, conn *sql.Conn, judge DedupJudge, w io.Writer) error {
	if _, err := listInstincts(ctx, conn); err != nil {
		return err
	}
	fmt.Fprintf(w, "0 pairs checked\n")
	return nil
}
