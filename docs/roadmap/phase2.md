# Phase 2 — instinct サブコマンド実装

## スコープ

Phase 1 で insert のみ実装した instinct に残りのサブコマンドを追加し、
日常的な instinct の確認・整理・共有ができる状態にする。

## タスク

- [x] `instinct list` — 一覧表示（content は40文字で打ち切り、ID短縮形・ヘッダー付きテーブル）
- [x] `instinct list --merged` — 個人 + チームの統合一覧（重複排除）
- [x] `instinct show <id>` — 指定した instinct の全フィールドを全文表示（Markdown風セクション形式）
- [x] `instinct dedup` — Haiku によるブランチ内 dedup
- [x] `instinct review` — main にない新規 instinct 一覧（レビュー待ちキュー）
- [x] `instinct push` — 個人ブランチをリモートへ送信
- [x] `instinct pull` — チームブランチをリモートから取得

## 前提

Phase 1 完了後に着手。
