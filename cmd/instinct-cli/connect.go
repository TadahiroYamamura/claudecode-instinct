package main

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	doltrepo "github.com/TadahiroYamamura/claudecode-instinct/cmd/instinct-cli/internal/dolt"
)

// convertRemoteURL converts SCP-style git URLs (git@host:path) to git+ssh format
// (git+ssh://git@host/path) which Dolt requires for SSH remotes.
func convertRemoteURL(url string) string {
	if strings.HasPrefix(url, "git@") && !strings.Contains(url, "://") {
		url = strings.Replace(url, ":", "/", 1)
		return "git+ssh://" + url
	}
	return url
}

type doltCloneFunc func(ctx context.Context, dataDir, refs, branch, remoteURL string) error

var defaultDoltClone doltCloneFunc = func(ctx context.Context, dataDir, refs, branch, remoteURL string) error {
	name, email, err := gitIdentity()
	if err != nil {
		return err
	}
	return doltrepo.Clone(ctx, dataDir, refs, branch, remoteURL, name, email)
}

type connectParams struct {
	RemoteURL string
	Refs      string
	Branch    string
	Yes       bool
}

func execConnect(projectDir string, params connectParams, in io.Reader, out io.Writer, cloneFn doltCloneFunc, repoFn func(*sql.Conn) Repository) error {
	dbDir := instinctDbDir(projectDir)

	cfg, err := loadConfig(dbDir)
	if err != nil {
		return fmt.Errorf("config.team.yml not found: run 'instinct-cli init' first")
	}

	if cfg.Dolt.TeamBranch == "" {
		return fmt.Errorf("dolt.team_branch is not set in config.team.yml: please set it manually")
	}

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

	ctx := context.Background()
	dataDir := instinctDataDir(projectDir)

	if _, statErr := os.Stat(dataDir); os.IsNotExist(statErr) {
		// clone path（2人目）: ローカルDBが存在しない
		if err := cloneFn(ctx, dataDir, cfg.Dolt.Refs, cfg.Dolt.TeamBranch, cfg.Dolt.RemoteURL); err != nil {
			return err
		}
		defaultBranch, _ := gitConfigValue("user.name")
		branch, err := resolve(params.Branch, sanitizeBranchName(defaultBranch), "Branch")
		if err != nil {
			return err
		}
		cloneConn, cloneCleanup, err := openConn(ctx, dataDir)
		if err != nil {
			return err
		}
		defer cloneCleanup()
		var branchCount int
		if err := cloneConn.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM dolt_remote_branches WHERE name = ?", "origin/"+branch,
		).Scan(&branchCount); err != nil {
			return err
		}
		if branchCount > 0 {
			_, err = cloneConn.ExecContext(ctx, "CALL dolt_checkout(?)", branch)
		} else {
			_, err = cloneConn.ExecContext(ctx, "CALL dolt_checkout('-b', ?)", branch)
		}
		if err != nil {
			return err
		}
		return writeUserConfig(dbDir, branch)
	}

	// push path（1人目）: ローカルDBが存在する
	conn, cleanup, err := openConn(ctx, dataDir)
	if err != nil {
		return fmt.Errorf("local DB not found: run 'instinct-cli init' first")
	}
	defer cleanup()
	defaultRemoteURL, _ := gitOutput(projectDir, "remote", "get-url", "origin")
	remoteURL, err := resolve(params.RemoteURL, defaultRemoteURL, "Remote URL")
	if err != nil {
		return err
	}
	if remoteURL == "" {
		return fmt.Errorf("remote URL is not set: run 'git remote add origin <url>' to configure a remote")
	}
	remoteURL = convertRemoteURL(remoteURL)

	defaultRefs := "refs/dolt/" + filepath.Base(projectDir)
	refs, err := resolve(params.Refs, defaultRefs, "Refs")
	if err != nil {
		return err
	}

	repo := repoFn(conn)
	repo.EnsureRemote(ctx, refs, remoteURL)

	if err := repo.Upload(ctx, "origin", cfg.Dolt.TeamBranch); err != nil {
		return err
	}

	return writeTeamConfig(dbDir, refs, cfg.Dolt.TeamBranch, remoteURL)
}
