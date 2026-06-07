# Dolt 概要

## 一言で言うと

**MySQL の構文で操作できる Git**。公式キャッチコピーは "Git for Data"、"It's like Git and MySQL had a baby."

---

## 一般的な RDBMS との違い

| 機能 | 通常の RDBMS | Dolt |
|---|---|---|
| スキーマ/データの履歴 | なし（WAL は内部用途のみ） | Git のコミットログとして完全保持 |
| ブランチ・マージ | なし | あり（`dolt_checkout`, `dolt_merge`） |
| ロールバック | トランザクション内のみ | 任意のコミットへ `dolt_reset('--hard')` |
| セル単位の監査 | 別途監査テーブルを作る必要がある | `dolt_history_<table>` / `dolt_diff_<table>` で標準搭載 |
| リモート共有 | レプリケーション | `push/pull/clone`（GitHub 相当の DoltHub あり） |

バージョン管理操作は **SQL のストアドプロシージャ**（`CALL dolt_commit(...)` など）または **CLI**（`dolt commit`）で行う。

---

## 主なユースケース

1. **データ監査** — セル単位で「誰がいつ何を変えたか」を追跡（規制対応・金融・医療など）
2. **データセットの共同編集** — DoltHub 経由でバージョン管理されたデータを公開・共有
3. **スキーマ変更の安全な検証** — ブランチ上で試験し、問題なければ main にマージ
4. **AI エージェントのメモリ** — マルチエージェント構成でのブランチ分離・状態巻き戻しが容易
5. **本番データの誤操作リカバリ** — `DROP TABLE` 後も `dolt_reset('--hard')` で即復元

---

## 技術スタック

MySQL との互換性は **MySQL のコードを一切含まない**独自実装で実現している。

```
クライアント
    ↓ MySQL ワイヤプロトコル
dolthub/vitess  ← Vitess フォーク（プロトコル処理・SQL パーサ）
    ↓
Dolt クエリエンジン（Go 独自実装）
    ↓
Noms 由来の Content-Addressed ストレージ
```

- **dolthub/vitess**: PlanetScale/Google の Vitess をフォークし、MySQL ワイヤプロトコル（`go/mysql`）と SQL パーサ（`go/vt/sqlparser`）部分だけを使用。分散 DB 機能（VTGate・VTTablet 等）は不使用。
- **Noms**: コンテンツアドレス指定のストレージ。テーブルデータを Git の DAG と同じ構造で管理するため、ブランチ・マージ・差分計算が効率的に動作する。

---

## 既存プロジェクトへの組み込み

MySQL 互換なので JDBC / mysql2 / SQLAlchemy などの既存ドライバを接続先変更だけで利用可能。

```bash
# インストール（Linux/Mac）
sudo bash -c 'curl -L https://github.com/dolthub/dolt/releases/latest/download/install.sh | bash'

# サーバー起動（port 3306、MySQL 互換）
dolt sql-server
```

```yaml
# Docker Compose への追加
services:
  db:
    image: dolthub/dolt-sql-server:latest
    ports:
      - "3306:3306"
```

バージョン管理操作はアプリから SQL で呼ぶだけ：

```sql
CALL dolt_commit('-am', 'データを更新');
SELECT * FROM dolt_log;
SELECT * FROM dolt_diff_orders;
```

**注意点**:
- MySQL 8.4 までのクライアント互換。MySQL 9.0 は認証方式が変わり別途設定が必要
- Postgres 互換が必要なら [Doltgres](https://github.com/dolthub/doltgresql)（Beta）がある
- 書き込みパフォーマンスは通常の MySQL より若干落ちる（バージョン管理のオーバーヘッド）
