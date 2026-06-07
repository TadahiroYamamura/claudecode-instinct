#!/bin/bash
set -e

HOOK_PHASE="${1:-post}"
INPUT_JSON=$(cat)

[ -z "$INPUT_JSON" ] && exit 0

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
