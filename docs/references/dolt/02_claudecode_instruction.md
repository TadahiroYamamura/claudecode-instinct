# Dolt を Claude Code セッションで扱う際に知るべきこと

このドキュメントは `01_overview.md` の補足として、Claude Code セッション内で Dolt を実装・設計する際に必要な実践的知識をまとめたものです。

---

## dolthub/driver: CLI なし・サーバーなしで Dolt を使う

Dolt は `dolt` コマンドをローカルにインストールしなくても使用できる。
Go の embedded driver `github.com/dolthub/driver` を使うと、SQLite と同じアーキテクチャで動作する。

```go
import (
    _ "github.com/dolthub/driver"
    "database/sql"
)

db, err := sql.Open("dolt", "file:///path/to/db-dir?commitname=Name&commitemail=email@example.com")
```

- `file://` DSN でローカルディレクトリを指定（サブディレクトリが Dolt DB として扱われる）
- **CGO 必須**（`go build` 時に `gcc` が必要）
- `dolt sql-server` は不要、`dolt` CLI も不要
- Knatch プロジェクト（`~/work/issues`）が実際の使用例

### Connector/Driver の役割

- `driver.go`: `init()` で `"dolt"` ドライバを `database/sql` に登録。`openSqlEngine()` が Dolt エンジンをプロセスにロードする
- `connector.go`: `NewConnector(cfg)` で接続設定。エンジンは遅延初期化・プロセス内で共有

---

## GitHub への push/pull

Dolt のデータは Git オブジェクトとして GitHub リポジトリに保存できる。
デフォルトの refs は `refs/dolt/data` だが、**カスタム refs** を設定することで複数プロジェクトの共存が可能。

```sql
-- カスタム refs でリモートを追加
CALL dolt_remote('add', '--ref', 'refs/dolt/oncall-platform', 'origin', 'git@github.com:ORG/REPO.git');

-- push（ブランチ名を明示）
CALL dolt_push('origin', 'my-branch');
-- → refs/dolt/oncall-platform/my-branch に格納される

-- pull
CALL dolt_pull('origin', 'main');
```

モノレポで複数プロジェクトが同一 GitHub リポジトリを使う場合、`--ref` で namespace を分けることで競合を防ぐ。

---

## ブランチ戦略

Dolt は Git と同様のブランチを持つ。SQL からブランチを横断してクエリできる。

```sql
-- 特定ブランチのテーブルを参照
SELECT * FROM instincts AS OF 'main';
SELECT * FROM instincts AS OF 'tadahiro';

-- ブランチ間の差分（追加されたレコードのみ）
SELECT * FROM dolt_diff_instincts
WHERE from_commit = HASHOF('main')
  AND to_commit   = HASHOF('tadahiro')
  AND diff_type   = 'added';

-- 複数ブランチのUNION
SELECT 'main' as branch, * FROM instincts AS OF 'main'
UNION ALL
SELECT 'tadahiro' as branch, * FROM instincts AS OF 'tadahiro';
```

### 典型的なブランチ運用パターン

| ブランチ | 用途 |
|---------|------|
| `main` | キュレーション済み・チーム共有 |
| `<username>` | 個人の instinct 生成・未レビュー |

---

## push/pull の SQL ストアドプロシージャ

```sql
-- コミット
CALL dolt_add('-A');
CALL dolt_commit('-m', 'Add new instincts');

-- push（origin の main ブランチへ）
CALL dolt_push('origin', 'main');

-- pull
CALL dolt_pull('origin', 'main');

-- リモート追加（カスタム refs あり）
CALL dolt_remote('add', '--ref', 'refs/dolt/my-namespace', 'origin', 'git@github.com:ORG/REPO.git');

-- ブランチ操作
CALL dolt_checkout('-b', 'new-branch');
CALL dolt_merge('feature-branch');
```

---

## dolt_diff_\<table\> システムテーブル

Dolt は各テーブルに対して自動的に `dolt_diff_<table>` システムテーブルを提供する。
これを使うと「前回のコミット以降に追加・変更・削除されたレコード」を SQL で取得できる。

```sql
-- main との差分で追加されたレコード
SELECT * FROM dolt_diff_instincts
WHERE from_commit = HASHOF('main')
  AND to_commit   = HASHOF('HEAD')
  AND diff_type   = 'added';

-- diff_type の値: 'added' | 'modified' | 'removed'
```

これは「レビュー待ちキュー」の実装に直接使える。

---

## 設計上の注意点

1. **CGO 必須**: `dolthub/driver` は CGO を使うため `gcc` が必要。CI 環境や Docker では注意。
2. **プロセス内シングルトン**: `openSem` (セマフォ) により同一プロセス内でエンジン初期化は直列化される。複数の `sql.Open` を呼んでも安全。
3. **モノレポでの refs 競合**: 同一 GitHub リポジトリに複数の Dolt DB をプッシュする場合は必ず `--ref` でnamespace を分けること。
4. **Python/Node.js からの利用**: `dolthub/driver` は Go 専用。Python や Node.js から使うには、Go で薄い CLI ラッパーを書いてサブプロセス呼び出しする方式が現実的。
