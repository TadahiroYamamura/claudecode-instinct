#!/usr/bin/env bats

setup() {
  TMPDIR="$(mktemp -d)"
  mkdir -p "$TMPDIR/.instinct-db"
  OBSERVE_SH="$BATS_TEST_DIRNAME/../skills/continuous-learning/hooks/observe.sh"
}

teardown() {
  rm -rf "$TMPDIR"
}

@test "empty stdin writes nothing to observations.jsonl" {
  echo "" | bash "$OBSERVE_SH" post

  [ ! -f "$TMPDIR/.instinct-db/observations.jsonl" ]
}

@test "PreToolUse hook records tool_start event" {
  local input="{\"tool_name\":\"Bash\",\"tool_input\":{\"command\":\"ls\"},\"session_id\":\"sess-1\",\"cwd\":\"$TMPDIR\"}"

  echo "$input" | bash "$OBSERVE_SH" pre

  grep -q '"event": "tool_start"' "$TMPDIR/.instinct-db/observations.jsonl"
}

@test "observation JSON contains tool name and session_id" {
  local input="{\"tool_name\":\"Read\",\"tool_input\":{},\"session_id\":\"sess-abc\",\"cwd\":\"$TMPDIR\"}"

  echo "$input" | bash "$OBSERVE_SH" post

  grep -q '"tool": "Read"' "$TMPDIR/.instinct-db/observations.jsonl"
  grep -q '"session": "sess-abc"' "$TMPDIR/.instinct-db/observations.jsonl"
}

@test "skips observation when INSTINCT_SKIP_OBSERVE is set to 1" {
  local input="{\"tool_name\":\"Bash\",\"session_id\":\"sess-1\",\"cwd\":\"$TMPDIR\"}"

  echo "$input" | INSTINCT_SKIP_OBSERVE=1 bash "$OBSERVE_SH" post

  [ ! -f "$TMPDIR/.instinct-db/observations.jsonl" ]
}

@test "skips observation when CLAUDE_CODE_ENTRYPOINT is not an interactive entrypoint" {
  local input="{\"tool_name\":\"Bash\",\"session_id\":\"sess-1\",\"cwd\":\"$TMPDIR\"}"

  echo "$input" | CLAUDE_CODE_ENTRYPOINT=api bash "$OBSERVE_SH" post

  [ ! -f "$TMPDIR/.instinct-db/observations.jsonl" ]
}

@test "skips observation when agent_id is present (subagent session)" {
  local input="{\"tool_name\":\"Bash\",\"session_id\":\"sess-1\",\"agent_id\":\"agent-123\",\"cwd\":\"$TMPDIR\"}"

  echo "$input" | bash "$OBSERVE_SH" post

  [ ! -f "$TMPDIR/.instinct-db/observations.jsonl" ]
}

@test "detects project dir by finding .instinct-db walking up from cwd" {
  local project_dir
  project_dir="$(mktemp -d)"
  mkdir -p "$project_dir/.instinct-db"
  local cwd_dir="$project_dir/src/pkg"
  mkdir -p "$cwd_dir"

  local input="{\"tool_name\":\"Bash\",\"session_id\":\"sess-1\",\"cwd\":\"$cwd_dir\"}"

  echo "$input" | bash "$OBSERVE_SH" post

  [ -f "$project_dir/.instinct-db/observations.jsonl" ]
  rm -rf "$project_dir"
}

@test "skips silently when no .instinct-db found in cwd hierarchy" {
  local tmpdir
  tmpdir="$(mktemp -d)"

  local input="{\"tool_name\":\"Bash\",\"session_id\":\"sess-1\",\"cwd\":\"$tmpdir\"}"

  echo "$input" | bash "$OBSERVE_SH" post

  [ ! -f "$tmpdir/observations.jsonl" ]
  rm -rf "$tmpdir"
}

@test "PreToolUse includes tool_input in observation and truncates at 5000 chars" {
  local long_val
  long_val=$(python3 -c "print('x' * 6000)")
  local input="{\"tool_name\":\"Write\",\"tool_input\":{\"content\":\"$long_val\"},\"session_id\":\"sess-1\",\"cwd\":\"$TMPDIR\"}"

  echo "$input" | bash "$OBSERVE_SH" pre

  local input_len
  input_len=$(python3 -c "
import json
d = json.loads(open('$TMPDIR/.instinct-db/observations.jsonl').read())
print(len(d.get('input', '')))
")
  [ "$input_len" -gt 0 ]
  [ "$input_len" -le 5000 ]
}

@test "redacts secret patterns in tool output" {
  local input="{\"tool_name\":\"Bash\",\"tool_response\":\"api_key=supersecrettoken123\",\"session_id\":\"sess-1\",\"cwd\":\"$TMPDIR\"}"

  echo "$input" | bash "$OBSERVE_SH" post

  grep -v "supersecrettoken123" "$TMPDIR/.instinct-db/observations.jsonl"
  grep -q "REDACTED" "$TMPDIR/.instinct-db/observations.jsonl"
}

@test "archives observations.jsonl when it exceeds 10MB" {
  dd if=/dev/zero bs=1M count=11 2>/dev/null | tr '\0' 'x' > "$TMPDIR/.instinct-db/observations.jsonl"

  local input="{\"tool_name\":\"Bash\",\"session_id\":\"sess-1\",\"cwd\":\"$TMPDIR\"}"
  echo "$input" | bash "$OBSERVE_SH" post

  local line_count
  line_count=$(wc -l < "$TMPDIR/.instinct-db/observations.jsonl")
  [ "$line_count" -eq 1 ]
  ls "$TMPDIR/.instinct-db/observations.archive/" | grep -q "observations-"
}

@test "sends SIGUSR1 to observer after every N observations" {
  local signal_file="$TMPDIR/got_signal"
  bash -c "trap 'touch $signal_file' USR1; while true; do sleep 10 & wait; done" &
  local observer_pid=$!
  echo "$observer_pid" > "$TMPDIR/.instinct-db/.observer.pid"

  local input="{\"tool_name\":\"Bash\",\"session_id\":\"sess-1\",\"cwd\":\"$TMPDIR\"}"

  for i in $(seq 1 3); do
    echo "$input" | INSTINCT_OBSERVER_SIGNAL_EVERY_N=3 bash "$OBSERVE_SH" post
  done

  sleep 0.2
  kill "$observer_pid" 2>/dev/null || true

  [ -f "$signal_file" ]
}

@test "valid PostToolUse JSON writes one observation to observations.jsonl" {
  local input="{\"tool_name\":\"Bash\",\"tool_input\":{\"command\":\"ls\"},\"tool_response\":\"file.txt\",\"session_id\":\"sess-1\",\"cwd\":\"$TMPDIR\"}"

  echo "$input" | bash "$OBSERVE_SH" post

  [ -f "$TMPDIR/.instinct-db/observations.jsonl" ]
  [ "$(wc -l < "$TMPDIR/.instinct-db/observations.jsonl")" -eq 1 ]
}
