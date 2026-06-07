package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	_ "github.com/dolthub/driver"
)

const dbName = "instincts"

func doltDSN(dataDir, commitName, commitEmail string) string {
	return "file://" + dataDir + "?commitname=" + commitName + "&commitemail=" + commitEmail
}

func gitConfigValue(key, fallback string) string {
	out, err := gitOutput("", "config", key)
	if err != nil || out == "" {
		return fallback
	}
	return out
}

func doltDSNWithGitIdentity(dataDir string) string {
	return doltDSN(dataDir,
		gitConfigValue("user.name", "instinct-cli"),
		gitConfigValue("user.email", "instinct@local"))
}

const createInstinctsTable = `CREATE TABLE instincts (
	id                VARCHAR(64)   PRIMARY KEY,
	content           TEXT          NOT NULL,
	trigger_desc      TEXT          NOT NULL,
	domain            VARCHAR(128),
	source            ENUM('auto','manual') NOT NULL DEFAULT 'auto',
	scope             ENUM('project','global') NOT NULL DEFAULT 'project',
	project_id        VARCHAR(12)   NOT NULL,
	project_name      VARCHAR(256),
	observation_count INT           NOT NULL DEFAULT 0,
	created_at        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP,
	updated_at        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
)`

// openConn returns a single Dolt connection pinned to the instincts database.
// The caller must call the returned cleanup func when done.
func openConn(ctx context.Context, dataDir string) (*sql.Conn, func(), error) {
	db, err := sql.Open("dolt", doltDSNWithGitIdentity(dataDir))
	if err != nil {
		return nil, nil, fmt.Errorf("open dolt: %w", err)
	}

	conn, err := db.Conn(ctx)
	if err != nil {
		db.Close()
		return nil, nil, fmt.Errorf("get conn: %w", err)
	}
	if _, err := conn.ExecContext(ctx, "USE "+dbName); err != nil {
		conn.Close()
		db.Close()
		return nil, nil, fmt.Errorf("use %s: %w", dbName, err)
	}

	cleanup := func() {
		conn.Close()
		db.Close()
	}
	return conn, cleanup, nil
}

// setupDB initializes a new Dolt database and schema in dataDir.
func setupDB(ctx context.Context, dataDir string) error {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	db, err := sql.Open("dolt", doltDSNWithGitIdentity(dataDir))
	if err != nil {
		return fmt.Errorf("open dolt: %w", err)
	}
	defer db.Close()

	conn, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("get conn: %w", err)
	}
	defer conn.Close()

	for _, stmt := range []string{
		"CREATE DATABASE " + dbName,
		"USE " + dbName,
		createInstinctsTable,
	} {
		if _, err := conn.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("exec %q: %w", stmt[:min(len(stmt), 40)], err)
		}
	}
	return nil
}

