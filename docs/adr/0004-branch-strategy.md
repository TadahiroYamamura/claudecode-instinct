# ADR-0004: 個人ブランチとチームブランチによる instinct 共有戦略

## Status

Accepted

## Context

チームで instinct を共有する際に、自動生成された未レビューの instinct がそのまま全員に適用されると品質保証ができない。特に導入初期は手動キュレーションが必要。一方、個人が独自に蓄積した instinct も参照できるようにしたい。

## Decision

Dolt のブランチ機能を用いて、個人ブランチとチームブランチ（main）を分離する。

**個人ブランチ**（例: `tadahiro`）
- Haiku が自動生成した instinct が蓄積される
- `instinct push` で `refs/dolt/<project>/tadahiro` に push

**チームブランチ**（`main`）
- 手動キュレーション・レビュー済みの instinct のみを格納
- 他のメンバーは `instinct pull` で取得

**参照時の統合**（Phase 2 以降）
- 個人 instinct + チーム instinct を重複排除して参照する
- Dolt の AS OF 構文でブランチ横断クエリが可能

レビューフローは `review_queue` テーブルを介する。

1. 個人ブランチで `instinct nominate <id...>` → チームブランチの `review_queue` に登録
2. レビュー担当者が `instinct review list` で確認
3. `instinct review approve <id...>` で承認 → `instincts` テーブルに昇格、`review_queue` から削除

## Implementation Notes

設定は2ファイルに分割されている。

**config.team.yml**（git管理）
```yaml
dolt:
  team_branch: main
  remote_url: "git@github.com:ORG/REPO.git"
```

**config.user.yml**（gitignore対象）
```yaml
dolt:
  branch: tadahiro  # git config user.name から init/connect 時に自動設定
```

個人ブランチ名のサニタイズルール: スペース→`_`、大文字→小文字、その他記号・非ASCII→削除。

`instinct init` / `instinct connect` 実行時に `git config user.name` を取得してデフォルト値として書き込む。取得できない場合は `"me"` にフォールバックする。

## Consequences

- 未レビューの instinct がチームに自動適用されるリスクがなくなる
- `review_queue` テーブルを仲介することでレビューフローが明確になる
- 複数メンバーのブランチを UNION することでクロスブランチ dedup も実現できる（Phase 3 以降）
- pull は完全手動（Phase 1）。main ブランチはキュレーション済みのため、手動 pull の頻度は低く問題にならない
