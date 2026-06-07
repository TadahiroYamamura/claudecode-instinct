package main

import "context"

func runSetup(projectDir string) error {
	return setupDB(context.Background(), instinctDataDir(projectDir))
}
