package gitclient

import "context"

// HybridClient uses a primary client and falls back to a secondary client when
// the primary returns ErrNotImplemented.
type HybridClient struct{
    primary GitClient
    fallback GitClient
}

func NewHybrid(primary GitClient, fallback GitClient) *HybridClient {
    return &HybridClient{primary: primary, fallback: fallback}
}

func (h *HybridClient) Open(ctx context.Context, repoPath string) (RepositoryHandle, error) {
    return h.primary.Open(ctx, repoPath)
}

func (h *HybridClient) CurrentBranch(ctx context.Context, repo RepositoryHandle) (string, error) {
    s, err := h.primary.CurrentBranch(ctx, repo)
    if err == ErrNotImplemented { return h.fallback.CurrentBranch(ctx, repo) }
    return s, err
}
func (h *HybridClient) RemoteURL(ctx context.Context, repo RepositoryHandle, remote string) (string, error) {
    s, err := h.primary.RemoteURL(ctx, repo, remote)
    if err == ErrNotImplemented { return h.fallback.RemoteURL(ctx, repo, remote) }
    return s, err
}
func (h *HybridClient) ListBranches(ctx context.Context, repo RepositoryHandle) ([]string, error) {
    s, err := h.primary.ListBranches(ctx, repo)
    if err == ErrNotImplemented { return h.fallback.ListBranches(ctx, repo) }
    return s, err
}
func (h *HybridClient) ListTags(ctx context.Context, repo RepositoryHandle) ([]string, error) {
    s, err := h.primary.ListTags(ctx, repo)
    if err == ErrNotImplemented { return h.fallback.ListTags(ctx, repo) }
    return s, err
}
func (h *HybridClient) LastCommit(ctx context.Context, repo RepositoryHandle) (string, error) {
    s, err := h.primary.LastCommit(ctx, repo)
    if err == ErrNotImplemented { return h.fallback.LastCommit(ctx, repo) }
    return s, err
}
func (h *HybridClient) Status(ctx context.Context, repo RepositoryHandle) (Status, error) {
    s, err := h.primary.Status(ctx, repo)
    if err == ErrNotImplemented { return h.fallback.Status(ctx, repo) }
    return s, err
}
func (h *HybridClient) Add(ctx context.Context, repo RepositoryHandle, path string) error {
    if err := h.primary.Add(ctx, repo, path); err == ErrNotImplemented { return h.fallback.Add(ctx, repo, path) }
    return nil
}
func (h *HybridClient) Reset(ctx context.Context, repo RepositoryHandle, path string) error {
    if err := h.primary.Reset(ctx, repo, path); err == ErrNotImplemented { return h.fallback.Reset(ctx, repo, path) }
    return nil
}
func (h *HybridClient) Commit(ctx context.Context, repo RepositoryHandle, msg string, opts CommitOptions) (string, error) {
    s, err := h.primary.Commit(ctx, repo, msg, opts)
    if err == ErrNotImplemented { return h.fallback.Commit(ctx, repo, msg, opts) }
    return s, err
}
func (h *HybridClient) Diff(ctx context.Context, repo RepositoryHandle, staged bool) (string, error) {
    s, err := h.primary.Diff(ctx, repo, staged)
    if err == ErrNotImplemented { return h.fallback.Diff(ctx, repo, staged) }
    return s, err
}
func (h *HybridClient) Fetch(ctx context.Context, repo RepositoryHandle, remote string) error {
    if err := h.primary.Fetch(ctx, repo, remote); err == ErrNotImplemented { return h.fallback.Fetch(ctx, repo, remote) }
    return nil
}
func (h *HybridClient) Push(ctx context.Context, repo RepositoryHandle, remote string) error {
    if err := h.primary.Push(ctx, repo, remote); err == ErrNotImplemented { return h.fallback.Push(ctx, repo, remote) }
    return nil
}
func (h *HybridClient) AheadBehind(ctx context.Context, repo RepositoryHandle, upstream string) (int, int, error) {
    a, b, err := h.primary.AheadBehind(ctx, repo, upstream)
    if err == ErrNotImplemented { return h.fallback.AheadBehind(ctx, repo, upstream) }
    return a, b, err
}
func (h *HybridClient) CreateBranch(ctx context.Context, repo RepositoryHandle, name string, track bool, baseRef string) error {
    if err := h.primary.CreateBranch(ctx, repo, name, track, baseRef); err == ErrNotImplemented { return h.fallback.CreateBranch(ctx, repo, name, track, baseRef) }
    return nil
}
func (h *HybridClient) CheckoutBranch(ctx context.Context, repo RepositoryHandle, name string, create bool, force bool) error {
    if err := h.primary.CheckoutBranch(ctx, repo, name, create, force); err == ErrNotImplemented { return h.fallback.CheckoutBranch(ctx, repo, name, create, force) }
    return nil
}


