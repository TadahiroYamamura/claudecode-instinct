package dolt

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"

	"github.com/TadahiroYamamura/claudecode-instinct/cmd/instinct-cli/internal/instincts"
)

type Repository struct {
	conn *sql.Conn
}

func NewRepository(conn *sql.Conn) *Repository {
	return &Repository{conn: conn}
}

func (r *Repository) InsertInstinct(ctx context.Context, p instincts.InsertParams) (string, error) {
	id := uuid.New().String()
	_, err := r.conn.ExecContext(ctx,
		`INSERT INTO instincts (id, content, trigger_desc, domain, scope, observation_count, project_id)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		id, p.Content, p.TriggerDesc, p.Domain, p.Scope, p.ObservationCount, p.ProjectID,
	)
	if err != nil {
		return "", fmt.Errorf("insert instinct: %w", err)
	}
	return id, nil
}
