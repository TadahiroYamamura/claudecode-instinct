package main

import (
	"context"
	"database/sql"
	"fmt"
)

type InstinctRow struct {
	Content string
}

func listInstincts(ctx context.Context, conn *sql.Conn) ([]InstinctRow, error) {
	rows, err := conn.QueryContext(ctx, "SELECT content FROM instincts ORDER BY created_at DESC")
	if err != nil {
		return nil, fmt.Errorf("list instincts: %w", err)
	}
	defer rows.Close()

	var result []InstinctRow
	for rows.Next() {
		var r InstinctRow
		if err := rows.Scan(&r.Content); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		result = append(result, r)
	}
	return result, rows.Err()
}
