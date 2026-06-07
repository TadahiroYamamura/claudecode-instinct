package main

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

// InsertParams holds the fields for a new instinct record.
type InsertParams struct {
	Content          string
	TriggerDesc      string
	Domain           string
	Scope            string
	ObservationCount int
	ProjectID        string
}

func insertInstinct(ctx context.Context, conn *sql.Conn, p InsertParams) error {
	id := uuid.New().String()
	_, err := conn.ExecContext(ctx,
		`INSERT INTO instincts (id, content, trigger_desc, domain, scope, observation_count, project_id)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		id, p.Content, p.TriggerDesc, p.Domain, p.Scope, p.ObservationCount, p.ProjectID,
	)
	if err != nil {
		return fmt.Errorf("insert instinct: %w", err)
	}
	return nil
}
