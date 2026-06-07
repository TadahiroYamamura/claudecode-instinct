# CLAUDE.md — claudecode-instinct plugin

## このプロジェクトの概要

Claude Code の PreToolUse/PostToolUse フックでツール使用を観察し、Haiku エージェントがパターンを検出して **instinct**（再利用可能な作業知見）を生成・蓄積する Claude Code プラグイン。

instinct の保存先は **Dolt**（MySQL 互換の Git-like DB）で、GitHub 経由でチームと共有する。

---

## 重要な前提知識

### Dolt の使い方
- `dolt` CLI は**不要**。`dolthub/driver`（Go embedded driver）でプロセス内に Dolt エンジンを組み込む（CGO 必須・`gcc` が必要）
- GitHub への push/pull は SQL ストアドプロシージャ（`CALL dolt_push(...)` 等）で行う
- 詳細: ADR-0001、ADR-0002、`docs/references/dolt/02_claudecode_instruction.md`

### ECC（Everything Claude Code）との関係
- observe.sh / observer-loop.sh は ECC から流用（`~/work/ECC` にクローン済み）
- ECC の YAML ベースの instinct ストレージを Dolt に置き換えたのがこのプラグイン
- セッション開始時の instinct 注入（SessionStart フック）は **Phase 2 以降**

---

## アーキテクチャ

```
PreToolUse / PostToolUse フック（hooks.json）
    ↓ observe-runner.js → observe.sh（ECC流用）
observations.jsonl（ローカル保存、非共有）
    ↓ 20観察ごとに SIGUSR1 でトリガー
observer-loop.sh → claude --model haiku（JSON出力）
    ↓ シェルが jq でパース
instinct-cli insert → .instinct-db/data/（Dolt DB）
    ↓ 手動トリガー
instinct-cli dedup → Haiku エージェント → dedup_decisions に記録
    ↓ レビュー・承認後
instinct-cli push → CALL dolt_push() → GitHub refs/dolt/<project>/
```

---

## ファイル構成（実装予定）

```
claudecode-instinct/
├── plugin.json
├── hooks/hooks.json
├── scripts/hooks/
│   ├── observe-runner.js
│   ├── run-with-flags.js
│   └── plugin-hook-bootstrap.js
├── skills/continuous-learning/
│   ├── hooks/observe.sh           # ECC流用
│   ├── scripts/detect-project.sh
│   └── agents/
│       ├── observer-loop.sh
│       └── session-guardian.sh
├── cmd/instinct-cli/main.go       # Go CLI（dolthub/driver）
└── docs/adr/
```

---

## Dolt DB 配置

```
<CLAUDE_PROJECT_DIR>/
└── .instinct-db/
    ├── data/       # Dolt DB本体（gitignore対象）
    └── config.yml  # プロジェクト固有設定（git管理）
```

ブランチ戦略: 個人ブランチ（自動蓄積）→ main（キュレーション済み）でチーム共有。詳細: ADR-0004。

スキーマ・config.yml 構造・サブコマンド一覧は `.claude/rules/impl-reference.md`（`cmd/**` 編集時に自動ロード）。

---

## Phase 1 スコープ（現在）

- [x] 設計・ADR 作成
- [ ] observe.sh / observer-loop.sh のセットアップ（ECC 流用）
- [ ] instinct-cli 実装（Go + dolthub/driver）
- [ ] hooks.json / plugin.json 作成
- [ ] oncall-platform への適用・動作確認

## Phase 2 以降（未着手）

- SessionStart フックによる instinct 注入
- 自動品質チェック
- dedup の ML モデル化（dedup_decisions の訓練データ活用）
- instinct-cli pull の自動化
