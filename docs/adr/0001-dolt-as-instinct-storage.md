# ADR-0001: instinct の保存先に Dolt を採用する

## Status

Accepted

## Context

ECC（Everything Claude Code）は PreToolUse/PostToolUse フックで観察した作業パターンを instinct として YAML ファイルに保存する。YAML ファイルはローカルの `~/.local/share/ecc-homunculus/` に個人ごとに保存されるため、チーム間での共有が困難だった。

チームで instinct を共有・蓄積するには、バージョン管理可能でチーム全員がアクセスできるストレージが必要。また、instinct の変更履歴・誰がどのブランチに何を追加したかを追跡できることが望ましい。

## Decision

instinct の保存先を YAML ファイルから **Dolt**（MySQL 互換の Git-like DB）に変更する。

- GitHub の `refs/dolt/<project>/` に Git オブジェクトとして push/pull することでチーム共有を実現する
- 個人ブランチとチームブランチ（main）を分け、main はキュレーション済みの instinct のみを持つ
- 観察ログ（observations.jsonl）はローカルのみに保存し、共有しない

## Consequences

- チームメンバーが同じ GitHub リポジトリを通じて instinct を共有できる
- Dolt のブランチ・diff 機能を使って「レビュー待ちの instinct」を SQL で抽出できる
- `dolt` CLI のローカルインストールが不要（`dolthub/driver` を使用）
- instinct の変更履歴が Git コミットとして残るため、過去の状態への巻き戻しが容易
- 観察データはローカルのみに留まるため、プライバシーリスクなし
