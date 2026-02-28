#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../../../../.." && pwd)"

TMP_DIR="$(mktemp -d)"
cleanup() {
  rm -rf "${TMP_DIR}"
}
trap cleanup EXIT

REMOTE="${TMP_DIR}/origin.git"
SEED="${TMP_DIR}/seed"
CLIENT="${TMP_DIR}/client"
CHECKER="${TMP_DIR}/check_remote_exists.go"
LOG_FILE="${SCRIPT_DIR}/repro_remote_branch_false_negative.log"

{
  echo "# Gap 1 Reproduction: Remote branch false negative"
  echo "# Date: $(date -u +"%Y-%m-%dT%H:%M:%SZ")"
  echo
  echo "## 1) Create remote + seed repository"
} >"${LOG_FILE}"

git init --bare "${REMOTE}" >>"${LOG_FILE}" 2>&1
git clone "${REMOTE}" "${SEED}" >>"${LOG_FILE}" 2>&1
git -C "${SEED}" config user.name "WSM Repro"
git -C "${SEED}" config user.email "wsm-repro@example.com"
git -C "${SEED}" checkout -b main >>"${LOG_FILE}" 2>&1
echo "seed" >"${SEED}/README.md"
git -C "${SEED}" add README.md >>"${LOG_FILE}" 2>&1
git -C "${SEED}" commit -m "seed commit" >>"${LOG_FILE}" 2>&1
git -C "${SEED}" push -u origin main >>"${LOG_FILE}" 2>&1
git -C "${SEED}" checkout -b feature/remote-only >>"${LOG_FILE}" 2>&1
echo "remote-only" >"${SEED}/feature.txt"
git -C "${SEED}" add feature.txt >>"${LOG_FILE}" 2>&1
git -C "${SEED}" commit -m "add remote-only feature branch" >>"${LOG_FILE}" 2>&1
git -C "${SEED}" push -u origin feature/remote-only >>"${LOG_FILE}" 2>&1

{
  echo
  echo "## 2) Create fresh client clone (no local feature branch)"
} >>"${LOG_FILE}"

git clone --branch main "${REMOTE}" "${CLIENT}" >>"${LOG_FILE}" 2>&1

{
  echo
  echo "git branch -a output:"
  git -C "${CLIENT}" branch -a
  echo
  echo "Ground truth using ls-remote:"
  git -C "${CLIENT}" ls-remote --heads origin feature/remote-only
} >>"${LOG_FILE}" 2>&1

cat >"${CHECKER}" <<'EOF'
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/go-go-golems/workspace-manager/pkg/wsm"
)

func main() {
	if len(os.Args) != 4 {
		fmt.Println("usage: check_remote_exists <repoPath> <branch> <backend>")
		os.Exit(2)
	}

	repoPath := os.Args[1]
	branch := os.Args[2]
	backend := os.Args[3]

	_ = os.Setenv("WSM_GIT_BACKEND", backend)
	wm := &wsm.WorkspaceManager{}
	ok, err := wm.CheckRemoteBranchExists(context.Background(), repoPath, branch)
	fmt.Printf("backend=%s exists=%v err=%v\n", backend, ok, err)
}
EOF

{
  echo
  echo "## 3) Call actual CheckRemoteBranchExists()"
} >>"${LOG_FILE}"

for backend in cli gogit hybrid; do
  (
    cd "${REPO_ROOT}"
    go run "${CHECKER}" "${CLIENT}" "feature/remote-only" "${backend}"
  ) >>"${LOG_FILE}" 2>&1
done

{
  echo
  echo "## 4) Expected vs actual"
  echo "Expected: true (origin has feature/remote-only)"
  echo "Actual: see backend lines above"
} >>"${LOG_FILE}"

echo "Wrote reproduction log: ${LOG_FILE}"
