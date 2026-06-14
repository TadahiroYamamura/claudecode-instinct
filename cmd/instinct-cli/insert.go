package main

import (
	"context"
	"fmt"

	"github.com/alecthomas/kong"
)

type insertFlags struct {
	Content string `kong:"required,name='content',help='instinct content'"`
	Trigger string `kong:"name='trigger',help='trigger description'"`
	Domain  string `kong:"name='domain',help='domain'"`
	Count   int    `kong:"required,name='count',help='observation count'"`
	Scope   string `kong:"name='scope',default='project',help='scope (project|global)'"`
}

func runInsert(ctx context.Context, repo Repository, args []string, projectIDFn func(string) (string, error)) error {
	var f insertFlags
	p, err := kong.New(&f)
	if err != nil {
		return err
	}
	if _, err := p.Parse(args); err != nil {
		return err
	}
	return execInsert(ctx, repo, f, projectIDFn)
}

func execInsert(ctx context.Context, repo Repository, f insertFlags, projectIDFn func(string) (string, error)) error {
	projectID, err := projectIDFn("")
	if err != nil {
		return fmt.Errorf("project id: %w", err)
	}
	_, err = repo.InsertInstinct(ctx, InsertParams{
		Content:          f.Content,
		TriggerDesc:      f.Trigger,
		Domain:           f.Domain,
		Scope:            f.Scope,
		ObservationCount: f.Count,
		ProjectID:        projectID,
	})
	return err
}

