# ADR-0005: モノレポ対応のカスタム Dolt refs

## Status

Accepted

## Context

複数のプロジェクト（例: `backend`, `mobile`）が同一 GitHub リポジトリ内に存在するモノレポ構成では、Dolt のデフォルト refs（`refs/dolt/data`）に両プロジェクトが push するとデータが上書きされる。

## Decision

`dolt remote add` の `--ref` オプションでプロジェクトごとに異なる refs namespace を設定する。

```bash
CALL dolt_remote('add', '--ref', 'refs/dolt/backend', 'origin', 'git@github.com:ORG/REPO.git');
CALL dolt_remote('add', '--ref', 'refs/dolt/mobile',  'origin', 'git@github.com:ORG/REPO.git');
```

push 結果:
- backend の instinct → `refs/dolt/backend/<branch>`
- mobile の instinct → `refs/dolt/mobile/<branch>`

この refs 値はプロジェクトルートの `.instinct-db/config.yml` に記載し、`instinct-cli setup` 実行時の cwd 名（`basename $(pwd)`）から自動推定する。`setup` はその cwd に `.instinct-db/` を作成するため、以降はこのディレクトリがプロジェクトルートとして機能する。

### プロジェクト ID の生成

モノレポ内の複数サブプロジェクトは git remote URL が同じになるため、URL だけでは識別できない。以下の方法で一意な project_id を生成する。

```python
hash_input = f"{git_remote_url}#{relative_path_from_git_root}"
# 例: "github.com/ORG/REPO#backend"
project_id = sha256(hash_input)[:12]
```

## Consequences

- モノレポ内の複数プロジェクトが同一 GitHub リポジトリを共有しても instinct データが競合しない
- `config.yml` の `dolt.refs` を変更するだけで refs を切り替えられる
- `instinct-cli setup` が自動推定するため、手動設定は最小限
- プロジェクト ID が URL + 相対パスのハッシュになることで、異なるマシン間でも同一プロジェクトを識別できる
