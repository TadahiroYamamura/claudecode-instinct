package main

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type InstinctRow struct {
	Content          string
	TriggerDesc      string
	Domain           string
	ObservationCount int
	Scope            string
	CreatedAt        time.Time
}

func listInstincts(ctx context.Context, conn *sql.Conn) ([]InstinctRow, error) {
	rows, err := conn.QueryContext(ctx, "SELECT content, trigger_desc, domain, observation_count, scope, created_at FROM instincts ORDER BY created_at DESC")
	if err != nil {
		return nil, fmt.Errorf("list instincts: %w", err)
	}
	defer rows.Close()

	var result []InstinctRow
	for rows.Next() {
		var r InstinctRow
		if err := rows.Scan(&r.Content, &r.TriggerDesc, &r.Domain, &r.ObservationCount, &r.Scope, &r.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		result = append(result, r)
	}
	return result, rows.Err()
}
