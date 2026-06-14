package main

import (
	"context"
	"database/sql"
	"fmt"

	doltrepo "github.com/TadahiroYamamura/claudecode-instinct/cmd/instinct-cli/internal/dolt"
)

func gitConfigValue(key string) (string, error) {
	out, err := gitOutput("", "config", key)
	if err != nil || out == "" {
		return "", fmt.Errorf("git config %s: not set (git initialized?)", key)
	}
	return out, nil
}

func gitIdentity() (name, email string, err error) {
	name, err = gitConfigValue("user.name")
	if err != nil {
		return
	}
	email, err = gitConfigValue("user.email")
	return
}

func openConn(ctx context.Context, dataDir string) (*sql.Conn, func(), error) {
	name, email, err := gitIdentity()
	if err != nil {
		return nil, nil, err
	}
	return doltrepo.OpenConn(ctx, dataDir, name, email)
}

func setupDB(ctx context.Context, dataDir string) error {
	name, email, err := gitIdentity()
	if err != nil {
		return err
	}
	return doltrepo.SetupDB(ctx, dataDir, name, email)
}
