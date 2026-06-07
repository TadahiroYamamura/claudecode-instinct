# インストールガイド

## 前提条件

| 要件 | バージョン | 備考 |
|------|-----------|------|
| Claude Code | v2.1 以上 | |
| Go | 1.22 以上 | `gcc` も必要（CGO ビルド） |
| git | 任意 | `user.name` / `user.email` の設定が必須 |

git の設定が未済の場合は先に行う。

```bash
git config --global user.name  "Your Name"
git config --global user.email "you@example.com"
```

---

## 1. instinct-cli のビルド

プラグインのインストール前に `instinct-cli` バイナリをビルドして PATH に置く。

```bash
# リポジトリを取得
git clone https://github.com/TadahiroYamamura/claudecode-instinct.git
cd claudecode-instinct/cmd/instinct-cli

# ビルド（CGO が必要なため gcc が必要）
go build -o instinct-cli .

# PATH の通った場所に配置
sudo mv instinct-cli /usr/local/bin/
# または
mv instinct-cli ~/bin/   # ~/bin が PATH に含まれている場合
```

動作確認。

```bash
instinct-cli --help
```

---

## 2. プラグインのインストール

### GitHub からインストール（公開後）

リポジトリをマーケットプレイスとして登録してからインストールする。

```bash
claude plugin marketplace add TadahiroYamamura/claudecode-instinct
claude plugin install claudecode-instinct@TadahiroYamamura
```

### ローカルパスからインストール（開発中・手元で試す場合）

クローンしたディレクトリをマーケットプレイスとして登録する。

```bash
claude plugin marketplace add /path/to/claudecode-instinct
claude plugin install claudecode-instinct@claudecode-instinct
```

---

## 3. プロジェクトへのセットアップ

instinct を記録したいプロジェクトのルートで一度だけ実行する。`.instinct-db/` が作成される。

```bash
cd /path/to/your-project
instinct-cli setup
```

作成されるファイル。

```
your-project/
└── .instinct-db/
    ├── data/         # Dolt DB 本体（git 管理外）
    ├── .gitignore    # ランタイムファイルの除外ルール（自動生成）
    └── config.yml    # プロジェクト固有設定（git 管理）
```

`config.yml` の初期内容（`instinct-cli setup` が自動生成）。

```yaml
dolt:
  refs: refs/dolt/your-project/
```

`observer.*` / `dolt.remote_url` などの追加設定は Phase 2 以降で対応予定。

---

## 4. 動作確認

observer-loop は Claude Code セッション開始時に自動起動する（SessionStart フック）。

Claude Code を通常通り起動して作業する。

```bash
claude
```

20 ツール操作が蓄積されると observer-loop が自動的に instinct 生成を試みる。
生成された instinct は次のコマンドで確認できる（`list` サブコマンドは Phase 2 実装予定）。

現時点では `instinct-cli list` が未実装（Phase 2 予定）のため、Dolt CLI で直接確認する。

```bash
dolt --data-dir=.instinct-db/data sql -q "SELECT content, trigger_desc, scope FROM instincts ORDER BY created_at DESC LIMIT 10"
```

Dolt CLI がなければ MySQL クライアント（mysql コマンド）でも接続できる。

---

## アンインストール

```bash
# プラグインを削除（GitHub からインストールした場合）
claude plugin uninstall claudecode-instinct@TadahiroYamamura
# ローカルパスからインストールした場合
claude plugin uninstall claudecode-instinct@claudecode-instinct

# プロジェクトの DB を削除（任意）
rm -rf .instinct-db/
```
