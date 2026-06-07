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

func findProjectDir() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
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

func main() {
	projectDir, err := findProjectDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	dataDir := instinctDataDir(projectDir)
	ctx := context.Background()
	conn, cleanup, err := openConn(ctx, dataDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer cleanup()

	if err := run(os.Args[1:], conn, func(_ string) (string, error) {
		return resolveProjectID(projectDir)
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
