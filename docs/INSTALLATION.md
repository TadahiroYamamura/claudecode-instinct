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

```bash
claude plugin install TadahiroYamamura/claudecode-instinct
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
    ├── data/       # Dolt DB 本体（git 管理外）
    └── config.yml  # プロジェクト固有設定（git 管理）
```

`.gitignore` に `.instinct-db/data/` を追加する。

```bash
echo '.instinct-db/data/' >> .gitignore
```

`config.yml` を必要に応じて編集する（`dolt.refs` はプロジェクト名から自動推定済み）。

```yaml
observer:
  enabled: true
  trigger_every: 20       # 何観察ごとに Haiku を起動するか
  active_hours: "800-2300"

dolt:
  remote_url: "git@github.com:ORG/REPO.git"
  refs: "refs/dolt/your-project"
```

---

## 4. observer-loop の起動

Claude Code を使い始める前に、バックグラウンドで observer-loop を起動する。

```bash
PLUGIN_ROOT="$(claude plugin path TadahiroYamamura/claudecode-instinct)"
PROJECT_ROOT="$(pwd)"   # .instinct-db があるディレクトリ

nohup bash "${PLUGIN_ROOT}/skills/continuous-learning/agents/observer-loop.sh" \
  "${PROJECT_ROOT}" \
  > "${PROJECT_ROOT}/.observer.log" 2>&1 &

echo "observer PID: $!"
```

停止するには `.observer.pid` に記録された PID を kill する。

```bash
kill "$(cat "${PROJECT_ROOT}/.observer.pid")"
```

---

## 5. 動作確認

Claude Code を通常通り起動して作業する。

```bash
claude
```

20 ツール操作が蓄積されると observer-loop が自動的に instinct 生成を試みる。
生成された instinct は次のコマンドで確認できる（`list` サブコマンドは Phase 2 実装予定）。

```bash
# 現時点での確認方法（直接 DB を参照）
sqlite3 .instinct-db/data/instincts/... # Dolt 形式のため dolt CLI が必要
```

---

## アンインストール

```bash
# プラグインを削除
claude plugin uninstall TadahiroYamamura/claudecode-instinct

# プロジェクトの DB を削除（任意）
rm -rf .instinct-db/
```
