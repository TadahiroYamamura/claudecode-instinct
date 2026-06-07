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
