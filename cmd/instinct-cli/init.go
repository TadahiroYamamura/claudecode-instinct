package main

import (
	"context"
	"io"
	"os"
	"path/filepath"
)

type initParams struct {
	Branch string
	Yes    bool
}

func execInit(projectDir string, params initParams, _ io.Reader, _ io.Writer) error {
	ctx := context.Background()

	branch := params.Branch
	if branch == "" {
		branch, _ = gitConfigValue("user.name")
	}
	branch = sanitizeBranchName(branch)

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

	if err := writeTeamConfig(dbDir, "", defaultTeamBranch, ""); err != nil {
		return err
	}
	if err := writeUserConfig(dbDir, branch); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dbDir, ".gitignore"), instinctDbGitignore, 0o644)
}
