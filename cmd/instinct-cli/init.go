package main

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type initParams struct {
	Branch     string
	TeamBranch string
	Yes        bool
}

func execInit(projectDir string, params initParams, in io.Reader, out io.Writer, repoFn func(*sql.Conn) Repository) (finalErr error) {
	ctx := context.Background()
	defaultBranch, _ := gitConfigValue("user.name")

	var reader *bufio.Reader
	if in != nil {
		reader = bufio.NewReader(in)
	}
	resolve := func(explicit, defaultVal, label string) (string, error) {
		if explicit != "" {
			return explicit, nil
		}
		if params.Yes || reader == nil {
			return defaultVal, nil
		}
		return promptWithDefault(reader, out, label, defaultVal)
	}

	branch, err := resolve(params.Branch, defaultBranch, "Branch")
	if err != nil {
		return err
	}
	branch = sanitizeBranchName(branch)

	teamBranch, err := resolve(params.TeamBranch, defaultTeamBranch, "Team branch")
	if err != nil {
		return err
	}

	dbDir := instinctDbDir(projectDir)
	dataDir := instinctDataDir(projectDir)

	_, statErr := os.Stat(dbDir)
	dbDirIsNew := os.IsNotExist(statErr)

	if err := setupDB(ctx, dataDir); err != nil {
		return err
	}
	defer func() {
		if finalErr != nil {
			if dbDirIsNew {
				os.RemoveAll(dbDir) //nolint:errcheck
			} else {
				os.RemoveAll(dataDir) //nolint:errcheck
			}
		}
	}()

	conn, cleanup, err := openConn(ctx, dataDir)
	if err != nil {
		return err
	}
	defer cleanup()

	repo := repoFn(conn)

	if err := repo.Commit(ctx, "init: create schema"); err != nil {
		return fmt.Errorf("initial commit: %w", err)
	}

	if teamBranch != defaultTeamBranch {
		if err := repo.CreateBranch(ctx, teamBranch); err != nil {
			return err
		}
		if err := repo.Checkout(ctx, defaultTeamBranch); err != nil {
			return err
		}
	}

	if err := repo.CreateBranch(ctx, branch); err != nil {
		return err
	}

	if err := writeTeamConfig(dbDir, "", teamBranch, ""); err != nil {
		return err
	}
	if err := writeUserConfig(dbDir, branch); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dbDir, ".gitignore"), instinctDbGitignore, 0o644)
}
