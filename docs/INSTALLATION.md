# インストールガイド

## 対応OS

| OS | アーキテクチャ |
|----|---------------|
| Linux | x86_64 (amd64) |
| macOS | Apple Silicon (arm64) |

## 前提条件

| 要件 | バージョン | 備考 |
|------|-----------|------|
| Claude Code | v2.1 以上 | |
| git | 任意 | `user.name` / `user.email` の設定が必須 |

git の設定が未済の場合は先に行う。

```bash
git config --global user.name  "Your Name"
git config --global user.email "you@example.com"
```

## 1. instinct のインストール

プラグインのインストール前に `instinct` バイナリをダウンロードして PATH に置く。

**Linux (amd64)**

```bash
curl -L -o instinct https://github.com/TadahiroYamamura/claudecode-instinct/releases/latest/download/instinct-linux-amd64
chmod +x instinct
sudo mv instinct /usr/local/bin/
```

**macOS (Apple Silicon)**

```bash
curl -L -o instinct https://github.com/TadahiroYamamura/claudecode-instinct/releases/latest/download/instinct-darwin-arm64
chmod +x instinct
sudo mv instinct /usr/local/bin/
```

動作確認。

```bash
instinct --help
```

## 2. プラグインのインストール

### GitHub からインストール（公開後）

リポジトリをマーケットプレイスとして登録してからインストールする。

```bash
claude plugin marketplace add TadahiroYamamura/claudecode-instinct
claude plugin install claudecode-instinct@TadahiroYamamura
```

## 3. プロジェクトへのセットアップ

instinct を記録したいプロジェクトのルートで一度だけ実行する。`.instinct-db/` が作成される。

```bash
cd /path/to/your-project
instinct setup
```

対話形式で branch / team_branch / remote_url を確認・変更できる。Enter でデフォルト値を採用。`-y` フラグを付けると全項目デフォルトで非対話実行。

```bash
instinct setup -y   # CI や自動化環境向け
```

作成されるファイル。

```
your-project/
└── .instinct-db/
    ├── data/         # Dolt DB 本体（git 管理外）
    ├── .gitignore    # ランタイムファイルの除外ルール（自動生成）
    └── config.yml    # プロジェクト固有設定（git 管理）
```

`config.yml` の初期内容（`instinct setup` が自動生成）。

```yaml
observer:
  enabled: true
  trigger_every: 20
  active_hours: "800-2300"

confidence:
  thresholds:
    low: 3
    medium: 6
    high: 11

dedup:
  auto_run_before_push: false

dolt:
  refs: refs/dolt/your-project/
  branch: tadahiro                        # git config user.name から自動取得
  team_branch: main                       # チームブランチ名
  remote_url: git@github.com:org/repo.git # git remote origin から自動取得
```

## 4. 動作確認

observer-loop は Claude Code セッション開始時に自動起動する（SessionStart フック）。

Claude Code を通常通り起動して作業する。

```bash
claude
```

20 ツール操作が蓄積されると observer-loop が自動的に instinct 生成を試みる。
生成された instinct は次のコマンドで確認できる（`list` サブコマンドは Phase 2 実装予定）。

現時点では `instinct list` が未実装（Phase 2 予定）のため、Dolt CLI で直接確認する。

```bash
dolt --data-dir=.instinct-db/data sql -q "SELECT content, trigger_desc, scope FROM instincts ORDER BY created_at DESC LIMIT 10"
```

Dolt CLI がなければ MySQL クライアント（mysql コマンド）でも接続できる。

## アンインストール

```bash
# バイナリを削除
rm /usr/local/bin/instinct

# プラグインを削除
claude plugin uninstall claudecode-instinct@TadahiroYamamura

# GitHubからDoltを削除（任意）
git push origin --delete refs/dolt/<your-project> # 実際のrefsの値は.instinct-db/config_team.ymlを参照してください
git push origin --delete __dolt_remote_info__

# プロジェクトの DB を削除（任意）
rm -rf .instinct-db/
```
