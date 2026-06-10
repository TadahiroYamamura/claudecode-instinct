package main

import (
	"strings"
	"testing"
)

// 入力が空のときdefaultValueを返す
func TestPromptWithDefault_ReturnsDefaultOnEmptyInput(t *testing.T) {
	r := strings.NewReader("\n")
	var w strings.Builder

	got, err := promptWithDefault(r, &w, "Branch", "tadahiro")
	if err != nil {
		t.Fatalf("promptWithDefault: %v", err)
	}
	if got != "tadahiro" {
		t.Errorf("expected tadahiro, got %q", got)
	}
}

// 入力があるときその値を返す
func TestPromptWithDefault_ReturnsEnteredValue(t *testing.T) {
	r := strings.NewReader("myname\n")
	var w strings.Builder

	got, err := promptWithDefault(r, &w, "Branch", "tadahiro")
	if err != nil {
		t.Fatalf("promptWithDefault: %v", err)
	}
	if got != "myname" {
		t.Errorf("expected myname, got %q", got)
	}
}

// プロンプト文字列が "ラベル [デフォルト値]: " の形式で出力される
func TestPromptWithDefault_ShowsLabelAndDefault(t *testing.T) {
	r := strings.NewReader("\n")
	var w strings.Builder

	promptWithDefault(r, &w, "Branch", "tadahiro") //nolint
	if w.String() != "Branch [tadahiro]: " {
		t.Errorf("expected %q, got %q", "Branch [tadahiro]: ", w.String())
	}
}
