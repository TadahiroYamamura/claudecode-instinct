# ADR-0006: Haiku エージェントによる dedup と訓練データ収集

## Status

Accepted

## Context

同じ観察パターンから意味的に同一の instinct が複数生成されることがある（同一ブランチ内・複数個人ブランチ間）。重複を人間だけで排除するのはレビュー負担が大きい。一方、「何を同じとみなすか」はルールベースで完全に定義できない意味的な判断であり、TF-IDF 等の決定論的アルゴリズムでは表現の揺れに対応できない。

また将来的に dedup を ML モデルで自動化する計画があり、そのための訓練データが必要。

## Decision

dedup を Haiku エージェントが担当し、判定結果を `dedup_decisions` テーブルに記録する。

**dedup のトリガー**: `instinct-cli dedup` コマンドで手動実行（push 前の任意実行）。push 前の強制実行はしない。

**dedup のフロー**:
1. `instinct-cli dedup` が対象ブランチの全 instinct を読み込む
2. Haiku に渡して重複候補を検出させる
3. 各判定（duplicate / distinct）を `reasoning` と3モデルのスコア（`sim_bigram`, `sim_trigram`, `sim_overlap`）とともに `dedup_decisions` に INSERT する
4. duplicate と判定されたペアはマージ（`observation_count` を合算）し、一方を削除する

**クロスブランチ dedup** (`--cross-branch` オプション):
- 複数個人ブランチの instinct を UNION して Haiku に渡す
- `source_branch_a`, `source_branch_b`, `winner_branch` で由来を記録する

**訓練データの活用**:
- `decided_by='agent' AND human_label IS NOT NULL` のレコードが教師あり学習データになる
- `human_label='wrong'` は Haiku の誤判定サンプルとして活用できる

## Consequences

- 表現の揺れがある意味的重複を検出できる
- 判定の透明性が確保される（`reasoning` フィールドで判定理由を記録）
- API コスト（Haiku 料金）が発生するが、dedup は頻繁に実行するものではないため許容範囲
- 将来的に ML モデルへ移行する際の訓練データが自動的に蓄積される
- `human_label` によって人間が誤判定を訂正でき、モデル品質の改善につながる
