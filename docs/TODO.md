# TODO

## Phase 1: db.go の責務整理

### Step 1: failing test を直す ✅
- [x] `execInit`：`teamBranch != "main"` のとき `CALL dolt_checkout('-b', teamBranch)` を発行する

### Step 2: dolt初期化の責務を整理する
- [x] `execInit` に初回 `CALL dolt_commit('-Am', 'init: create schema')` を追加する
  - `setupDB` はDDLのみ、コミットは `execInit` の責務
  - `setupInitPath` は変更なし（`setup` は将来削除予定のため）
- [x] `doltInit` を廃止する
  - スタブ（`return nil`）で未使用
  - `TestDoltInit_CreatesDoltFileInTargetDirectory` を削除（`TestInit_CreatesDoltDBWithoutRemote` でカバー済み）

### Step 3: `openConn` の未初期化DBケースをテストする ✅
- [x] `instincts` DB が存在しない状態で `openConn` を呼ぶとエラーになることを確認するテストを追加

---

## Phase 2: `connect` コマンドの実装

`setup` コマンドを `init` + `connect` に分割する。`setup` は削除。

- [x] `connect` コマンドの実装（1人目: push path）
- [x] `connect` コマンドの実装（2人目: clone path）
- [x] `connect` を dispatch に登録
- [x] `setup` コマンドとそのテストを削除

---

## Phase 3: Repository インターフェースを切る

Phase 2 完了後に、浮き上がる境界を確認してから設計する。

### 方針
- `Repository` インターフェース（`InsertInstinct`, `ListInstincts`, `GetInstinct` 等）を定義
- `execInsert` / `execList` / `execDedup` 等のユースケースはインターフェース経由にする
- `DoltRepository` が実装を持つ
- `push` / `pull` / `clone` は既存の関数型インジェクション（`doltPushFunc` 等）と統一する

### 期待する効果
- ユースケースのテストに実Doltが不要になる
- Dolt固有のテスト（SQL正当性）は `DoltRepository` 側に集約される

---

## メモ

- `insert` / `list` / `show` / `dedup` / `review` は現状すべて実Doltが必要なテスト構造
- `push` / `pull` / `clone` の関数型インジェクション（`doltPushFunc` 等）は Phase 3 の参考にする
