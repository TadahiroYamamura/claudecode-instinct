#!/bin/bash

PROJECT_DIR="${1:?project directory required}"

echo $$ > "${PROJECT_DIR}/.observer.pid"

_handle_usr1() {
  local obs_file="${PROJECT_DIR}/observations.jsonl"
  [ -f "$obs_file" ] || return 0

  local prompt
  prompt="$(cat <<PROMPT
以下のツール使用観察ログを分析し、将来の作業に役立つ知見（instinct）をJSON配列で返してください。

出力はJSON配列のみ（説明文なし）:
[
  {
    "content": "具体的な行動指針",
    "trigger_desc": "この知見を適用すべき状況",
    "domain": "分野（workflow/code/testing/git など）",
    "observation_count": この知見の根拠となった観察の件数（整数）,
    "scope": "project" または "global"（プロジェクト固有なら project、どの作業でも使えるなら global）
  }
]

知見がない場合は [] を返してください。

## 観察ログ
$(cat "$obs_file")
PROMPT
)"

  local claude_output
  claude_output=$(claude --model "${INSTINCT_CLAUDE_MODEL:-haiku}" --print "$prompt" 2>/dev/null) || return 0

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
    count = str(item.get('observation_count', 0))
    scope = item.get('scope', 'project')
    if not content:
        continue
    subprocess.run(
        ['instinct-cli', 'insert',
         '--content', content,
         '--trigger', trigger,
         '--domain', domain,
         '--count', count,
         '--scope', scope],
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
