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
  sim_bigram      DECIMAL(4,3),
  sim_trigram     DECIMAL(4,3),
  sim_overlap     DECIMAL(4,3),
  decided_by      ENUM('agent','human') NOT NULL DEFAULT 'agent',
  human_label     ENUM('correct','wrong'),  -- 人間による事後訂正（ML訓練データ用）
  source_branch_a VARCHAR(128),
  source_branch_b VARCHAR(128),
  winner_branch   VARCHAR(128),
  created_at      TIMESTAMP     DEFAULT CURRENT_TIMESTAMP
);
```

## 設定ファイル構造

設定は2ファイルに分割されている。

### config.team.yml（Git管理・チーム共有）

```yaml
observer:
  enabled: true
  trigger_every: 20
  active_hours: "800-2300"

confidence:
  review_min: 6   # この観察数以上のinstinctのみreviewコマンドに表示する

dedup:
  auto_run_before_push: false
  similarity_threshold: 0.15   # いずれかのモデルがこの値以上のペアのみHaikuに送る

dolt:
  refs: "refs/dolt/project-name/"  # モノレポ対応：プロジェクト固有 namespace（setup時にディレクトリ名から自動設定）
  team_branch: main                # チームブランチ名（list --merged の参照先）
  remote_url: "git@github.com:ORG/REPO.git"  # push/pull先（setup時にgit remote origin urlから自動設定）
```

### config.user.yml（gitignore対象・ユーザー固有）

```yaml
dolt:
  branch: tadahiro  # 個人ブランチ名（setup時にgit config user.nameから自動設定、スペースはハイフンに変換）
```

**config.user.yml が存在しない = setup 未実施 → エラー**（フォールバックなし）

## instinct-cli サブコマンド

| コマンド | 説明 |
|---------|------|
| `instinct-cli setup [-y]` | Dolt DB 初期化 + config.yml 生成（対話形式、`-y` で非対話） |
| `instinct-cli insert` | instinct を working set に INSERT（commit しない） |
| `instinct-cli commit [-m msg]` | working set を Dolt commit として記録（observer-loop.sh がバッチ後に呼ぶ） |
| `instinct-cli list` | 一覧表示 |
| `instinct-cli list --merged` | 個人 + チームの統合一覧（重複排除） |
| `instinct-cli show <id>` | 指定した instinct の全フィールドを全文表示 |
| `instinct-cli dedup` | Haiku によるブランチ内 dedup（dedup_decisions に記録 + commit） |
| `instinct-cli review` | TUI でレビュー候補を選択し review_queue に登録（observation_count >= review_min かつチームブランチ未マージのもの） |
| `instinct-cli push` | config.user.yml の branch をリモートへ push（branch 未設定はエラー、main へのフォールバックなし） |
| `instinct-cli pull` | チームブランチと個人ブランチの両方をpull（チーム→個人の順、完了後は個人ブランチに滞留） |
