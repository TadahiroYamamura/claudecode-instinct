package main

import (
	"io"
	"os"
	"path/filepath"
)

type initParams struct {
	Branch string
	Yes    bool
}

func execInit(projectDir string, params initParams, _ io.Reader, _ io.Writer) error {
	if err := os.MkdirAll(instinctDataDir(projectDir), 0o755); err != nil {
		return err
	}
	branch := params.Branch
	if branch == "" {
		branch, _ = gitConfigValue("user.name")
	}
	dbDir := instinctDbDir(projectDir)
	if err := writeTeamConfig(dbDir, "", defaultTeamBranch, ""); err != nil {
		return err
	}
	if err := writeUserConfig(dbDir, sanitizeBranchName(branch)); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dbDir, ".gitignore"), instinctDbGitignore, 0o644)
}
