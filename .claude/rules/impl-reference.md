---
paths: "cmd/**,**/.instinct-db/**"
---

# instinct-cli 実装リファレンス

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
  content_a       TEXT          NOT NULL,
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

## config.yml 構造

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
  branch: tadahiro                 # 個人ブランチ名（setup時にgit config user.nameから自動設定）
```

## instinct-cli サブコマンド

| コマンド | 説明 |
|---------|------|
| `instinct-cli setup` | Dolt DB 初期化 + リモート設定 |
| `instinct-cli insert` | instinct を INSERT |
| `instinct-cli list` | 一覧表示 |
| `instinct-cli list --merged` | 個人 + チームの統合一覧（重複排除） |
| `instinct-cli show <id>` | 指定した instinct の全フィールドを全文表示 |
| `instinct-cli dedup` | Haiku によるデータブランチ内 dedup |
| `instinct-cli dedup --cross-branch` | 複数個人ブランチ横断 dedup |
| `instinct-cli review` | main にない新規 instinct 一覧（レビュー待ちキュー） |
| `instinct-cli push` | CALL dolt_push() |
| `instinct-cli pull` | CALL dolt_pull() |
