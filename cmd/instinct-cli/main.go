package main

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/alecthomas/kong"
)

func openProjectConn(cwd string) (*sql.Conn, string, func(), error) {
	projectDir, err := findProjectDirFrom(cwd)
	if err != nil {
		return nil, "", nil, err
	}
	conn, cleanup, err := openConn(context.Background(), instinctDataDir(projectDir))
	if err != nil {
		return nil, "", nil, err
	}
	return conn, projectDir, cleanup, nil
}

type setupCmd struct {
	Yes bool `kong:"short='y',help='Accept all defaults without prompting'"`
}

type listCmd struct {
	Merged bool `kong:"name='merged',help='Include main branch instincts (deduped by ID)'"`
}

type showCmd struct {
	ID string `arg:"" name:"id" help:"Short ID (first 8 chars) of the instinct"`
}

type cliStruct struct {
	Setup  setupCmd    `cmd:"" help:"Initialize .instinct-db in current directory"`
	Insert insertFlags `cmd:"" help:"Insert an instinct"`
	List   listCmd     `cmd:"" help:"List instincts"`
	Show   showCmd     `cmd:"" help:"Show full details of an instinct"`
}

func instinctDbDir(projectDir string) string {
	return filepath.Join(projectDir, ".instinct-db")
}

func instinctDataDir(projectDir string) string {
	return filepath.Join(instinctDbDir(projectDir), "data")
}

func findProjectDirFrom(startDir string) (string, error) {
	dir := startDir
	for {
		if _, err := os.Stat(instinctDbDir(dir)); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf(".instinct-db not found in any parent directory")
		}
		dir = parent
	}
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
	switch kctx.Command() {
	case "setup":
		return runSetup(cwd, cli.Setup.Yes, in, out)
	case "insert":
		conn, projectDir, cleanup, err := openProjectConn(cwd)
		if err != nil {
			return err
		}
		defer cleanup()
		return execInsert(context.Background(), conn, cli.Insert, func(_ string) (string, error) {
			return resolveProjectID(projectDir)
		})
	case "list":
		conn, projectDir, cleanup, err := openProjectConn(cwd)
		if err != nil {
			return err
		}
		defer cleanup()
		cfg, _ := loadConfig(instinctDbDir(projectDir))
		if cfg == nil {
			cfg = &InstinctConfig{}
		}
		if cli.List.Merged {
			return execListMerged(context.Background(), conn, cfg, os.Stdout)
		}
		return execList(context.Background(), conn, os.Stdout)
	case "show <id>":
		conn, _, cleanup, err := openProjectConn(cwd)
		if err != nil {
			return err
		}
		defer cleanup()
		return execShow(context.Background(), conn, cli.Show.ID, os.Stdout)
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
