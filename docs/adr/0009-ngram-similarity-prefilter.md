# ADR-0009: 文字 n-gram 類似度による Haiku 呼び出し事前フィルタリング

## Status

Accepted

## Context

ADR-0006 で決定した dedup フローでは、全ペアを無条件に Haiku へ送信していた。instinct 数が増えると O(n²) の API コストが発生する。

事前にスコアリングして明らかに非類似なペアを除外すれば、Haiku の呼び出し回数を大幅に削減できる。ただし、フィルタリングで除外したペアは Haiku へ送られないため、**false negative（本来は重複だが見逃す）が発生しうる**。

また現時点ではデータが少なく、どのモデル（bigram / trigram / Jaccard overlap）が実際の重複検出に有効かを事前に判断できない。

## Decision

### 3モデルを常に並行計算する

bigram コサイン類似度・trigram コサイン類似度・bigram Jaccard 係数の3モデルを全ペアで常に計算し、スコアを `dedup_decisions` の別カラム（`sim_bigram`, `sim_trigram`, `sim_overlap`）に記録する。

単一モデルに固定すると、そのモデルが有効かどうかをモデル切替後にしか検証できない。3モデルを並行記録することで、Haiku の `decision` を正解ラベルとして**事後に各モデルの予測精度を比較**できる。

### OR ロジックで Haiku へ送信する

いずれか1モデルのスコアが閾値（デフォルト 0.15）以上であればペアを Haiku へ送信する（AND ではなく OR）。

AND ロジックにすると全モデルが閾値を超えた場合のみ送信されるため false negative が増える。運用初期はデータが少ないので、取りこぼしを最小化する OR ロジックを選ぶ。

### 閾値は config で調整可能にする

`config.team.yml` の `dedup.similarity_threshold` で閾値を設定し、データが蓄積された後に運用成績を見て調整できるようにする。

## Consequences

- 非類似ペアの大半をスキップでき、Haiku の API コストを削減できる
- 3モデルのスコアが蓄積されることで、将来的に最適モデルや適切な閾値を実績から選べる
- OR ロジックにより false negative を抑制できるが、その分 Haiku への送信数は AND より多い
- 1万件規模になると O(n²) のスキャン自体がボトルネックになる。その場合は全ドキュメントを一括でベクトル化するバッチ計算インターフェースへの変更が必要になる（現在の `computeAllScores(a, b string)` ペア単位では scikit-learn 等のメリットを活かせない）
