package main

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/alecthomas/kong"
	"github.com/google/uuid"
)

type InsertParams struct {
	Content          string
	TriggerDesc      string
	Domain           string
	Scope            string
	ObservationCount int
	ProjectID        string
}

type insertFlags struct {
	Content string `kong:"required,name='content',help='instinct content'"`
	Trigger string `kong:"name='trigger',help='trigger description'"`
	Domain  string `kong:"name='domain',help='domain'"`
	Count   int    `kong:"required,name='count',help='observation count'"`
	Scope   string `kong:"name='scope',default='project',help='scope (project|global)'"`
}

func runInsert(ctx context.Context, conn *sql.Conn, args []string, projectIDFn func(string) (string, error)) error {
	var f insertFlags
	p, err := kong.New(&f)
	if err != nil {
		return err
	}
	if _, err := p.Parse(args); err != nil {
		return err
	}
	return execInsert(ctx, conn, f, projectIDFn)
}

func execInsert(ctx context.Context, conn *sql.Conn, f insertFlags, projectIDFn func(string) (string, error)) error {
	projectID, err := projectIDFn("")
	if err != nil {
		return fmt.Errorf("project id: %w", err)
	}
	return insertInstinct(ctx, conn, InsertParams{
		Content:          f.Content,
		TriggerDesc:      f.Trigger,
		Domain:           f.Domain,
		Scope:            f.Scope,
		ObservationCount: f.Count,
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
