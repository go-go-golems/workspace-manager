#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../../../../.." && pwd)"

TMP_DIR="$(mktemp -d)"
cleanup() {
  rm -rf "${TMP_DIR}"
}
trap cleanup EXIT

PROGRAM="${TMP_DIR}/repro_hybrid_error_swallow.go"
LOG_FILE="${SCRIPT_DIR}/repro_hybrid_error_swallow.log"

cat >"${PROGRAM}" <<'EOF'
package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-go-golems/workspace-manager/pkg/wsm/gitclient"
)

type fakeRepo struct {
	path string
}

func (r *fakeRepo) Path() string {
	return r.path
}

type fakeClient struct {
	addErr     error
	pushErr    error
	addCalls   int
	pushCalls  int
	backendTag string
}

func (f *fakeClient) Open(ctx context.Context, repoPath string) (gitclient.RepositoryHandle, error) {
	return &fakeRepo{path: repoPath}, nil
}
func (f *fakeClient) CurrentBranch(ctx context.Context, repo gitclient.RepositoryHandle) (string, error) {
	return "main", nil
}
func (f *fakeClient) RemoteURL(ctx context.Context, repo gitclient.RepositoryHandle, remote string) (string, error) {
	return "", nil
}
func (f *fakeClient) LocalBranchExists(ctx context.Context, repo gitclient.RepositoryHandle, branch string) (bool, error) {
	return false, nil
}
func (f *fakeClient) ListLocalBranches(ctx context.Context, repo gitclient.RepositoryHandle) ([]string, error) {
	return []string{"main"}, nil
}
func (f *fakeClient) ListRemoteTrackingBranches(ctx context.Context, repo gitclient.RepositoryHandle, remote string) ([]string, error) {
	return []string{}, nil
}
func (f *fakeClient) RemoteTrackingBranchExists(ctx context.Context, repo gitclient.RepositoryHandle, remote string, branch string) (bool, error) {
	return false, nil
}
func (f *fakeClient) ListTags(ctx context.Context, repo gitclient.RepositoryHandle) ([]string, error) {
	return []string{}, nil
}
func (f *fakeClient) LastCommit(ctx context.Context, repo gitclient.RepositoryHandle) (string, error) {
	return "", nil
}
func (f *fakeClient) Status(ctx context.Context, repo gitclient.RepositoryHandle) (gitclient.Status, error) {
	return gitclient.Status{}, nil
}
func (f *fakeClient) Add(ctx context.Context, repo gitclient.RepositoryHandle, path string) error {
	f.addCalls++
	return f.addErr
}
func (f *fakeClient) Reset(ctx context.Context, repo gitclient.RepositoryHandle, path string) error {
	return nil
}
func (f *fakeClient) Commit(ctx context.Context, repo gitclient.RepositoryHandle, msg string, opts gitclient.CommitOptions) (string, error) {
	return "", nil
}
func (f *fakeClient) Diff(ctx context.Context, repo gitclient.RepositoryHandle, staged bool) (string, error) {
	return "", nil
}
func (f *fakeClient) Fetch(ctx context.Context, repo gitclient.RepositoryHandle, remote string) error {
	return nil
}
func (f *fakeClient) Push(ctx context.Context, repo gitclient.RepositoryHandle, remote string) error {
	f.pushCalls++
	return f.pushErr
}
func (f *fakeClient) AheadBehind(ctx context.Context, repo gitclient.RepositoryHandle, upstream string) (int, int, error) {
	return 0, 0, nil
}
func (f *fakeClient) CreateBranch(ctx context.Context, repo gitclient.RepositoryHandle, name string, track bool, baseRef string) error {
	return nil
}
func (f *fakeClient) CheckoutBranch(ctx context.Context, repo gitclient.RepositoryHandle, name string, create bool, force bool) error {
	return nil
}

func main() {
	ctx := context.Background()
	repo := &fakeRepo{path: "/tmp/repo"}

	fmt.Println("Case A: primary returns real errors, fallback should not run, error should bubble up")
	primary := &fakeClient{
		addErr:     errors.New("primary add failed"),
		pushErr:    errors.New("primary push failed"),
		backendTag: "primary",
	}
	fallback := &fakeClient{backendTag: "fallback"}
	h := gitclient.NewHybrid(primary, fallback)

	addErr := h.Add(ctx, repo, ".")
	pushErr := h.Push(ctx, repo, "origin")
	fmt.Printf("Add: err=%v primaryCalls=%d fallbackCalls=%d\n", addErr, primary.addCalls, fallback.addCalls)
	fmt.Printf("Push: err=%v primaryCalls=%d fallbackCalls=%d\n", pushErr, primary.pushCalls, fallback.pushCalls)

	fmt.Println()
	fmt.Println("Case B: primary returns ErrNotImplemented, fallback should run")
	primary2 := &fakeClient{
		addErr:     gitclient.ErrNotImplemented,
		pushErr:    gitclient.ErrNotImplemented,
		backendTag: "primary2",
	}
	fallback2 := &fakeClient{backendTag: "fallback2"}
	h2 := gitclient.NewHybrid(primary2, fallback2)
	addErr2 := h2.Add(ctx, repo, ".")
	pushErr2 := h2.Push(ctx, repo, "origin")
	fmt.Printf("Add: err=%v primaryCalls=%d fallbackCalls=%d\n", addErr2, primary2.addCalls, fallback2.addCalls)
	fmt.Printf("Push: err=%v primaryCalls=%d fallbackCalls=%d\n", pushErr2, primary2.pushCalls, fallback2.pushCalls)
}
EOF

{
  echo "# Gap 2 Reproduction: HybridClient error swallowing"
  echo "# Date: $(date -u +"%Y-%m-%dT%H:%M:%SZ")"
  echo
  (
    cd "${REPO_ROOT}"
    go run "${PROGRAM}"
  )
} >"${LOG_FILE}" 2>&1

echo "Wrote reproduction log: ${LOG_FILE}"
