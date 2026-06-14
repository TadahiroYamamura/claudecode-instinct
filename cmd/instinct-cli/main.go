package main

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/alecthomas/kong"

	doltrepo "github.com/TadahiroYamamura/claudecode-instinct/cmd/instinct-cli/internal/dolt"
)

var defaultRepoFn = func(conn *sql.Conn) Repository {
	return doltrepo.NewRepository(conn)
}

func openProjectConn(cwd string, repoFn func(*sql.Conn) Repository) (Repository, string, func(), error) {
	projectDir, err := findProjectDirFrom(cwd)
	if err != nil {
		return nil, "", nil, err
	}

	userCfg, err := loadUserConfig(instinctDbDir(projectDir))
	if err != nil {
		return nil, "", nil, fmt.Errorf("setup not complete: %w", err)
	}

	conn, cleanup, err := openConn(context.Background(), instinctDataDir(projectDir))
	if err != nil {
		return nil, "", nil, err
	}

	repo := repoFn(conn)
	if err := repo.Checkout(context.Background(), userCfg.Dolt.Branch); err != nil {
		cleanup()
		return nil, "", nil, fmt.Errorf("checkout %s: %w", userCfg.Dolt.Branch, err)
	}

	return repo, projectDir, cleanup, nil
}

type initCmd struct {
	Yes        bool   `kong:"short='y',help='Accept all defaults without prompting'"`
	Branch     string `kong:"name='branch',short='b',help='Personal branch name (default: git config user.name)'"`
	TeamBranch string `kong:"name='team-branch',help='Team branch name (default: main)'"`
}

type connectCmd struct {
	Yes       bool   `kong:"short='y',help='Accept all defaults without prompting'"`
	Branch    string `kong:"name='branch',short='b',help='Personal branch name (default: git config user.name)'"`
	RemoteURL string `kong:"name='remote-url',short='r',help='Remote URL'"`
	Refs      string `kong:"name='refs',help='Dolt refs namespace (e.g. refs/dolt/myproject)'"`
}

type listCmd struct {
	Merged bool `kong:"name='merged',help='Include main branch instincts (deduped by ID)'"`
}

type showCmd struct {
	ID string `arg:"" name:"id" help:"Short ID (first 8 chars) of the instinct"`
}

type commitCmd struct {
	Message string `kong:"name='message',short='m',default='observer: batch commit',help='Commit message'"`
}

type dedupCmd struct{}
type nominateCmd struct{}
type pushCmd struct{}
type pullCmd struct{}

type cliStruct struct {
	Init     initCmd     `cmd:"" help:"Initialize .instinct-db locally (no remote required)"`
	Connect  connectCmd  `cmd:"" help:"Connect .instinct-db to a remote (push or clone)"`
	Insert   insertFlags `cmd:"" help:"Insert an instinct"`
	List     listCmd     `cmd:"" help:"List instincts"`
	Show     showCmd     `cmd:"" help:"Show full details of an instinct"`
	Commit   commitCmd   `cmd:"" help:"Commit working set to Dolt history"`
	Dedup    dedupCmd    `cmd:"" help:"Detect and merge duplicate instincts using Haiku"`
	Nominate nominateCmd `cmd:"" help:"Nominate instincts for team review (submit to review_queue)"`
	Push     pushCmd     `cmd:"" help:"Push personal branch to remote repository"`
	Pull     pullCmd     `cmd:"" help:"Pull team branch from remote repository"`
}

func instinctDbDir(projectDir string) string {
	return filepath.Join(projectDir, ".instinct-db")
}

func instinctDataDir(projectDir string) string {
	return filepath.Join(instinctDbDir(projectDir), "data")
}

