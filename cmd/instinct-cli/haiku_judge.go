package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

const dedupModel = "haiku"

type claudeRunner func(ctx context.Context, model, prompt string) (string, error)

var runClaude claudeRunner = func(ctx context.Context, model, prompt string) (string, error) {
	out, err := exec.CommandContext(ctx, "claude", "--model", model, "--print", prompt).Output()
	return string(out), err
}

func makeHaikuJudge(runner claudeRunner) DedupJudge {
	return func(ctx context.Context, a, b InstinctRow) (DedupDecision, error) {
		prompt := fmt.Sprintf(`以下の2つのinstinctが意味的に重複しているか判定してください。
JSONのみ返してください（説明文なし）:
{"decision":"duplicate"または"distinct","reasoning":"判定理由","similarity":0.0〜1.0}

## instinct A
content: %s
trigger: %s

## instinct B
content: %s
trigger: %s`, a.Content, a.TriggerDesc, b.Content, b.TriggerDesc)

		output, err := runner(ctx, dedupModel, prompt)
		if err != nil {
			return DedupDecision{}, fmt.Errorf("claude haiku: %w", err)
		}

		var d DedupDecision
		if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &d); err != nil {
			return DedupDecision{}, fmt.Errorf("parse haiku response: %w", err)
		}
		return d, nil
	}
}

var haikuJudge DedupJudge = makeHaikuJudge(runClaude)
