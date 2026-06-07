#!/bin/bash

PROJECT_DIR="${1:?project directory required}"

echo $$ > "${PROJECT_DIR}/.observer.pid"

_handle_usr1() {
  local obs_file="${PROJECT_DIR}/observations.jsonl"
  [ -f "$obs_file" ] || return 0

  local claude_output
  claude_output=$(claude --model haiku --print "$(cat "$obs_file")" 2>/dev/null) || return 0

  echo "$claude_output" | python3 -c "
import json, sys, subprocess, os

try:
    data = json.load(sys.stdin)
except Exception:
    sys.exit(0)
if not isinstance(data, list):
    sys.exit(0)

for item in data:
    content = item.get('content', '')
    trigger = item.get('trigger_desc', '')
    domain = item.get('domain', '')
    if not content:
        continue
    subprocess.run(
        ['instinct-cli', 'insert',
         '--content', content,
         '--trigger', trigger,
         '--domain', domain],
        env=os.environ, capture_output=True
    )
" 2>/dev/null || true

  local archive_dir="${PROJECT_DIR}/observations.archive"
  mkdir -p "$archive_dir"
  mv "$obs_file" "$archive_dir/observations-$(date +%Y%m%d-%H%M%S)-$$.jsonl" 2>/dev/null || true
}

trap '_handle_usr1' USR1

while true; do
  sleep 3600 & wait
done
