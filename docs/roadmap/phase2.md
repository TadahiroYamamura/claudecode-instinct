# Phase 2 — 品質・共有の強化

## スコープ

Phase 1 で蓄積されたデータを活用し、instinct の品質向上とチーム共有を自動化する。

## タスク

- [ ] SessionStart フックによる instinct 注入（セッション開始時にコンテキストへ展開）
- [ ] 自動品質チェック（生成 instinct のスコアリング・フィルタリング）
- [ ] dedup の ML モデル化（dedup_decisions を訓練データとして活用）
- [ ] instinct-cli pull の自動化（SessionStart 時に最新チーム instinct を取得）

## 前提

Phase 1 完了後に着手。
