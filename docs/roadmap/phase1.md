# Phase 1 — MVP

## スコープ

ローカル環境での観察・instinct 生成・Dolt 保存の基本フローを動かす。

## タスク

- [x] 設計・ADR 作成
- [x] observe.sh セットアップ（Linux専用・TDD実装）
- [x] observer-loop.sh セットアップ（ECC 流用・JSON出力→instinct連携）
- [x] instinct 実装（Go + dolthub/driver）
- [x] hooks.json / plugin.json 作成
- [x] SessionStart フックで observer-loop.sh を自動起動（start-observer.sh）
- [x] project への適用・動作確認

## 完了条件

project で実際にツール使用が観察され、instinct が Dolt に自動挿入される。
