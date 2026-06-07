package main

import (
	"strings"
	"testing"
)

// doltDSNに渡したname/emailがDSNに含まれる
func TestDoltDSN_IncludesCommitNameAndEmail(t *testing.T) {
	dsn := doltDSN("/tmp/data", "Taro Yamada", "taro@example.com")

	if !strings.Contains(dsn, "Taro") {
		t.Errorf("DSN does not include commit name, got: %s", dsn)
	}
	if !strings.Contains(dsn, "taro@example.com") || strings.Contains(dsn, "instinct@local") {
		t.Errorf("DSN does not include commit email, got: %s", dsn)
	}
}
