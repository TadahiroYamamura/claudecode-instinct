#!/usr/bin/env bats

setup() {
  TMPDIR="$(mktemp -d)"
  mkdir -p "$TMPDIR/.instinct-db"
  OBSERVER_SH="$BATS_TEST_DIRNAME/../skills/continuous-learning/agents/observer-loop.sh"
}

teardown() {
  if [ -f "$TMPDIR/.observer.pid" ]; then
    kill "$(cat "$TMPDIR/.observer.pid")" 2>/dev/null || true
  fi
  rm -rf "$TMPDIR"
}

@test "exits with error when project directory argument is missing" {
  local exit_code=0
  bash "$OBSERVER_SH" 2>/dev/null || exit_code=$?
  [ "$exit_code" -ne 0 ]
}

@test "writes own PID to .observer.pid on startup" {
  bash "$OBSERVER_SH" "$TMPDIR" &
  local launched_pid=$!
  sleep 0.2

  [ -f "$TMPDIR/.observer.pid" ]
  [ "$(cat "$TMPDIR/.observer.pid")" = "$launched_pid" ]
}

@test "SIGUSR1 triggers claude with observations as prompt" {
  local fake_bin="$TMPDIR/bin"
  mkdir -p "$fake_bin"
  cat > "$fake_bin/claude" <<SH
#!/bin/bash
echo '[]'
touch "$TMPDIR/claude_called"
SH
  chmod +x "$fake_bin/claude"

  echo '{"event":"tool_complete","tool":"Bash"}' > "$TMPDIR/observations.jsonl"

  PATH="$fake_bin:$PATH" bash "$OBSERVER_SH" "$TMPDIR" &
  local pid=$!
  sleep 0.2

  kill -USR1 "$pid"
  sleep 0.3

  kill "$pid" 2>/dev/null || true
  [ -f "$TMPDIR/claude_called" ]
}

@test "claude JSON output is passed to instinct-cli insert" {
  local fake_bin="$TMPDIR/bin"
  mkdir -p "$fake_bin"
  cat > "$fake_bin/claude" <<'SH'
#!/bin/bash
echo '[{"content":"コマンド実行前に計画を立てる","trigger_desc":"Bash tool 使用時","domain":"workflow"}]'
SH
  chmod +x "$fake_bin/claude"
  cat > "$fake_bin/instinct-cli" <<SH
#!/bin/bash
echo "\$@" >> "$TMPDIR/instinct_cli_calls"
SH
  chmod +x "$fake_bin/instinct-cli"

  echo '{"event":"tool_complete","tool":"Bash"}' > "$TMPDIR/observations.jsonl"

  PATH="$fake_bin:$PATH" bash "$OBSERVER_SH" "$TMPDIR" &
  local pid=$!
  sleep 0.2

  kill -USR1 "$pid"
  sleep 0.3

  kill "$pid" 2>/dev/null || true
  grep -q "insert" "$TMPDIR/instinct_cli_calls"
  grep -q "コマンド実行前に計画を立てる" "$TMPDIR/instinct_cli_calls"
}

@test "observation_count from claude output is passed to instinct-cli as --count" {
  local fake_bin="$TMPDIR/bin"
  mkdir -p "$fake_bin"
  cat > "$fake_bin/claude" <<'SH'
#!/bin/bash
echo '[{"content":"テスト前に仕様を確認する","trigger_desc":"テスト実行時","domain":"testing","observation_count":7}]'
SH
  chmod +x "$fake_bin/claude"
  cat > "$fake_bin/instinct-cli" <<SH
#!/bin/bash
echo "\$@" >> "$TMPDIR/instinct_cli_calls"
SH
  chmod +x "$fake_bin/instinct-cli"

  echo '{"event":"tool_complete","tool":"Bash"}' > "$TMPDIR/observations.jsonl"

  PATH="$fake_bin:$PATH" bash "$OBSERVER_SH" "$TMPDIR" &
  local pid=$!
  sleep 0.2

  kill -USR1 "$pid"
  sleep 0.3

  kill "$pid" 2>/dev/null || true
  grep -q "\-\-count" "$TMPDIR/instinct_cli_calls"
  grep -q "7" "$TMPDIR/instinct_cli_calls"
}

@test "claude is called with a structured prompt requesting JSON array output" {
  local fake_bin="$TMPDIR/bin"
  mkdir -p "$fake_bin"
  cat > "$fake_bin/claude" <<SH
#!/bin/bash
# Record the full prompt passed via --print
for arg in "\$@"; do
  echo "\$arg" >> "$TMPDIR/claude_prompt"
done
echo '[]'
SH
  chmod +x "$fake_bin/claude"

  echo '{"event":"tool_complete","tool":"Bash"}' > "$TMPDIR/observations.jsonl"

  PATH="$fake_bin:$PATH" bash "$OBSERVER_SH" "$TMPDIR" &
  local pid=$!
  sleep 0.2

  kill -USR1 "$pid"
  sleep 0.3

  kill "$pid" 2>/dev/null || true
  grep -qi "json" "$TMPDIR/claude_prompt"
  grep -qi "content" "$TMPDIR/claude_prompt"
  grep -qi "trigger" "$TMPDIR/claude_prompt"
}

@test "observations.jsonl is archived after claude processing" {
  local fake_bin="$TMPDIR/bin"
  mkdir -p "$fake_bin"
  cat > "$fake_bin/claude" <<'SH'
#!/bin/bash
echo '[]'
SH
  chmod +x "$fake_bin/claude"

  echo '{"event":"tool_complete","tool":"Bash"}' > "$TMPDIR/observations.jsonl"

  PATH="$fake_bin:$PATH" bash "$OBSERVER_SH" "$TMPDIR" &
  local pid=$!
  sleep 0.2

  kill -USR1 "$pid"
  sleep 0.3

  kill "$pid" 2>/dev/null || true
  [ ! -f "$TMPDIR/observations.jsonl" ]
  ls "$TMPDIR/observations.archive/" | grep -q "observations-"
}
