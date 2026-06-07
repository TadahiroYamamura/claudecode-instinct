# claudecode-instinct

Claude Code の PreToolUse/PostToolUse フックでツール使用を自動観察し、Haiku エージェントがプロジェクト固有の作業パターン（**instinct**）を学習・蓄積する Claude Code プラグイン。蓄積した instinct は Dolt（Git-like DB）を通じて GitHub 経由でチームと共有できる。

---

## 前提条件

- Claude Code v2.1 以上
- Go 1.22 以上（`gcc` も必要、CGO ビルドのため）
- Node.js 18 以上
- Bash

CI 環境では Claude Code を `--bare` フラグで起動することでプラグインを無効化できる。

---

## インストール

詳細な手順は [docs/INSTALLATION.md](docs/INSTALLATION.md) を参照。

```bash
# instinct-cli をビルドして PATH に配置した後
claude plugin install TadahiroYamamura/claudecode-instinct
```

---

## プロジェクトへのセットアップ

プラグインをインストールしたプロジェクトのルートで一度だけ実行する。

```bash
# Dolt DB を初期化し、GitHub リモートを設定する
# remote_url と refs は自動推定されるが、確認・修正できる
instinct-cli setup
```

実行すると `<PROJECT_ROOT>/.instinct-db/` が作成される。

```
.instinct-db/
├── data/       # Dolt DB 本体（.gitignore に追加済み）
└── config.yml  # プロジェクト固有設定（git 管理）
```

### config.yml のカスタマイズ

```yaml
observer:
  enabled: true
  trigger_every: 20       # 何観察ごとに Haiku を起動するか
  active_hours: "800-2300"

dolt:
  remote_url: "git@github.com:ORG/REPO.git"
  refs: "refs/dolt/project-name"  # モノレポの場合はプロジェクト名で分ける
```

---

## 基本的な使い方

### 観察と instinct 生成（自動）

Claude Code を通常通り使うだけで観察が蓄積される。20 観察ごとに Haiku エージェントが自動起動し、instinct を生成・保存する。

### instinct の確認

```bash
# 自分のブランチの instinct 一覧
instinct-cli list

# チームブランチ（main）+ 自分の instinct を重複排除して表示
instinct-cli list --merged
```

### 重複排除

```bash
# 自分のブランチ内の重複を排除（Haiku エージェントが判定）
instinct-cli dedup

# 複数の個人ブランチを横断して重複排除
instinct-cli dedup --cross-branch tadahiro,kenji,alice
```

### チームとの共有（push）

```bash
# レビュー待ちキュー（main にない新規 instinct）を確認
instinct-cli review

# 個人ブランチを GitHub に push
instinct-cli push
```

### チームの instinct を取得（pull）

```bash
# キュレーション済みのチームブランチ（main）を取得
instinct-cli pull
```

---

## レビューワークフロー

チームで instinct を共有する際の標準的な流れ。

```
1. 各メンバーが instinct-cli push で個人ブランチを push

2. レビュー担当者が instinct-cli review で候補を確認
   （main にない、新規追加された instinct の一覧）

3. 必要に応じて instinct-cli dedup --cross-branch で
   メンバー間の重複を排除

4. 手動レビュー・承認

5. main ブランチにマージ

6. 各メンバーが instinct-cli pull で取得
```

---

## モノレポでの利用

同一 GitHub リポジトリに複数プロジェクトがある場合、`config.yml` の `dolt.refs` でプロジェクトごとに異なる namespace を設定する。

```yaml
# oncall-platform の config.yml
dolt:
  refs: "refs/dolt/oncall-platform"

# oncall-flutter の config.yml
dolt:
  refs: "refs/dolt/oncall-flutter"
```

これにより `refs/dolt/oncall-platform/<branch>` と `refs/dolt/oncall-flutter/<branch>` に分離されて格納される。

---

## CI での利用

CI 環境ではプラグインのフックが不要なため、`--bare` フラグで無効化する。

```bash
claude --bare -p "your prompt" --allowedTools "Read,Edit,Bash"
```

---

## dedup_decisions の活用

`instinct-cli dedup` の実行結果は `.instinct-db/data/` の `dedup_decisions` テーブルに記録される。「何を同じとみなしたか」の判断データが蓄積され、将来的な dedup モデルの訓練データとして活用できる。

```bash
# Haiku の判定に誤りがあった場合は human_label で訂正
instinct-cli dedup-label <decision-id> --label wrong
```

---

## architecture Decision Records

設計上の重要な決定は `docs/adr/` を参照。
