#!/bin/bash
INPUT_JSON=$(cat)

_cwd=$(echo "$INPUT_JSON" | python3 -c "import json,sys; print(json.load(sys.stdin).get('cwd',''))" 2>/dev/null || true)
[ -z "$_cwd" ] && exit 0

PROJECT_DIR=$(CWD_PATH="$_cwd" python3 -c "
import os
path = os.path.abspath(os.environ['CWD_PATH'])
while True:
    if os.path.isdir(os.path.join(path, '.instinct-db')):
        print(path); break
    parent = os.path.dirname(path)
    if parent == path: break
    path = parent
" 2>/dev/null || true)
[ -z "$PROJECT_DIR" ] && exit 0

pid_file="${PROJECT_DIR}/.instinct-db/.observer.pid"
if [ -f "$pid_file" ]; then
    pid=$(cat "$pid_file" 2>/dev/null || true)
    if kill -0 "$pid" 2>/dev/null; then
        exit 0
    fi
fi

if [ -z "${OBSERVER_SH:-}" ]; then
    PLUGIN_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
    OBSERVER_SH="${PLUGIN_ROOT}/skills/continuous-learning/agents/observer-loop.sh"
fi

nohup bash "$OBSERVER_SH" "$PROJECT_DIR" >/dev/null 2>&1 &
disown
