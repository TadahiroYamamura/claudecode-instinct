# ADR-0003: instinct データベーススキーマ設計

## Status

Accepted

## Context

ECC の instinct は YAML ファイルで `confidence`（信頼度スコア）フィールドを持つ。信頼度は観察数から導出される（3-5件→0.5、6-10件→0.7、11件以上→0.85）。SQL テーブルに変換する際に、この冗長性を解消する必要があった。

また、dedup エージェント（Haiku）の判定結果を将来の ML モデル訓練データとして活用したいという要件があった。

## Decision

### instincts テーブル

`confidence` カラムを持たず、`observation_count` のみを保持する。confidence が必要な場面ではアプリケーション層で計算する。

```sql
CREATE TABLE instincts (
  id                VARCHAR(64)   PRIMARY KEY,  -- UUID（instinct-cli が生成）
  content           TEXT          NOT NULL,
  trigger_desc      TEXT          NOT NULL,
  domain            VARCHAR(128),
  source            ENUM('auto','manual') NOT NULL DEFAULT 'auto',
  scope             ENUM('project','global') NOT NULL DEFAULT 'project',
  project_id        VARCHAR(12)   NOT NULL,
  project_name      VARCHAR(256),
  observation_count INT           NOT NULL DEFAULT 0,
  created_at        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP,
  updated_at        TIMESTAMP     DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);
```

### dedup_decisions テーブル

Haiku エージェントが「何を同じとみなしたか」の判定結果と、判定時点の instinct スナップショットを保存する。

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
  human_label     ENUM('correct','wrong'),
  source_branch_a VARCHAR(128),
  source_branch_b VARCHAR(128),
  winner_branch   VARCHAR(128),
  created_at      TIMESTAMP     DEFAULT CURRENT_TIMESTAMP
);
```

## Consequences

- `confidence` と `observation_count` の二重管理がなくなり、データの一貫性が保たれる
- `observation_count` は INSERT 時に Haiku が推定した値をセット。dedup でマージされた場合は `count_A + count_B` に更新される（自動インクリメントは行わない）
- `dedup_decisions.human_label` により、Haiku の判定に対する人間の訂正を記録できる。`decided_by='agent' AND human_label IS NOT NULL` が教師あり学習データになる
- 判定時点のスナップショット（`content_a`, `content_b` 等）を保持することで、後から instinct が変更されても判定根拠が追跡できる
