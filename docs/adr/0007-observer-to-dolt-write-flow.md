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
    "content": "make unit-test を使うこと。go test ./... は使わない",
    "trigger_desc": "Goのテストを実行する時",
    "domain": "testing",
    "scope": "project",
    "observation_count": 5
  }
]
```

- `scope`: `"project"`（プロジェクト固有）または `"global"`（汎用）を Haiku が観察内容から判断して出力
- `observation_count`: この知見の根拠となった観察の件数を Haiku が推定して出力

observer-loop.sh 側の処理（Python で JSON パース）:

```python
for item in data:
    subprocess.run([
        'instinct-cli', 'insert',
        '--content', item['content'],
        '--trigger', item['trigger_desc'],
        '--domain',  item.get('domain', ''),
        '--count',   str(item.get('observation_count', 0)),
        '--scope',   item.get('scope', 'project'),
    ])
```

`--project-id` は instinct-cli 側がプロジェクトディレクトリの git 情報から自動生成する。

## Consequences

- Haiku の `--allowedTools` を最小限（`Read` のみ）に保てる
- Haiku の責務が「観察の分析と JSON 出力」に限定され、ツール呼び出しの副作用がない
- Haiku が出力した JSON が不正な場合、Python 側でエラーを捕捉しやすい
- Dolt へのアクセスが `instinct-cli` に一元化され、Go 以外のランタイム（bash/Node.js）が Dolt に直接触れない
