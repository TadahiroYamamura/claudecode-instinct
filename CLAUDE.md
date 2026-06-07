# CLAUDE.md — claudecode-instinct plugin

## このプロジェクトの概要

Claude Code の PreToolUse/PostToolUse フックでツール使用を観察し、Haiku エージェントがパターンを検出して **instinct**（再利用可能な作業知見）を生成・蓄積する Claude Code プラグイン。

instinct の保存先は **Dolt**（MySQL 互換の Git-like DB）で、GitHub 経由でチームと共有する。

---

## 重要な前提知識

### Dolt の使い方
- `dolt` CLI は**不要**。`dolthub/driver`（Go embedded driver）でプロセス内に Dolt エンジンを組み込む
- CGO 必須。`gcc` が必要
- GitHub への push/pull は SQL ストアドプロシージャ（`CALL dolt_push(...)` 等）で行う
- 詳細: `~/work/life/research/study-session/dolt/02_claudecode_instruction.md`

### ECC（Everything Claude Code）との関係
- observe.sh / observer-loop.sh は ECC から流用（`~/work/ECC` にクローン済み）
- ECC の YAML ベースの instinct ストレージを Dolt に置き換えたのがこのプラグイン
- セッション開始時の instinct 注入（SessionStart フック）は **Phase 2 以降**。Phase 1 には含まない

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

## ファイル構成

```
claudecode-instinct/
├── plugin.json                    # Claude Code プラグイン定義
├── hooks/
│   └── hooks.json                 # PreToolUse / PostToolUse フック登録
├── scripts/
│   └── hooks/
│       ├── observe-runner.js      # フック → observe.sh ランナー
│       ├── run-with-flags.js      # フックフラグ制御
│       └── plugin-hook-bootstrap.js
├── skills/
│   └── continuous-learning/
│       ├── hooks/
│       │   └── observe.sh         # 観察スクリプト（ECC流用・改変最小限）
│       ├── scripts/
│       │   ├── detect-project.sh  # プロジェクトID検出
│       │   └── lib/
│       │       └── homunculus-dir.sh
│       └── agents/
│           ├── observer-loop.sh   # Haiku 起動・JSON 収集
│           └── session-guardian.sh
├── cmd/
│   └── instinct-cli/             # Go CLI（dolthub/driver使用）
│       └── main.go
└── docs/
    └── adr/                      # Architecture Decision Records
```

---

## Dolt DB 構成

プラグインが各プロジェクトで使う Dolt DB の配置：

```
<CLAUDE_PROJECT_DIR>/
├── .instinct-db/
│   ├── data/          # Dolt DB本体（gitignore対象）
│   └── config.yml     # プロジェクト固有設定（git管理）
```

### config.yml の構造

```yaml
observer:
  enabled: true
  trigger_every: 20
  active_hours: "800-2300"

confidence:
  thresholds:
    low: 3        # 3-5件 → confidence 0.5
    medium: 6     # 6-10件 → confidence 0.7
    high: 11      # 11件以上 → confidence 0.85

dedup:
  auto_run_before_push: false

dolt:
  remote_url: "git@github.com:ORG/REPO.git"
  refs: "refs/dolt/project-name"   # モノレポ対応：プロジェクト固有 namespace
```

---

## テーブルスキーマ

### instincts

```sql
CREATE TABLE instincts (
  id                VARCHAR(64)   PRIMARY KEY,  -- UUID（instinct-cli が生成）
  content           TEXT          NOT NULL,
  trigger_desc      TEXT          NOT NULL,
  domain            VARCHAR(128),
  source            ENUM('auto','manual') NOT NULL DEFAULT 'auto',
  scope             ENUM('project','global') NOT NULL DEFAULT 'project',
  project_id        VARCHAR(12)   NOT NULL,      -- git remote URL の SHA256[:12]
  project_name      VARCHAR(256),
  observation_count INT           NOT NULL DEFAULT 0,
  created_at        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP,
  updated_at        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);
```

### dedup_decisions

```sql
CREATE TABLE dedup_decisions (
  id              VARCHAR(64)   PRIMARY KEY,
  instinct_id_a   VARCHAR(64)   NOT NULL,
  instinct_id_b   VARCHAR(64)   NOT NULL,
  content_a       TEXT          NOT NULL,   -- 判定時点のスナップショット
  content_b       TEXT          NOT NULL,
  trigger_a       TEXT          NOT NULL,
  trigger_b       TEXT          NOT NULL,
  decision        ENUM('duplicate','distinct') NOT NULL,
  reasoning       TEXT,
  similarity      DECIMAL(4,3),
  decided_by      ENUM('agent','human') NOT NULL DEFAULT 'agent',
  human_label     ENUM('correct','wrong'),  -- 人間による事後訂正（ML訓練データ用）
  source_branch_a VARCHAR(128),
  source_branch_b VARCHAR(128),
  winner_branch   VARCHAR(128),
  created_at      TIMESTAMP     DEFAULT CURRENT_TIMESTAMP
);
```

---

## ブランチ戦略

```
個人ブランチ（例: tadahiro）
    → 自動生成 instinct が蓄積される
    → CALL dolt_push('origin', 'tadahiro')
    → refs/dolt/<project>/tadahiro に格納

チームブランチ（main）
    → 手動キュレーション・レビュー済み instinct
    → CALL dolt_pull('origin', 'main') でチーム共有を取得
```

---

## instinct-cli サブコマンド（実装予定）

| コマンド | 説明 |
|---------|------|
| `instinct-cli setup` | Dolt DB 初期化 + リモート設定 |
| `instinct-cli insert` | instinct を INSERT |
| `instinct-cli list` | 一覧表示 |
| `instinct-cli list --merged` | 個人 + チームの統合一覧（重複排除） |
| `instinct-cli dedup` | Haiku によるデータブランチ内 dedup |
| `instinct-cli dedup --cross-branch` | 複数個人ブランチ横断 dedup |
| `instinct-cli review` | main にない新規 instinct 一覧（レビュー待ちキュー） |
| `instinct-cli push` | CALL dolt_push() |
| `instinct-cli pull` | CALL dolt_pull() |

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
