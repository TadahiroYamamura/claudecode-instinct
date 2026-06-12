package main

import (
	"io"
	"os"
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
	return writeUserConfig(instinctDbDir(projectDir), sanitizeBranchName(branch))
}
