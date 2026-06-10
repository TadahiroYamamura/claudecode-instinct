package main

import (
	"context"
	"database/sql"
	"fmt"
	"io"
)

func execShow(ctx context.Context, conn *sql.Conn, shortID string, w io.Writer) error {
	var r InstinctRow
	err := conn.QueryRowContext(ctx,
		"SELECT id, content, trigger_desc, domain, observation_count, scope, created_at FROM instincts WHERE id LIKE ?",
		shortID+"%",
	).Scan(&r.ID, &r.Content, &r.TriggerDesc, &r.Domain, &r.ObservationCount, &r.Scope, &r.CreatedAt)
	if err == sql.ErrNoRows {
		return fmt.Errorf("instinct %q not found", shortID)
	}
	if err != nil {
		return fmt.Errorf("show instinct: %w", err)
	}
	fmt.Fprintf(w, "id:      %s\n", r.ID)
	fmt.Fprintf(w, "content: %s\n", r.Content)
	fmt.Fprintf(w, "trigger: %s\n", r.TriggerDesc)
	fmt.Fprintf(w, "domain:  %s\n", r.Domain)
	fmt.Fprintf(w, "obs:     %d\n", r.ObservationCount)
	fmt.Fprintf(w, "scope:   %s\n", r.Scope)
	return nil
}
