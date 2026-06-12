package main

import (
	"io"
	"os"
)

type initParams struct {
	Branch string
	Yes    bool
}

func execInit(projectDir string, _ initParams, _ io.Reader, _ io.Writer) error {
	return os.MkdirAll(instinctDataDir(projectDir), 0o755)
}
