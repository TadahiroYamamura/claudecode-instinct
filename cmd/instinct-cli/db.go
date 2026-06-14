package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	_ "github.com/dolthub/driver"

	doltrepo "github.com/TadahiroYamamura/claudecode-instinct/cmd/instinct-cli/internal/dolt"
)

const dbName = "instincts"

func doltDSN(dataDir, commitName, commitEmail string) string {
	return "file://" + dataDir + "?commitname=" + commitName + "&commitemail=" + commitEmail
}

func gitConfigValue(key string) (string, error) {
	out, err := gitOutput("", "config", key)
	if err != nil || out == "" {
		return "", fmt.Errorf("git config %s: not set (git initialized?)", key)
	}
	return out, nil
}

func doltDSNWithGitIdentity(dataDir string) (string, error) {
	name, err := gitConfigValue("user.name")
	if err != nil {
		return "", err
	}
	email, err := gitConfigValue("user.email")
	if err != nil {
		return "", err
	}
	return doltDSN(dataDir, name, email), nil
}


func openDoltDB(dataDir string) (*sql.DB, error) {
	dsn, err := doltDSNWithGitIdentity(dataDir)
	if err != nil {
		return nil, err
	}
	db, err := sql.Open("dolt", dsn)
	if err != nil {
		return nil, fmt.Errorf("open dolt: %w", err)
	}
	return db, nil
}

// openConn returns a single Dolt connection pinned to the instincts database.
// The caller must call the returned cleanup func when done.
func openConn(ctx context.Context, dataDir string) (*sql.Conn, func(), error) {
	db, err := openDoltDB(dataDir)
	if err != nil {
		return nil, nil, err
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

	db, err := openDoltDB(dataDir)
	if err != nil {
		return err
	}
	defer db.Close()

	conn, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("get conn: %w", err)
	}
	defer conn.Close()

	stmts := append([]string{"CREATE DATABASE " + dbName, "USE " + dbName}, doltrepo.Schema()...)
	for _, stmt := range stmts {
		if _, err := conn.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("exec %q: %w", stmt[:min(len(stmt), 40)], err)
		}
	}
	return nil
}

