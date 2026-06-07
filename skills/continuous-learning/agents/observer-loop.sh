#!/bin/bash

echo $$ > "${CLAUDE_PROJECT_DIR}/.observer.pid"

_handle_usr1() {
  local obs_file="${CLAUDE_PROJECT_DIR}/observations.jsonl"
  [ -f "$obs_file" ] || return 0
  claude --model haiku --print "$(cat "$obs_file")" > /dev/null 2>&1 || true
}

trap '_handle_usr1' USR1

while true; do
  sleep 3600 & wait
done
