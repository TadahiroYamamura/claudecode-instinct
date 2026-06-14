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
```

### チームへの推薦（nominate）

```bash
# 推薦候補一覧（observation_count が閾値以上かつ main 未マージ）
instinct nominate list

# 指定 ID を review_queue に登録
instinct nominate <id...>
```

### レビューキューの確認・承認（review）

```bash
# review_queue の一覧を表示
instinct review list

# 指定 ID を承認してチームブランチに昇格
instinct review approve <id...>
```

### チームとの共有（push）

```bash
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
1. 各メンバーが instinct nominate <id...> で推薦候補を review_queue に登録

2. 各メンバーが instinct push で個人ブランチを push

3. レビュー担当者が instinct review list で review_queue を確認

4. レビュー担当者が instinct review approve <id...> で承認・チームブランチへ昇格

5. 各メンバーが instinct pull でチームブランチを取得
```

---

## モノレポでの利用

同一 GitHub リポジトリに複数プロジェクトがある場合、`config.team.yml` の `dolt.refs` でプロジェクトごとに異なる namespace を設定する。

```yaml
# our-project の config.team.yml
dolt:
  refs: "refs/dolt/our-project"

# their-project の config.team.yml
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

---

## architecture Decision Records

設計上の重要な決定は `docs/adr/` を参照。
