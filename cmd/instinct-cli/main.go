package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/alecthomas/kong"
)

type setupCmd struct{}

type listCmd struct{}

type cliStruct struct {
	Setup  setupCmd    `cmd:"" help:"Initialize .instinct-db in current directory"`
	Insert insertFlags `cmd:"" help:"Insert an instinct"`
	List   listCmd     `cmd:"" help:"List instincts"`
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
		return runSetup(cwd)
	case "insert":
		projectDir, err := findProjectDirFrom(cwd)
		if err != nil {
			return err
		}
		conn, cleanup, err := openConn(context.Background(), instinctDataDir(projectDir))
		if err != nil {
			return err
		}
		defer cleanup()
		return execInsert(context.Background(), conn, cli.Insert, func(_ string) (string, error) {
			return resolveProjectID(projectDir)
		})
	case "list":
		projectDir, err := findProjectDirFrom(cwd)
		if err != nil {
			return err
		}
		conn, cleanup, err := openConn(context.Background(), instinctDataDir(projectDir))
		if err != nil {
			return err
		}
		defer cleanup()
		return execList(context.Background(), conn, os.Stdout)
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
	if err := dispatch(os.Args[1:], cwd); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
