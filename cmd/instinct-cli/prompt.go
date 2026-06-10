package main

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

func promptWithDefault(r io.Reader, w io.Writer, label, defaultValue string) (string, error) {
	fmt.Fprintf(w, "%s [%s]: ", label, defaultValue)
	line, err := bufio.NewReader(r).ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return defaultValue, nil
	}
	return line, nil
}
