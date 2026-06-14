#!/bin/bash
# Dolt がリモート（GitHub）に作成した refs を全て削除する。
# E2E テスト終了後のクリーンアップに使用。
#
# Usage: cleanup-dolt-remote.sh [--dry-run] <remote-url> <refs-namespace>
# Example: cleanup-dolt-remote.sh git@github.com:user/repo.git refs/dolt/e2e-instinct-test
#          cleanup-dolt-remote.sh --dry-run git@github.com:user/repo.git refs/dolt/e2e-instinct-test
#
# Dolt が作成する refs:
#   <refs-namespace>          ... ブランチデータ（例: refs/dolt/myproject）
#   __dolt_remote_info__      ... Dolt メタデータ（全プロジェクト共通）
set -euo pipefail

DRY_RUN=false
if [ "${1:-}" = "--dry-run" ]; then
    DRY_RUN=true
    shift
fi

REMOTE_URL="${1:?Usage: $0 [--dry-run] <remote-url> <refs-namespace>}"
REFS_NAMESPACE="${2:?Usage: $0 [--dry-run] <remote-url> <refs-namespace>}"

all_remote_refs=$(git ls-remote "$REMOTE_URL" 2>/dev/null | awk '{print $2}')

refs_to_delete=$(echo "$all_remote_refs" | grep -E \
    "^${REFS_NAMESPACE}$|^${REFS_NAMESPACE}/|^refs/heads/__dolt_remote_info__$" || true)

if [ -z "$refs_to_delete" ]; then
    echo "No Dolt refs found under ${REFS_NAMESPACE} (or __dolt_remote_info__) on remote."
    exit 0
fi

if [ "$DRY_RUN" = true ]; then
    echo "[dry-run] Would delete the following refs from ${REMOTE_URL}:"
    echo "$refs_to_delete" | while IFS= read -r ref; do
        echo "  ${ref}"
    done
    exit 0
fi

echo "$refs_to_delete" | while IFS= read -r ref; do
    echo "Deleting ${ref} ..."
    git push "$REMOTE_URL" ":${ref}" 2>&1 || echo "  Warning: failed to delete ${ref}"
done

echo "Cleanup complete."
