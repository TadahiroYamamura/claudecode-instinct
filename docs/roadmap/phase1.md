# Phase 1 — MVP

## スコープ

ローカル環境での観察・instinct 生成・Dolt 保存の基本フローを動かす。

## タスク

- [x] 設計・ADR 作成
- [ ] observe.sh / observer-loop.sh セットアップ（ECC 流用）
- [ ] instinct-cli 実装（Go + dolthub/driver）
- [ ] hooks.json / plugin.json 作成
- [ ] oncall-platform への適用・動作確認

## 完了条件

oncall-platform で実際にツール使用が観察され、instinct が Dolt に自動挿入される。
