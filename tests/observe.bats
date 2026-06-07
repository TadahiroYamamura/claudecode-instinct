#!/usr/bin/env bats

setup() {
  TMPDIR="$(mktemp -d)"
  export CLAUDE_PROJECT_DIR="$TMPDIR"
  OBSERVE_SH="$BATS_TEST_DIRNAME/../skills/continuous-learning/hooks/observe.sh"
}

teardown() {
  rm -rf "$TMPDIR"
}

@test "empty stdin writes nothing to observations.jsonl" {
  echo "" | bash "$OBSERVE_SH" post

  [ ! -f "$TMPDIR/observations.jsonl" ]
}

@test "PreToolUse hook records tool_start event" {
  local input='{"tool_name":"Bash","tool_input":{"command":"ls"},"session_id":"sess-1","cwd":"/tmp"}'

  echo "$input" | bash "$OBSERVE_SH" pre

  grep -q '"event": "tool_start"' "$TMPDIR/observations.jsonl"
}

@test "observation JSON contains tool name and session_id" {
  local input='{"tool_name":"Read","tool_input":{},"session_id":"sess-abc","cwd":"/tmp"}'

  echo "$input" | bash "$OBSERVE_SH" post

  grep -q '"tool": "Read"' "$TMPDIR/observations.jsonl"
  grep -q '"session": "sess-abc"' "$TMPDIR/observations.jsonl"
}

@test "valid PostToolUse JSON writes one observation to observations.jsonl" {
  local input='{"tool_name":"Bash","tool_input":{"command":"ls"},"tool_response":"file.txt","session_id":"sess-1","cwd":"/tmp"}'

  echo "$input" | bash "$OBSERVE_SH" post

  [ -f "$TMPDIR/observations.jsonl" ]
  [ "$(wc -l < "$TMPDIR/observations.jsonl")" -eq 1 ]
}
