# ADR-0008: プラグインを独立したリポジトリで管理する

## Status

Accepted

## Context

当初はプラグインをプロジェクトリポジトリの `.claude-plugin/` に配置する案を検討していた。しかし、対象プロジェクトの README は環境構築手順が長く、プラグインのドキュメントを追記しても埋もれてしまう可能性があった。

また、将来的に複数のプロジェクトで同じプラグインを使い回すことを考えると、プラグインコードをプロジェクトに埋め込む形は適切でない。

## Decision

プラグインコードを独立したリポジトリ（`TadahiroYamamura/claudecode-instinct`）で管理する。

- プラグインのインストール: `claude plugin install TadahiroYamamura/claudecode-instinct`
- instinct データ（Dolt DB）は各プロジェクトリポジトリの `refs/dolt/<project>/` に push する
- プロジェクト側には `.instinct-db/config.yml`（設定）と `.instinct-db/data/`（Dolt DB、gitignore）のみが存在する

Dolt DB のローカル配置:

```
$CLAUDE_PROJECT_DIR/
└── .instinct-db/
    ├── data/       # Dolt DB 本体（gitignore）
    └── config.yml  # プロジェクト固有設定（git 管理）
```

## Consequences

- プラグイン固有の README・ドキュメントが独立して整備できる
- 複数プロジェクトで同じプラグインバージョンを使い回せる
- プラグインコードの更新がプロジェクトリポジトリに影響しない
- instinct データは各プロジェクトのリポジトリに残るため、データとコードが分離される
- プロジェクト側の設定（`config.yml`）はプロジェクトの git で管理されるため、チーム設定の共有が可能
