# ADR-0007: Observer から Dolt への書き込みフロー

## Status

Accepted

## Context

Haiku エージェントが生成した instinct を Dolt DB に書き込む方法として、2 つの案を検討した。

**案 A**: Haiku の `--allowedTools` に `instinct-cli` を追加し、Haiku 自身が直接 `instinct-cli insert` を呼ぶ

**案 B**: Haiku は JSON を標準出力するだけにし、シェルがパースして `instinct-cli insert` を呼ぶ

## Decision

**案 B** を採用する。Haiku の出力を JSON 限定とし、シェルがパースして `instinct-cli` を呼ぶ。

Haiku の出力フォーマット:

```json
[
  {
    "trigger_desc": "Goのテストを実行する時",
    "content": "make unit-test を使うこと。go test ./... は使わない",
    "domain": "testing",
    "scope": "project",
    "observation_count": 5
  }
]
```

シェル側の処理:

```bash
echo "$haiku_output" | jq -c '.[]' | while IFS= read -r item; do
  "$INSTINCT_CLI" insert \
    --trigger  "$(echo "$item" | jq -r '.trigger_desc')" \
    --content  "$(echo "$item" | jq -r '.content')" \
    --domain   "$(echo "$item" | jq -r '.domain')" \
    --count    "$(echo "$item" | jq -r '.observation_count')" \
    --project-id "$PROJECT_ID"
done
```

## Consequences

- Haiku の `--allowedTools` を最小限（`Read` のみ）に保てる
- Haiku の責務が「観察の分析と JSON 出力」に限定され、ツール呼び出しの副作用がない
- JSON パースに `jq` が必要（ほぼすべての開発環境に存在する）
- Haiku が出力した JSON が不正な場合、シェル側でエラーを捕捉しやすい
- Dolt へのアクセスが `instinct-cli` に一元化され、Go 以外のランタイム（bash/Node.js）が Dolt に直接触れない
