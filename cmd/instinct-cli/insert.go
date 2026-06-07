package main

import (
	"context"
	"database/sql"
	"flag"
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

func runInsert(ctx context.Context, conn *sql.Conn, args []string, projectIDFn func(string) (string, error)) error {
	fs := flag.NewFlagSet("insert", flag.ContinueOnError)
	content := fs.String("content", "", "instinct content")
	trigger := fs.String("trigger", "", "trigger description")
	domain := fs.String("domain", "", "domain")
	count := fs.Int("count", 0, "observation count")
	scope := fs.String("scope", "project", "scope (project|global)")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if *content == "" {
		return fmt.Errorf("--content is required")
	}

	projectID, err := projectIDFn("")
	if err != nil {
		return fmt.Errorf("project id: %w", err)
	}

	return insertInstinct(ctx, conn, InsertParams{
		Content:          *content,
		TriggerDesc:      *trigger,
		Domain:           *domain,
		Scope:            *scope,
		ObservationCount: *count,
		ProjectID:        projectID,
	})
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
