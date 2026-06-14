package dolt

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/TadahiroYamamura/claudecode-instinct/cmd/instinct-cli/internal/instincts"
)

type Repository struct {
	conn *sql.Conn
}

func NewRepository(conn *sql.Conn) *Repository {
	return &Repository{conn: conn}
}

func (r *Repository) ListInstincts(ctx context.Context) ([]instincts.InstinctRow, error) {
	rows, err := r.conn.QueryContext(ctx,
		"SELECT id, content, trigger_desc, domain, observation_count, scope, created_at FROM instincts ORDER BY created_at DESC")
	if err != nil {
		return nil, fmt.Errorf("list instincts: %w", err)
	}
	defer rows.Close()

	var result []instincts.InstinctRow
	for rows.Next() {
		var row instincts.InstinctRow
		var createdAt time.Time
		if err := rows.Scan(&row.ID, &row.Content, &row.TriggerDesc, &row.Domain, &row.ObservationCount, &row.Scope, &createdAt); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		row.CreatedAt = createdAt
		result = append(result, row)
	}
	return result, rows.Err()
}

func (r *Repository) GetInstinct(ctx context.Context, shortID string) (*instincts.InstinctRow, error) {
	var row instincts.InstinctRow
	err := r.conn.QueryRowContext(ctx,
		"SELECT id, content, trigger_desc, domain, observation_count, scope, created_at FROM instincts WHERE id LIKE ?",
		shortID+"%",
	).Scan(&row.ID, &row.Content, &row.TriggerDesc, &row.Domain, &row.ObservationCount, &row.Scope, &row.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("instinct %q not found", shortID)
	}
	if err != nil {
		return nil, fmt.Errorf("get instinct: %w", err)
	}
	return &row, nil
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
