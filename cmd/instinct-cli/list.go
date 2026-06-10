package main

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"text/tabwriter"
	"time"
)

type InstinctRow struct {
	ID               string
	Content          string
	TriggerDesc      string
	Domain           string
	ObservationCount int
	Scope            string
	CreatedAt        time.Time
}

func listInstincts(ctx context.Context, conn *sql.Conn) ([]InstinctRow, error) {
	rows, err := conn.QueryContext(ctx, "SELECT id, content, trigger_desc, domain, observation_count, scope, created_at FROM instincts ORDER BY created_at DESC")
	if err != nil {
		return nil, fmt.Errorf("list instincts: %w", err)
	}
	defer rows.Close()

	var result []InstinctRow
	for rows.Next() {
		var r InstinctRow
		if err := rows.Scan(&r.ID, &r.Content, &r.TriggerDesc, &r.Domain, &r.ObservationCount, &r.Scope, &r.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

const (
	contentMaxRunes = 40
	shortIDLen      = 8
)

func truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "..."
}

func execList(ctx context.Context, conn *sql.Conn, w io.Writer) error {
	rows, err := listInstincts(ctx, conn)
	if err != nil {
		return err
	}
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tCONTENT\tTRIGGER\tDOMAIN\tOBS\tSCOPE")
	for _, r := range rows {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%d\t%s\n",
			r.ID[:shortIDLen],
			truncate(r.Content, contentMaxRunes),
			r.TriggerDesc,
			r.Domain,
			r.ObservationCount,
			r.Scope,
		)
	}
	return tw.Flush()
}
