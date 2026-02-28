package gitclient

import (
	"context"
	stderrors "errors"
	"strings"
)

// HybridClient uses a primary client and falls back to a secondary client when
// the primary returns ErrNotImplemented.
type HybridClient struct {
	primary  GitClient
	fallback GitClient
}

func NewHybrid(primary GitClient, fallback GitClient) *HybridClient {
	return &HybridClient{primary: primary, fallback: fallback}
}

func shouldFallback(err error) bool {
	return stderrors.Is(err, ErrNotImplemented)
}

func shouldFallbackPush(err error) bool {
	if shouldFallback(err) {
		return true
	}
	if err == nil {
		return false
	}
	// go-git can fail to resolve remotes in worktree paths; CLI backend handles this reliably.
	return strings.Contains(strings.ToLower(err.Error()), "remote not found")
}

func (h *HybridClient) Open(ctx context.Context, repoPath string) (RepositoryHandle, error) {
	r, err := h.primary.Open(ctx, repoPath)
	if shouldFallback(err) {
		return h.fallback.Open(ctx, repoPath)
	}
	return r, err
}

func (h *HybridClient) CurrentBranch(ctx context.Context, repo RepositoryHandle) (string, error) {
	s, err := h.primary.CurrentBranch(ctx, repo)
	if shouldFallback(err) {
		return h.fallback.CurrentBranch(ctx, repo)
	}
	return s, err
}
func (h *HybridClient) RemoteURL(ctx context.Context, repo RepositoryHandle, remote string) (string, error) {
	s, err := h.primary.RemoteURL(ctx, repo, remote)
	if shouldFallback(err) {
		return h.fallback.RemoteURL(ctx, repo, remote)
	}
	return s, err
}
func (h *HybridClient) LocalBranchExists(ctx context.Context, repo RepositoryHandle, branch string) (bool, error) {
	ok, err := h.primary.LocalBranchExists(ctx, repo, branch)
	if shouldFallback(err) {
		return h.fallback.LocalBranchExists(ctx, repo, branch)
	}
	return ok, err
}
func (h *HybridClient) ListLocalBranches(ctx context.Context, repo RepositoryHandle) ([]string, error) {
	s, err := h.primary.ListLocalBranches(ctx, repo)
	if shouldFallback(err) {
		return h.fallback.ListLocalBranches(ctx, repo)
	}
	return s, err
}
func (h *HybridClient) ListRemoteTrackingBranches(ctx context.Context, repo RepositoryHandle, remote string) ([]string, error) {
	s, err := h.primary.ListRemoteTrackingBranches(ctx, repo, remote)
	if shouldFallback(err) {
		return h.fallback.ListRemoteTrackingBranches(ctx, repo, remote)
	}
	return s, err
}
func (h *HybridClient) RemoteTrackingBranchExists(ctx context.Context, repo RepositoryHandle, remote string, branch string) (bool, error) {
	ok, err := h.primary.RemoteTrackingBranchExists(ctx, repo, remote, branch)
	if shouldFallback(err) {
		return h.fallback.RemoteTrackingBranchExists(ctx, repo, remote, branch)
	}
	return ok, err
}
func (h *HybridClient) ListTags(ctx context.Context, repo RepositoryHandle) ([]string, error) {
	s, err := h.primary.ListTags(ctx, repo)
	if shouldFallback(err) {
		return h.fallback.ListTags(ctx, repo)
	}
	return s, err
}
func (h *HybridClient) LastCommit(ctx context.Context, repo RepositoryHandle) (string, error) {
	s, err := h.primary.LastCommit(ctx, repo)
	if shouldFallback(err) {
		return h.fallback.LastCommit(ctx, repo)
	}
	return s, err
}
func (h *HybridClient) Status(ctx context.Context, repo RepositoryHandle) (Status, error) {
	s, err := h.primary.Status(ctx, repo)
	if shouldFallback(err) {
		return h.fallback.Status(ctx, repo)
	}
	return s, err
}
func (h *HybridClient) Add(ctx context.Context, repo RepositoryHandle, path string) error {
	err := h.primary.Add(ctx, repo, path)
	if shouldFallback(err) {
		return h.fallback.Add(ctx, repo, path)
	}
	return err
}
func (h *HybridClient) Reset(ctx context.Context, repo RepositoryHandle, path string) error {
	err := h.primary.Reset(ctx, repo, path)
	if shouldFallback(err) {
		return h.fallback.Reset(ctx, repo, path)
	}
	return err
}
func (h *HybridClient) Commit(ctx context.Context, repo RepositoryHandle, msg string, opts CommitOptions) (string, error) {
	s, err := h.primary.Commit(ctx, repo, msg, opts)
	if shouldFallback(err) {
		return h.fallback.Commit(ctx, repo, msg, opts)
	}
	return s, err
}
func (h *HybridClient) Diff(ctx context.Context, repo RepositoryHandle, staged bool) (string, error) {
	s, err := h.primary.Diff(ctx, repo, staged)
	if shouldFallback(err) {
		return h.fallback.Diff(ctx, repo, staged)
	}
	return s, err
}
func (h *HybridClient) Fetch(ctx context.Context, repo RepositoryHandle, remote string) error {
	err := h.primary.Fetch(ctx, repo, remote)
	if shouldFallback(err) {
		return h.fallback.Fetch(ctx, repo, remote)
	}
	return err
}
func (h *HybridClient) Push(ctx context.Context, repo RepositoryHandle, remote string) error {
	err := h.primary.Push(ctx, repo, remote)
	if shouldFallbackPush(err) {
		return h.fallback.Push(ctx, repo, remote)
	}
	return err
}
func (h *HybridClient) AheadBehind(ctx context.Context, repo RepositoryHandle, upstream string) (int, int, error) {
	a, b, err := h.primary.AheadBehind(ctx, repo, upstream)
	if shouldFallback(err) {
		return h.fallback.AheadBehind(ctx, repo, upstream)
	}
	return a, b, err
}
func (h *HybridClient) CreateBranch(ctx context.Context, repo RepositoryHandle, name string, track bool, baseRef string) error {
	err := h.primary.CreateBranch(ctx, repo, name, track, baseRef)
	if shouldFallback(err) {
		return h.fallback.CreateBranch(ctx, repo, name, track, baseRef)
	}
	return err
}
func (h *HybridClient) CheckoutBranch(ctx context.Context, repo RepositoryHandle, name string, create bool, force bool) error {
	err := h.primary.CheckoutBranch(ctx, repo, name, create, force)
	if shouldFallback(err) {
		return h.fallback.CheckoutBranch(ctx, repo, name, create, force)
	}
	return err
}
