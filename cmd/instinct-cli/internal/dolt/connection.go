package dolt

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	_ "github.com/dolthub/driver"
)

const dbName = "instincts"

func dsn(dataDir, commitName, commitEmail string) string {
	return "file://" + dataDir + "?commitname=" + commitName + "&commitemail=" + commitEmail + "&database=" + dbName
}

// OpenDB opens a raw Dolt SQL database at dataDir.
func OpenDB(dataDir, commitName, commitEmail string) (*sql.DB, error) {
	db, err := sql.Open("dolt", dsn(dataDir, commitName, commitEmail))
	if err != nil {
		return nil, fmt.Errorf("open dolt: %w", err)
	}
	return db, nil
}

// OpenConn opens a single Dolt connection pinned to the instincts database.
// The caller must call the returned cleanup func when done.
func OpenConn(ctx context.Context, dataDir, commitName, commitEmail string) (*sql.Conn, func(), error) {
	db, err := OpenDB(dataDir, commitName, commitEmail)
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

// SetupDB initializes a new Dolt database and schema in dataDir.
func SetupDB(ctx context.Context, dataDir, commitName, commitEmail string) error {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	db, err := OpenDB(dataDir, commitName, commitEmail)
	if err != nil {
		return err
	}
	defer db.Close()
	conn, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("get conn: %w", err)
	}
	defer conn.Close()
	stmts := append([]string{"CREATE DATABASE " + dbName, "USE " + dbName}, Schema()...)
	for _, stmt := range stmts {
		if _, err := conn.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("exec %q: %w", stmt[:min(len(stmt), 40)], err)
		}
	}
	return nil
}

// Clone clones a Dolt repository from remoteURL into dataDir.
func Clone(ctx context.Context, dataDir, refs, branch, remoteURL, commitName, commitEmail string) error {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	db, err := OpenDB(dataDir, commitName, commitEmail)
	if err != nil {
		return err
	}
	defer db.Close()
	conn, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("get conn: %w", err)
	}
	defer conn.Close()
	_, err = conn.ExecContext(ctx, "CALL dolt_clone('--ref', ?, '--branch', ?, ?, '.')", refs, branch, remoteURL)
	return err
}
