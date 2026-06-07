#!/bin/bash
set -e

HOOK_PHASE="${1:-post}"
INPUT_JSON=$(cat)

[ -z "$INPUT_JSON" ] && exit 0

[ "${INSTINCT_SKIP_OBSERVE:-0}" = "1" ] && exit 0

case "${CLAUDE_CODE_ENTRYPOINT:-cli}" in
  cli|sdk-ts|claude-desktop) ;;
  *) exit 0 ;;
esac

_agent_id=$(echo "$INPUT_JSON" | python3 -c "import json,sys; print(json.load(sys.stdin).get('agent_id',''))" 2>/dev/null || true)
[ -n "$_agent_id" ] && exit 0

if [ -z "${CLAUDE_PROJECT_DIR:-}" ]; then
  _cwd=$(echo "$INPUT_JSON" | python3 -c "import json,sys; print(json.load(sys.stdin).get('cwd',''))" 2>/dev/null || true)
  if [ -n "$_cwd" ]; then
    CLAUDE_PROJECT_DIR=$(python3 -c "
import os
path = os.path.abspath('$_cwd')
while True:
    if os.path.isdir(os.path.join(path, '.instinct-db')):
        print(path); break
    parent = os.path.dirname(path)
    if parent == path: break
    path = parent
" 2>/dev/null || true)
  fi
fi
[ -z "${CLAUDE_PROJECT_DIR:-}" ] && exit 0

OBSERVATIONS_FILE="${CLAUDE_PROJECT_DIR}/observations.jsonl"
MAX_FILE_SIZE_MB=10

if [ -f "$OBSERVATIONS_FILE" ]; then
  file_size_mb=$(du -m "$OBSERVATIONS_FILE" 2>/dev/null | cut -f1)
  if [ "${file_size_mb:-0}" -ge "$MAX_FILE_SIZE_MB" ]; then
    archive_dir="${CLAUDE_PROJECT_DIR}/observations.archive"
    mkdir -p "$archive_dir"
    mv "$OBSERVATIONS_FILE" "$archive_dir/observations-$(date +%Y%m%d-%H%M%S)-$$.jsonl" 2>/dev/null || true
  fi
fi
timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
if [ "$HOOK_PHASE" = "pre" ]; then event="tool_start"; else event="tool_complete"; fi

echo "$INPUT_JSON" | TIMESTAMP="$timestamp" EVENT="$event" python3 -c '
import json, sys, os, re

TRUNCATE = 5000
_SECRET_RE = re.compile(
    r"(?i)(api[_-]?key|token|secret|password|authorization|credentials?|auth)"
    r"([\"'"'"'\s:=]+)"
    r"([A-Za-z]+\s+)?"
    r"([A-Za-z0-9_\-/.+=]{8,})"
)

def scrub(val):
    if val is None:
        return None
    return _SECRET_RE.sub(lambda m: m.group(1) + m.group(2) + (m.group(3) or "") + "[REDACTED]", str(val))

data = json.load(sys.stdin)
event = os.environ["EVENT"]

obs = {
    "timestamp": os.environ["TIMESTAMP"],
    "event": event,
    "tool": data.get("tool_name", "unknown"),
    "session": data.get("session_id", "unknown"),
}

if event == "tool_start":
    raw = data.get("tool_input", {})
    obs["input"] = scrub((json.dumps(raw) if isinstance(raw, dict) else str(raw))[:TRUNCATE])
else:
    raw = data.get("tool_response", data.get("tool_output", ""))
    obs["output"] = scrub((json.dumps(raw) if isinstance(raw, dict) else str(raw or ""))[:TRUNCATE])

print(json.dumps(obs))
' >> "$OBSERVATIONS_FILE"

# Signal observer every N observations
SIGNAL_EVERY_N="${INSTINCT_OBSERVER_SIGNAL_EVERY_N:-20}"
SIGNAL_COUNTER_FILE="${CLAUDE_PROJECT_DIR}/.observer-signal-counter"
counter=0
if [ -f "$SIGNAL_COUNTER_FILE" ]; then
  counter=$(cat "$SIGNAL_COUNTER_FILE" 2>/dev/null || echo 0)
fi
counter=$((counter + 1))
if [ "$counter" -ge "$SIGNAL_EVERY_N" ]; then
  counter=0
  pid_file="${CLAUDE_PROJECT_DIR}/.observer.pid"
  if [ -f "$pid_file" ]; then
    observer_pid=$(cat "$pid_file" 2>/dev/null || true)
    case "$observer_pid" in
      ''|*[!0-9]*|0|1) rm -f "$pid_file" ;;
      *) kill -USR1 "$observer_pid" 2>/dev/null || true ;;
    esac
  fi
fi
echo "$counter" > "$SIGNAL_COUNTER_FILE"
