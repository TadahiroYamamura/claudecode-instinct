# ADR-0004: 個人ブランチとチームブランチによる instinct 共有戦略

## Status

Accepted

## Context

チームで instinct を共有する際に、自動生成された未レビューの instinct がそのまま全員に適用されると品質保証ができない。特に導入初期は手動キュレーションが必要。一方、個人が独自に蓄積した instinct も参照できるようにしたい。

## Decision

Dolt のブランチ機能を用いて、個人ブランチとチームブランチ（main）を分離する。

**個人ブランチ**（例: `tadahiro`）
- Haiku が自動生成した instinct が蓄積される
- `instinct-cli push` で `refs/dolt/<project>/tadahiro` に push

**チームブランチ**（`main`）
- 手動キュレーション・レビュー済みの instinct のみを格納
- 他のメンバーは `instinct-cli pull` で取得

**参照時の統合**（Phase 2 以降）
- 個人 instinct + チーム instinct を重複排除して参照する
- Dolt の AS OF 構文でブランチ横断クエリが可能

```sql
-- チームにない個人の instinct を抽出（レビュー待ちキュー）
SELECT * FROM dolt_diff_instincts
WHERE from_commit = HASHOF('main')
  AND to_commit   = HASHOF('tadahiro')
  AND diff_type   = 'added';
```

## Implementation Notes

個人ブランチ名は `.instinct-db/config.yml` の `dolt.branch` で管理する。

```yaml
dolt:
  branch: tadahiro  # git config user.name から setup 時に自動設定
```

`instinct-cli setup` 実行時に `git config user.name` を取得してデフォルト値として書き込む。
取得できない場合は `"me"` にフォールバックする。

## Consequences

- 未レビューの instinct がチームに自動適用されるリスクがなくなる
- `dolt_diff_instincts` を使ったレビュー待ちキューの実装が自然にできる
- 複数メンバーのブランチを UNION することでクロスブランチ dedup も実現できる
- pull は完全手動（Phase 1）。main ブランチはキュレーション済みのため、手動 pull の頻度は低く問題にならない
