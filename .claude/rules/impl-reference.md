---
paths: "cmd/**,**/.instinct-db/**"
---

# instinct 実装リファレンス

## テーブルスキーマ

### instincts

```sql
CREATE TABLE instincts (
  id                VARCHAR(64)   PRIMARY KEY,  -- UUID（instinct が生成）
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

### review_queue

```sql
CREATE TABLE review_queue (
  instinct_id     VARCHAR(64)   PRIMARY KEY,
  content         TEXT          NOT NULL,
  trigger_desc    TEXT          NOT NULL,
  domain          VARCHAR(128),
  observation_count INT         NOT NULL DEFAULT 0,
  scope           ENUM('project','global') NOT NULL DEFAULT 'project',
  project_id      VARCHAR(12)   NOT NULL DEFAULT '',
  submitted_by    VARCHAR(256)  NOT NULL,
  submitted_at    TIMESTAMP     DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
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
  branch: tadahiro  # 個人ブランチ名（init/connect時にgit config user.nameから自動設定、スペース→_・大文字→小文字・その他記号/非ASCII→削除）
```

**config.user.yml が存在しない = setup 未実施 → エラー**（フォールバックなし）

## instinct サブコマンド

| コマンド | 説明 |
|---------|------|
| `instinct init [-y] [-b branch] [--team-branch branch]` | Dolt DB をローカルに初期化（リモート不要）。エラー時は作成したファイルをクリーンアップ |
| `instinct connect [-y] [-b branch] [-r remote-url] [--refs refs]` | .instinct-db をリモートに接続（push / clone）。エラー時は作成したファイルをクリーンアップ |
| `instinct insert` | instinct を working set に INSERT（commit しない） |
| `instinct commit [-m msg]` | working set を Dolt commit として記録（observer-loop.sh がバッチ後に呼ぶ） |
| `instinct list` | 一覧表示 |
| `instinct list --merged` | 個人 + チームの統合一覧（重複排除） |
| `instinct show <id>` | 指定した instinct の全フィールドを全文表示 |
| `instinct dedup` | Haiku によるブランチ内 dedup（dedup_decisions に記録 + commit） |
| `instinct nominate [list]` | 推薦候補一覧を表示（observation_count >= review_min かつチームブランチ未マージのもの） |
| `instinct nominate <id...>` | 指定 ID を review_queue に登録（推薦）。引数なし or `list` で一覧表示に fallback |
| `instinct review list` | review_queue の一覧を表示（チームブランチ上の review_queue を参照） |
| `instinct review approve <id...>` | 指定 ID を承認してチームブランチに昇格 |
| `instinct push` | config.user.yml の branch をリモートへ push（branch 未設定はエラー、main へのフォールバックなし） |
| `instinct pull` | チームブランチと個人ブランチの両方を pull（チーム→個人の順）。個人ブランチがリモート未存在の場合はスキップ（エラーにならない） |
