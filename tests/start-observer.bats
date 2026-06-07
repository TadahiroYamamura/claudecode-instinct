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

@test "observerが起動済みのとき二重起動しない" {
  # 先にobserverを起動してPIDを記録
  echo '{"session_id":"test-session","cwd":"'"$TMPDIR"'"}' \
    | OBSERVER_SH="$OBSERVER_SH" bash "$START_OBSERVER_SH"
  sleep 0.3
  first_pid=$(cat "$TMPDIR/.observer.pid")

  # 再度呼び出す
  echo '{"session_id":"test-session","cwd":"'"$TMPDIR"'"}' \
    | OBSERVER_SH="$OBSERVER_SH" bash "$START_OBSERVER_SH"
  sleep 0.1
  second_pid=$(cat "$TMPDIR/.observer.pid")

  [ "$first_pid" = "$second_pid" ]
}

@test "observerが未起動のとき.observer.pidを作成する" {
  echo '{"session_id":"test-session","cwd":"'"$TMPDIR"'"}' \
    | OBSERVER_SH="$OBSERVER_SH" bash "$START_OBSERVER_SH"
  sleep 0.3

  [ -f "$TMPDIR/.observer.pid" ]
}
