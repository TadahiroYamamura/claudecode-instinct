#!/usr/bin/env bats

setup() {
  TMPDIR="$(mktemp -d)"
  mkdir -p "$TMPDIR/.instinct-db"
  OBSERVER_SH="$BATS_TEST_DIRNAME/../skills/continuous-learning/agents/observer-loop.sh"
  export CLAUDE_PROJECT_DIR="$TMPDIR"
}

teardown() {
  # Kill any observer process still running in this test's project dir
  if [ -f "$TMPDIR/.observer.pid" ]; then
    kill "$(cat "$TMPDIR/.observer.pid")" 2>/dev/null || true
  fi
  rm -rf "$TMPDIR"
}

@test "writes own PID to .observer.pid on startup" {
  bash "$OBSERVER_SH" &
  local launched_pid=$!
  sleep 0.2

  [ -f "$TMPDIR/.observer.pid" ]
  [ "$(cat "$TMPDIR/.observer.pid")" = "$launched_pid" ]
}

@test "SIGUSR1 triggers claude with observations as prompt" {
  local fake_bin="$TMPDIR/bin"
  mkdir -p "$fake_bin"
  cat > "$fake_bin/claude" <<'SH'
#!/bin/bash
echo '[]'
touch "${CLAUDE_PROJECT_DIR}/claude_called"
SH
  chmod +x "$fake_bin/claude"

  echo '{"event":"tool_complete","tool":"Bash"}' > "$TMPDIR/observations.jsonl"

  PATH="$fake_bin:$PATH" bash "$OBSERVER_SH" &
  local pid=$!
  sleep 0.2

  kill -USR1 "$pid"
  sleep 0.3

  kill "$pid" 2>/dev/null || true
  [ -f "$TMPDIR/claude_called" ]
}
