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

const defaultTeamBranch = "main"

type configData struct {
	ProjectName string
	Branch      string
	TeamBranch  string
	RemoteURL   string
}

func runSetup(projectDir string, yes bool, in io.Reader, out io.Writer) error {
	if err := setupDB(context.Background(), instinctDataDir(projectDir)); err != nil {
		return err
	}

	defaultBranch, _ := gitConfigValue("user.name")
	defaultRemote, _ := gitOutput(projectDir, "remote", "get-url", "origin")

	var branch, teamBranch, remoteURL string
	if yes {
		branch, teamBranch, remoteURL = defaultBranch, defaultTeamBranch, defaultRemote
	} else {
		reader := bufio.NewReader(in)
		var err error
		if branch, err = promptWithDefault(reader, out, "Branch", defaultBranch); err != nil {
			return err
		}
		if teamBranch, err = promptWithDefault(reader, out, "Team branch", defaultTeamBranch); err != nil {
			return err
		}
		if remoteURL, err = promptWithDefault(reader, out, "Remote URL", defaultRemote); err != nil {
			return err
		}
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
