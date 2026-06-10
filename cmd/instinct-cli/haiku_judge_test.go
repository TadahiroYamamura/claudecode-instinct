package main

import (
	"context"
	"testing"
)

// haikuJudgeはclaudeのJSON出力をDedupDecisionに変換する
func TestHaikuJudge_ParsesDuplicateDecision(t *testing.T) {
	mockRunner := func(_ context.Context, _, _ string) (string, error) {
		return `{"decision":"duplicate","reasoning":"同じ知見の言い換え","similarity":0.85}`, nil
	}
	judge := makeHaikuJudge(mockRunner)

	a := InstinctRow{Content: "テスト前にlintを通す", TriggerDesc: "テスト実行時"}
	b := InstinctRow{Content: "lintエラーを解消してからテストを走らせる", TriggerDesc: "テスト実行時"}

	d, err := judge(context.Background(), a, b)
	if err != nil {
		t.Fatalf("judge: %v", err)
	}
	if d.Decision != decisionDuplicate {
		t.Errorf("expected decision=%q, got %q", decisionDuplicate, d.Decision)
	}
	if d.Similarity != 0.85 {
		t.Errorf("expected similarity=0.85, got %f", d.Similarity)
	}
	if d.Reasoning != "同じ知見の言い換え" {
		t.Errorf("expected reasoning set, got %q", d.Reasoning)
	}
}
