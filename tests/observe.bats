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

@test "skips observation when CLAUDE_CODE_ENTRYPOINT is not an interactive entrypoint" {
  local input='{"tool_name":"Bash","session_id":"sess-1","cwd":"/tmp"}'

  echo "$input" | CLAUDE_CODE_ENTRYPOINT=api bash "$OBSERVE_SH" post

  [ ! -f "$TMPDIR/observations.jsonl" ]
}

@test "skips observation when agent_id is present (subagent session)" {
  local input='{"tool_name":"Bash","session_id":"sess-1","agent_id":"agent-123","cwd":"/tmp"}'

  echo "$input" | bash "$OBSERVE_SH" post

  [ ! -f "$TMPDIR/observations.jsonl" ]
}

@test "detects project dir from cwd git root when CLAUDE_PROJECT_DIR is unset" {
  local git_repo
  git_repo="$(mktemp -d)"
  git -C "$git_repo" init -q
  local subdir="$git_repo/src"
  mkdir -p "$subdir"

  local input="{\"tool_name\":\"Bash\",\"session_id\":\"sess-1\",\"cwd\":\"$subdir\"}"

  unset CLAUDE_PROJECT_DIR
  echo "$input" | bash "$OBSERVE_SH" post

  [ -f "$git_repo/observations.jsonl" ]
  rm -rf "$git_repo"
}

@test "valid PostToolUse JSON writes one observation to observations.jsonl" {
  local input='{"tool_name":"Bash","tool_input":{"command":"ls"},"tool_response":"file.txt","session_id":"sess-1","cwd":"/tmp"}'

  echo "$input" | bash "$OBSERVE_SH" post

  [ -f "$TMPDIR/observations.jsonl" ]
  [ "$(wc -l < "$TMPDIR/observations.jsonl")" -eq 1 ]
}
