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
data = json.load(sys.stdin)
print(json.dumps({
    "timestamp": os.environ["TIMESTAMP"],
    "event": os.environ["EVENT"],
    "tool": data.get("tool_name", "unknown"),
    "session": data.get("session_id", "unknown"),
}))
' >> "$OBSERVATIONS_FILE"
