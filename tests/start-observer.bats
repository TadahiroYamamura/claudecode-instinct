#!/usr/bin/env bats

setup() {
  TMPDIR="$(mktemp -d)"
  mkdir -p "$TMPDIR/.instinct-db"
  START_OBSERVER_SH="$BATS_TEST_DIRNAME/../hooks/start-observer.sh"
  OBSERVER_SH="$BATS_TEST_DIRNAME/../skills/continuous-learning/agents/observer-loop.sh"
}

teardown() {
  if [ -f "$TMPDIR/.observer.pid" ]; then
    kill "$(cat "$TMPDIR/.observer.pid")" 2>/dev/null || true
  fi
  rm -rf "$TMPDIR"
}

@test "observerが未起動のとき.observer.pidを作成する" {
  echo '{"session_id":"test-session","cwd":"'"$TMPDIR"'"}' \
    | OBSERVER_SH="$OBSERVER_SH" bash "$START_OBSERVER_SH"
  sleep 0.3

  [ -f "$TMPDIR/.observer.pid" ]
}
