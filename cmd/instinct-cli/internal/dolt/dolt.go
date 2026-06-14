package dolt

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
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

func (r *Repository) ListMergedInstincts(ctx context.Context, teamBranch string) ([]instincts.InstinctRow, error) {
	// AS OF はプレースホルダー非対応のため Sprintf で埋め込む。
	// teamBranch は config.yml 由来（ユーザー入力ではない）。
	query := fmt.Sprintf(`
		SELECT id, content, trigger_desc, domain, observation_count, scope, created_at FROM instincts
		UNION
		SELECT id, content, trigger_desc, domain, observation_count, scope, created_at FROM instincts AS OF '%s'
		ORDER BY created_at DESC`, teamBranch)
	rows, err := r.conn.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list merged instincts: %w", err)
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

func (r *Repository) ListReviewInstincts(ctx context.Context, teamBranch string, minObservations int) ([]instincts.InstinctRow, error) {
	// AS OF はプレースホルダー非対応のため Sprintf で埋め込む。
	// teamBranch は config.yml 由来（ユーザー入力ではない）。
	query := fmt.Sprintf(`
		SELECT id, content, trigger_desc, domain, observation_count, scope, created_at
		FROM instincts
		WHERE id NOT IN (SELECT id FROM instincts AS OF '%s')
		  AND observation_count >= ?
		ORDER BY created_at DESC`, teamBranch)
	rows, err := r.conn.QueryContext(ctx, query, minObservations)
	if err != nil {
		return nil, fmt.Errorf("list review instincts: %w", err)
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
	var createdAt time.Time
	err := r.conn.QueryRowContext(ctx,
		"SELECT id, content, trigger_desc, domain, observation_count, scope, created_at FROM instincts WHERE id LIKE ?",
		shortID+"%",
	).Scan(&row.ID, &row.Content, &row.TriggerDesc, &row.Domain, &row.ObservationCount, &row.Scope, &createdAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("instinct %q not found", shortID)
	}
	if err != nil {
		return nil, fmt.Errorf("get instinct: %w", err)
	}
	row.CreatedAt = createdAt
	return &row, nil
}

func (r *Repository) InsertDedupDecision(ctx context.Context, a, b instincts.InstinctRow, d instincts.DedupDecision, scores instincts.SimilarityScores) error {
	_, err := r.conn.ExecContext(ctx, `INSERT INTO dedup_decisions
		(id, instinct_id_a, instinct_id_b, content_a, content_b, trigger_a, trigger_b, decision, reasoning, sim_bigram, sim_trigram, sim_overlap)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		uuid.New().String(), a.ID, b.ID, a.Content, b.Content, a.TriggerDesc, b.TriggerDesc,
		d.Decision, d.Reasoning, scores.Bigram, scores.Trigram, scores.Overlap,
	)
	return err
}

func (r *Repository) MergeAndDelete(ctx context.Context, winner, loser instincts.InstinctRow) error {
	if _, err := r.conn.ExecContext(ctx,
		"UPDATE instincts SET observation_count = observation_count + ? WHERE id = ?",
		loser.ObservationCount, winner.ID,
	); err != nil {
		return err
	}
	_, err := r.conn.ExecContext(ctx, "DELETE FROM instincts WHERE id = ?", loser.ID)
	return err
}

func (r *Repository) SubmitToReviewQueue(ctx context.Context, teamBranch string, rows []instincts.InstinctRow, personalBranch, submittedBy string) error {
	if _, err := r.conn.ExecContext(ctx, "CALL dolt_checkout(?)", teamBranch); err != nil {
		return fmt.Errorf("checkout %s: %w", teamBranch, err)
	}
	defer r.conn.ExecContext(ctx, "CALL dolt_checkout(?)", personalBranch) //nolint:errcheck

	for _, row := range rows {
		_, err := r.conn.ExecContext(ctx, `
			INSERT INTO review_queue
			  (instinct_id, content, trigger_desc, domain, observation_count, scope, submitted_by)
			VALUES (?, ?, ?, ?, ?, ?, ?)
			ON DUPLICATE KEY UPDATE
			  submitted_by = VALUES(submitted_by),
			  submitted_at = CURRENT_TIMESTAMP`,
			row.ID, row.Content, row.TriggerDesc, row.Domain, row.ObservationCount, row.Scope, submittedBy)
		if err != nil {
			return fmt.Errorf("insert review_queue %s: %w", row.ID[:8], err)
		}
	}

	msg := fmt.Sprintf("review: submit %d instinct(s) by %s", len(rows), submittedBy)
	if _, err := r.conn.ExecContext(ctx, "CALL dolt_commit('-Am', ?)", msg); err != nil {
		if strings.Contains(err.Error(), "nothing to commit") {
			return nil
		}
		return fmt.Errorf("commit review_queue: %w", err)
	}
	return nil
}

func (r *Repository) Upload(ctx context.Context, remote, branch string) error {
	_, err := r.conn.ExecContext(ctx, "CALL dolt_push(?, ?)", remote, branch)
	return err
}

func (r *Repository) Sync(ctx context.Context, remote, branch string) error {
	_, err := r.conn.ExecContext(ctx, "CALL dolt_pull(?, ?)", remote, branch)
	return err
}

func (r *Repository) EnsureRemote(ctx context.Context, refs, remoteURL string) {
	r.conn.ExecContext(ctx, "CALL dolt_remote('add', '--ref', ?, 'origin', ?)", refs, remoteURL) //nolint:errcheck
}

func (r *Repository) Checkout(ctx context.Context, branch string) error {
	_, err := r.conn.ExecContext(ctx, "CALL dolt_checkout(?)", branch)
	return err
}

func (r *Repository) CreateBranch(ctx context.Context, branch string) error {
	_, err := r.conn.ExecContext(ctx, "CALL dolt_checkout('-b', ?)", branch)
	return err
}

func (r *Repository) Commit(ctx context.Context, message string) error {
	_, err := r.conn.ExecContext(ctx, "CALL dolt_commit('-Am', ?)", message)
	return err
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
