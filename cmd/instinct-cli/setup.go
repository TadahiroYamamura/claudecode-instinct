package main

import (
	"bytes"
	"context"
	_ "embed"
	"os"
	"path/filepath"
	"text/template"
)

//go:embed templates/gitignore.tmpl
var instinctDbGitignore []byte

//go:embed templates/config.tmpl
var configTemplate string

type configData struct {
	ProjectName string
	Branch      string
	RemoteURL   string
}

func runSetup(projectDir string) error {
	if err := setupDB(context.Background(), instinctDataDir(projectDir)); err != nil {
		return err
	}

	branch, err := gitConfigValue("user.name")
	if err != nil {
		return err
	}

	remoteURL, _ := gitOutput(projectDir, "remote", "get-url", "origin")

	dbDir := instinctDbDir(projectDir)

	var buf bytes.Buffer
	tmpl := template.Must(template.New("config").Parse(configTemplate))
	if err := tmpl.Execute(&buf, configData{
		ProjectName: filepath.Base(projectDir),
		Branch:      branch,
		RemoteURL:   remoteURL,
	}); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dbDir, "config.yml"), buf.Bytes(), 0o644); err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(dbDir, ".gitignore"), instinctDbGitignore, 0o644)
}
