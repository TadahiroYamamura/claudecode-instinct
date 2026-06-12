# claudecode-instinct

Claude Code の PreToolUse/PostToolUse フックでツール使用を自動観察し、Haiku エージェントがプロジェクト固有の作業パターン（**instinct**）を学習・蓄積する Claude Code プラグイン。蓄積した instinct は Dolt（Git-like DB）を通じて GitHub 経由でチームと共有できる。

## 対応OS

| OS | アーキテクチャ |
|----|---------------|
| Linux | x86_64 (amd64) |
| macOS | Apple Silicon (arm64) |

---

## インストール

[docs/INSTALLATION.md](docs/INSTALLATION.md) を参照。

---

## 基本的な使い方

### 観察と instinct 生成（自動）

Claude Code を通常通り使うだけで観察が蓄積される。20 観察ごとに Haiku エージェントが自動起動し、instinct を生成・保存する。

### instinct の確認

```bash
# 自分のブランチの instinct 一覧
instinct list

# チームブランチ（main）+ 自分の instinct を重複排除して表示
instinct list --merged
```

### 重複排除

```bash
# 自分のブランチ内の重複を排除（Haiku エージェントが判定）
instinct dedup

# 複数の個人ブランチを横断して重複排除
instinct dedup --cross-branch tadahiro,kenji,alice
```

### チームとの共有（push）

```bash
# レビュー待ちキュー（main にない新規 instinct）を確認
instinct review

# 個人ブランチを GitHub に push
instinct push
```

### チームの instinct を取得（pull）

```bash
# キュレーション済みのチームブランチ（main）を取得
instinct pull
```

---

## レビューワークフロー

チームで instinct を共有する際の標準的な流れ。

```
1. 各メンバーが instinct push で個人ブランチを push

2. レビュー担当者が instinct review で候補を確認
   （main にない、新規追加された instinct の一覧）

3. 必要に応じて instinct dedup --cross-branch で
   メンバー間の重複を排除

4. 手動レビュー・承認

5. main ブランチにマージ

6. 各メンバーが instinct pull で取得
```

---

## モノレポでの利用

同一 GitHub リポジトリに複数プロジェクトがある場合、`config.yml` の `dolt.refs` でプロジェクトごとに異なる namespace を設定する。

```yaml
# our-project の config.yml
dolt:
  refs: "refs/dolt/our-project"

# their-project の config.yml
dolt:
  refs: "refs/dolt/their-project"
```

これにより `refs/dolt/our-project/<branch>` と `refs/dolt/their-project/<branch>` に分離されて格納される。

---

## CI での利用

CI 環境ではプラグインのフックが不要なため、`--bare` フラグで無効化する。

```bash
claude --bare -p "your prompt" --allowedTools "Read,Edit,Bash"
```

---

## dedup_decisions の活用

`instinct dedup` の実行結果は `.instinct-db/data/` の `dedup_decisions` テーブルに記録される。「何を同じとみなしたか」の判断データが蓄積され、将来的な dedup モデルの訓練データとして活用できる。

```bash
# Haiku の判定に誤りがあった場合は human_label で訂正
instinct dedup-label <decision-id> --label wrong
```

---

## architecture Decision Records

設計上の重要な決定は `docs/adr/` を参照。
