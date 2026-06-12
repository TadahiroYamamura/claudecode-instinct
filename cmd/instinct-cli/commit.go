package main

import (
	"context"
	"database/sql"
	"fmt"
)

func execCommit(ctx context.Context, conn *sql.Conn, message string) error {
	if message == "" {
		message = "observer: batch commit"
	}
	if _, err := conn.ExecContext(ctx, "CALL dolt_commit('-Am', ?)", message); err != nil {
		return fmt.Errorf("dolt_commit: %w", err)
	}
	return nil
}
