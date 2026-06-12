package main

import (
	"bufio"
	"context"
	"io"
	"os"
	"path/filepath"
)

type initParams struct {
	Branch     string
	TeamBranch string
	Yes        bool
}

func execInit(projectDir string, params initParams, in io.Reader, out io.Writer) error {
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

	if err := setupDB(ctx, dataDir); err != nil {
		return err
	}

	conn, cleanup, err := openConn(ctx, dataDir)
	if err != nil {
		return err
	}
	defer cleanup()

	if _, err := conn.ExecContext(ctx, "CALL dolt_checkout('-b', ?)", branch); err != nil {
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
