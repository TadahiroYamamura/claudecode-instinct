package main

import (
	"bufio"
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"text/template"
)

//go:embed templates/gitignore.tmpl
var instinctDbGitignore []byte

//go:embed templates/config_team.tmpl
var configTeamTemplate string

//go:embed templates/config_user.tmpl
var configUserTemplate string

var configTeamTmpl = template.Must(template.New("config_team").Parse(configTeamTemplate))
var configUserTmpl = template.Must(template.New("config_user").Parse(configUserTemplate))

const defaultTeamBranch = "main"

type teamConfigData struct {
	ProjectName string
	TeamBranch  string
	RemoteURL   string
}

type userConfigData struct {
	Branch string
}

type doltCloneFunc func(ctx context.Context, dataDir, refs, branch, remoteURL string) error

var defaultDoltClone doltCloneFunc = func(ctx context.Context, dataDir, refs, branch, remoteURL string) error {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	db, err := openDoltDB(dataDir)
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

func runSetup(projectDir string, yes bool, in io.Reader, out io.Writer) error {
	return execSetup(projectDir, yes, in, out, defaultDoltClone, defaultDoltPush)
}

func execSetup(projectDir string, yes bool, in io.Reader, out io.Writer, cloneFn doltCloneFunc, pushFn doltPushFunc) error {
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

	if remoteURL == "" {
		return fmt.Errorf("remote_url is required: specify git remote origin or answer the prompt")
	}

	branch = sanitizeBranchName(branch)
	projectName := filepath.Base(projectDir)
	refs := "refs/dolt/" + projectName + "/"
	dbDir := instinctDbDir(projectDir)
	dataDir := instinctDataDir(projectDir)

	cloneErr := cloneFn(context.Background(), dataDir, refs, teamBranch, remoteURL)
	if cloneErr == nil {
		return setupClonePath(context.Background(), dbDir, dataDir, branch)
	}
	return setupInitPath(context.Background(), dbDir, dataDir, refs, projectName, branch, teamBranch, remoteURL, pushFn)
}

func setupClonePath(ctx context.Context, dbDir, dataDir, branch string) error {
	conn, cleanup, err := openConn(ctx, dataDir)
	if err != nil {
		return err
	}
	defer cleanup()

	var count int
	row := conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM dolt_remote_branches WHERE name = ?", "origin/"+branch)
	_ = row.Scan(&count)

	if count > 0 {
		_, _ = conn.ExecContext(ctx, "CALL dolt_checkout(?)", branch)
	} else {
		_, _ = conn.ExecContext(ctx, "CALL dolt_checkout('-b', ?)", branch)
	}

	return writeUserConfig(dbDir, branch)
}

func setupInitPath(ctx context.Context, dbDir, dataDir, refs, projectName, branch, teamBranch, remoteURL string, pushFn doltPushFunc) error {
	if err := setupDB(ctx, dataDir); err != nil {
		return err
	}

	conn, cleanup, err := openConn(ctx, dataDir)
	if err != nil {
		return err
	}
	defer cleanup()

	if _, err := conn.ExecContext(ctx, "CALL dolt_commit('-Am', 'init: create schema')"); err != nil {
		return fmt.Errorf("initial commit: %w", err)
	}

	ensureRemote(ctx, conn, refs, remoteURL)

	if err := pushFn(ctx, conn, "origin", teamBranch); err != nil {
		return fmt.Errorf("push team branch: %w", err)
	}

	_, _ = conn.ExecContext(ctx, "CALL dolt_checkout('-b', ?)", branch)

	if err := writeTeamConfig(dbDir, projectName, teamBranch, remoteURL); err != nil {
		return err
	}
	if err := writeUserConfig(dbDir, branch); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dbDir, ".gitignore"), instinctDbGitignore, 0o644)
}

func writeTeamConfig(dbDir, projectName, teamBranch, remoteURL string) error {
	var buf bytes.Buffer
	if err := configTeamTmpl.Execute(&buf, teamConfigData{
		ProjectName: projectName,
		TeamBranch:  teamBranch,
		RemoteURL:   remoteURL,
	}); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dbDir, "config.team.yml"), buf.Bytes(), 0o644)
}

func sanitizeBranchName(name string) string {
	var out []rune
	for _, r := range name {
		if r == ' ' || r == '/' || r == '\\' {
			out = append(out, '-')
		} else {
			out = append(out, r)
		}
	}
	return string(out)
}

func writeUserConfig(dbDir, branch string) error {
	var buf bytes.Buffer
	if err := configUserTmpl.Execute(&buf, userConfigData{Branch: branch}); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dbDir, "config.user.yml"), buf.Bytes(), 0o644)
}
