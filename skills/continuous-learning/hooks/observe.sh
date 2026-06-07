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
    CLAUDE_PROJECT_DIR=$(git -C "$_cwd" rev-parse --show-toplevel 2>/dev/null || true)
  fi
fi
[ -z "${CLAUDE_PROJECT_DIR:-}" ] && exit 0

OBSERVATIONS_FILE="${CLAUDE_PROJECT_DIR}/observations.jsonl"
timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
if [ "$HOOK_PHASE" = "pre" ]; then event="tool_start"; else event="tool_complete"; fi

echo "$INPUT_JSON" | TIMESTAMP="$timestamp" EVENT="$event" python3 -c '
import json, sys, os

TRUNCATE = 5000
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
    obs["input"] = (json.dumps(raw) if isinstance(raw, dict) else str(raw))[:TRUNCATE]
else:
    raw = data.get("tool_response", data.get("tool_output", ""))
    obs["output"] = (json.dumps(raw) if isinstance(raw, dict) else str(raw or ""))[:TRUNCATE]

print(json.dumps(obs))
' >> "$OBSERVATIONS_FILE"
