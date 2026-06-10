package main

import (
	"bufio"
	"bytes"
	"context"
	_ "embed"
	"io"
	"os"
	"path/filepath"
	"text/template"
)

//go:embed templates/gitignore.tmpl
var instinctDbGitignore []byte

//go:embed templates/config.tmpl
var configTemplate string

var configTmpl = template.Must(template.New("config").Parse(configTemplate))

type configData struct {
	ProjectName string
	Branch      string
	TeamBranch  string
	RemoteURL   string
}

func runSetup(projectDir string, in io.Reader, out io.Writer) error {
	if err := setupDB(context.Background(), instinctDataDir(projectDir)); err != nil {
		return err
	}

	defaultBranch, _ := gitConfigValue("user.name")
	defaultRemote, _ := gitOutput(projectDir, "remote", "get-url", "origin")

	reader := bufio.NewReader(in)
	branch, err := promptWithDefault(reader, out, "Branch", defaultBranch)
	if err != nil {
		return err
	}
	teamBranch, err := promptWithDefault(reader, out, "Team branch", "main")
	if err != nil {
		return err
	}
	remoteURL, err := promptWithDefault(reader, out, "Remote URL", defaultRemote)
	if err != nil {
		return err
	}

	dbDir := instinctDbDir(projectDir)

	var buf bytes.Buffer
	if err := configTmpl.Execute(&buf, configData{
		ProjectName: filepath.Base(projectDir),
		Branch:      branch,
		TeamBranch:  teamBranch,
		RemoteURL:   remoteURL,
	}); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dbDir, "config.yml"), buf.Bytes(), 0o644); err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(dbDir, ".gitignore"), instinctDbGitignore, 0o644)
}
