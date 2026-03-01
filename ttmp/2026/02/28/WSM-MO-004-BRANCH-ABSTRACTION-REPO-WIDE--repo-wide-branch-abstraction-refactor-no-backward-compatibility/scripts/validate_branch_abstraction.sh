#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && git rev-parse --show-toplevel)"
cd "$ROOT_DIR"

echo "[WSM-MO-004] branch abstraction validation"
echo "repo: $ROOT_DIR"

go test ./pkg/wsm/branch -v
go test ./pkg/wsm/gitclient -run 'Hybrid|RemoteTracking|RemoteBranch|CreateBranch|GoGitCreateBranch' -v
go test ./pkg/wsm -run 'SyncSwitchBranch|BranchServiceRemoteTrackingExists|ResolveBranch|CreateWorktreeForAdd' -v

echo
echo "[WSM-MO-004] policy leakage scan"
if rg -n "ListBranches|CheckBranchExists\\(|CheckRemoteBranchExists\\(|origin/main\\b" pkg/wsm cmd/cmds; then
  echo "Leakage scan found remaining legacy patterns"
  exit 1
fi

echo "Validation complete: no legacy branch-policy patterns found."
