# ADR-0002: dolthub/driver による Dolt エンジンの埋め込み

## Status

Accepted

## Context

Dolt を使うには通常 `dolt` CLI をローカルにインストールするか、`dolt sql-server` を起動する必要がある。プラグインの利用者にこれらのセットアップを強制すると、導入障壁が上がりチームへの展開が困難になる。

instinct（Dolt に読み書きする Go CLI ツール）が Dolt にアクセスするための手段を選定する必要があった。

## Decision

`github.com/dolthub/driver`（Go embedded driver）を採用する。

- SQLite と同じアーキテクチャ：Dolt エンジン全体を Go プロセスにリンクする
- `database/sql` 互換のインターフェースで標準的な Go DB コードとして扱える
- `dolt` CLI も `dolt sql-server` も不要
- `instinct` バイナリは初回実行時に自動ビルドし、`.gitignore` で管理する（プリコンパイル済みバイナリはコミットしない）

## Consequences

- ユーザーが `dolt` をインストールせずにプラグインを使える
- CGO が必要なため、`go` と `gcc` のビルド環境が必須
- 初回実行時にビルドが走るため、1〜2 分の待機が発生する
- クロスコンパイルは可能だが CGO の制約上やや複雑になる
- Python / Node.js から Dolt を直接使う embedded driver は存在しないため、Dolt アクセスは instinct（Go 製）に集約する