func findProjectDirFrom(startDir string) (string, error) {
	// git rev-parse --show-toplevel はワークツリーごとに異なるルートを返す。
	// in-treeワークツリーが親ワークツリーの .instinct-db を誤発見しないよう
	// このルートを超えた探索を止める。
	gitRoot, _ := gitOutput(startDir, "rev-parse", "--show-toplevel")

	dir := startDir
	for {
		if _, err := os.Stat(instinctDbDir(dir)); err == nil {
			return dir, nil
		}
		if gitRoot != "" && dir == gitRoot {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf(".instinct-db not found in any parent directory")
}

func dispatch(args []string, cwd string, in io.Reader, out io.Writer) error {
	var cli cliStruct
	p, err := kong.New(&cli)
	if err != nil {
		return err
	}
	kctx, err := p.Parse(args)
	if err != nil {
		return err
	}
	ctx := context.Background()
	switch kctx.Command() {
	case "init":
		return execInit(cwd, initParams{Branch: cli.Init.Branch, TeamBranch: cli.Init.TeamBranch, Yes: cli.Init.Yes}, in, out, defaultRepoFn)
	case "connect":
		return execConnect(cwd, connectParams{Branch: cli.Connect.Branch, RemoteURL: cli.Connect.RemoteURL, Refs: cli.Connect.Refs, Yes: cli.Connect.Yes}, in, out, defaultDoltClone, defaultRepoFn)
	case "insert":
		repo, projectDir, cleanup, err := openProjectConn(cwd, defaultRepoFn)
		if err != nil {
			return err
		}
		defer cleanup()
		return execInsert(ctx, repo, cli.Insert, func(_ string) (string, error) {
			return resolveProjectID(projectDir)
		})
	case "list":
		repo, projectDir, cleanup, err := openProjectConn(cwd, defaultRepoFn)
		if err != nil {
			return err
		}
		defer cleanup()
		cfg, _ := loadConfig(instinctDbDir(projectDir))
		if cfg == nil {
			cfg = &InstinctConfig{}
		}
		if cli.List.Merged {
			return execListMerged(ctx, repo, cfg, out)
		}
		return execList(ctx, repo, out)
	case "show <id>":
		repo, _, cleanup, err := openProjectConn(cwd, defaultRepoFn)
		if err != nil {
			return err
		}
		defer cleanup()
		return execShow(ctx, repo, cli.Show.ID, out)
	case "commit":
		repo, _, cleanup, err := openProjectConn(cwd, defaultRepoFn)
		if err != nil {
			return err
		}
		defer cleanup()
		return execCommit(ctx, repo, cli.Commit.Message)
	case "dedup":
		repo, projectDir, cleanup, err := openProjectConn(cwd, defaultRepoFn)
		if err != nil {
			return err
		}
		defer cleanup()
		cfg, _ := loadConfig(instinctDbDir(projectDir))
		return execDedup(ctx, repo, haikuJudge, similarityThresholdFromConfig(cfg), out)
	case "nominate":
		repo, projectDir, cleanup, err := openProjectConn(cwd, defaultRepoFn)
		if err != nil {
			return err
		}
		defer cleanup()
		cfg, _ := loadConfig(instinctDbDir(projectDir))
		if cfg == nil {
			cfg = &InstinctConfig{}
		}
		userCfg, err := loadUserConfig(instinctDbDir(projectDir))
		if err != nil {
			return err
		}
		submittedBy, _ := gitConfigValue("user.name")
		return execNominate(ctx, repo, cfg, userCfg.Dolt.Branch, submittedBy, ttyNominateSelector, out)
	case "push":
		repo, projectDir, cleanup, err := openProjectConn(cwd, defaultRepoFn)
		if err != nil {
			return err
		}
		defer cleanup()
		cfg, err := loadConfig(instinctDbDir(projectDir))
		if err != nil {
			return err
		}
		userCfg, err := loadUserConfig(instinctDbDir(projectDir))
		if err != nil {
			return err
		}
		return execPush(ctx, repo, cfg, userCfg.Dolt.Branch, out)
	case "pull":
		repo, projectDir, cleanup, err := openProjectConn(cwd, defaultRepoFn)
		if err != nil {
			return err
		}
		defer cleanup()
		cfg, err := loadConfig(instinctDbDir(projectDir))
		if err != nil {
			return err
		}
		userCfg, err := loadUserConfig(instinctDbDir(projectDir))
		if err != nil {
			return err
		}
		return execPull(ctx, repo, cfg, userCfg.Dolt.Branch, out)
	default:
		return fmt.Errorf("unknown command: %s", kctx.Command())
	}
}

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := dispatch(os.Args[1:], cwd, os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
