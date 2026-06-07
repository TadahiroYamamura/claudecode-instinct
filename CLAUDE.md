# claudecode-instinct

Claude Code の PreToolUse/PostToolUse フックでツール使用を観察し、Haiku がパターンを検出して **instinct**（再利用可能な作業知見）を生成・蓄積するプラグイン。instinct は **Dolt**（MySQL 互換の Git-like DB）に保存し、GitHub 経由でチームと共有する。

## 前提知識

**Dolt**: `dolt` CLI 不要。`dolthub/driver`（Go embedded）でプロセス内に組み込む（CGO 必須・`gcc` が必要）。push/pull は `CALL dolt_push(...)` 等の SQL ストアドプロシージャで行う。→ ADR-0001、ADR-0002、`docs/references/dolt/02_claudecode_instruction.md`

**ECC**: observe.sh / observer-loop.sh は ECC（Everything Claude Code）から流用。ECC の YAML ストレージを Dolt に置き換えたのがこのプラグイン。SessionStart フックによる instinct 注入は Phase 2 以降。

## データフロー

```
PreToolUse/PostToolUse フック
    ↓ observe-runner.js → observe.sh
observations.jsonl（ローカル・非共有）
    ↓ 20観察ごとに SIGUSR1
observer-loop.sh → claude haiku（JSON）
    ↓ jq
instinct-cli insert → .instinct-db/（Dolt）
    ↓ 手動
instinct-cli dedup → dedup_decisions
    ↓ レビュー後
instinct-cli push → GitHub refs/dolt/<project>/
```

## 構成

```
claudecode-instinct/
├── plugin.json / hooks/hooks.json
├── scripts/hooks/            # observe-runner.js など
├── skills/continuous-learning/
│   ├── hooks/observe.sh      # ECC流用
│   └── agents/observer-loop.sh
├── cmd/instinct-cli/         # Go CLI（dolthub/driver）
└── docs/adr/
```

各プロジェクトの `.instinct-db/data/` が Dolt DB 本体（gitignore）、`.instinct-db/config.yml` が設定（git管理）。ブランチ戦略: 個人ブランチ（自動蓄積）→ main（キュレーション）→ チーム共有。→ ADR-0004

スキーマ・config.yml・サブコマンド一覧は `.claude/rules/impl-reference.md`（`cmd/**` 編集時に自動ロード）。

## Phase 1（現在進行中）

- [x] 設計・ADR 作成
- [ ] observe.sh / observer-loop.sh セットアップ（ECC 流用）
- [ ] instinct-cli 実装（Go + dolthub/driver）
- [ ] hooks.json / plugin.json 作成
- [ ] oncall-platform への適用・動作確認
