# Phase 2 — instinct-cli サブコマンド実装

## スコープ

Phase 1 で insert のみ実装した instinct-cli に残りのサブコマンドを追加し、
日常的な instinct の確認・整理・共有ができる状態にする。

## タスク

- [x] `instinct-cli list` — 一覧表示（content は40文字で打ち切り、ID短縮形・ヘッダー付きテーブル）
- [x] `instinct-cli list --merged` — 個人 + チームの統合一覧（重複排除）
- [x] `instinct-cli show <id>` — 指定した instinct の全フィールドを全文表示（Markdown風セクション形式）
- [ ] `instinct-cli dedup` — Haiku によるブランチ内 dedup
- [ ] `instinct-cli dedup --cross-branch` — 複数個人ブランチ横断 dedup
- [ ] `instinct-cli review` — main にない新規 instinct 一覧（レビュー待ちキュー）
- [ ] `instinct-cli push` — `CALL dolt_push()` でチームリポジトリへ送信
- [ ] `instinct-cli pull` — `CALL dolt_pull()` でチームリポジトリから取得

## 前提

Phase 1 完了後に着手。
