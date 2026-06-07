package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/alecthomas/kong"
)

type cliStruct struct {
	Insert insertFlags `cmd:"" help:"Insert an instinct"`
}

func run(args []string, conn *sql.Conn, projectIDFn func(string) (string, error)) error {
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
	case "insert":
		return execInsert(context.Background(), conn, cli.Insert, projectIDFn)
	default:
		return fmt.Errorf("unknown command: %s", kctx.Command())
	}
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

func dispatch(args []string, cwd string) error {
	if len(args) > 0 && args[0] == "setup" {
		return runSetup(cwd)
	}
	projectDir, err := findProjectDirFrom(cwd)
	if err != nil {
		return err
	}
	conn, cleanup, err := openConn(context.Background(), instinctDataDir(projectDir))
	if err != nil {
		return err
	}
	defer cleanup()
	return run(args, conn, func(_ string) (string, error) {
		return resolveProjectID(projectDir)
	})
}

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := dispatch(os.Args[1:], cwd); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
