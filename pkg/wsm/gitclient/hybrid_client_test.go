package gitclient

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

type hybridTestRepo struct {
	path string
}

func (r *hybridTestRepo) Path() string {
	return r.path
}

type hybridTestClient struct {
	addErr            error
	resetErr          error
	fetchErr          error
	pushErr           error
	createBranchErr   error
	checkoutBranchErr error
	remoteExists      bool
	remoteExistsErr   error

	addCalls            int
	resetCalls          int
	fetchCalls          int
	pushCalls           int
	createBranchCalls   int
	checkoutBranchCalls int
	remoteExistsCalls   int
}

func (f *hybridTestClient) Open(ctx context.Context, repoPath string) (RepositoryHandle, error) {
	return &hybridTestRepo{path: repoPath}, nil
}

func (f *hybridTestClient) CurrentBranch(ctx context.Context, repo RepositoryHandle) (string, error) {
	return "main", nil
}

func (f *hybridTestClient) RemoteURL(ctx context.Context, repo RepositoryHandle, remote string) (string, error) {
	return "", nil
}

func (f *hybridTestClient) LocalBranchExists(ctx context.Context, repo RepositoryHandle, branch string) (bool, error) {
	return false, nil
}

func (f *hybridTestClient) ListLocalBranches(ctx context.Context, repo RepositoryHandle) ([]string, error) {
	return []string{"main"}, nil
}

func (f *hybridTestClient) ListRemoteTrackingBranches(ctx context.Context, repo RepositoryHandle, remote string) ([]string, error) {
	return []string{}, nil
}

func (f *hybridTestClient) RemoteTrackingBranchExists(ctx context.Context, repo RepositoryHandle, remote string, branch string) (bool, error) {
	f.remoteExistsCalls++
	return f.remoteExists, f.remoteExistsErr
}

func (f *hybridTestClient) ListTags(ctx context.Context, repo RepositoryHandle) ([]string, error) {
	return []string{}, nil
}

func (f *hybridTestClient) LastCommit(ctx context.Context, repo RepositoryHandle) (string, error) {
	return "", nil
}

func (f *hybridTestClient) Status(ctx context.Context, repo RepositoryHandle) (Status, error) {
	return Status{}, nil
}

func (f *hybridTestClient) Add(ctx context.Context, repo RepositoryHandle, path string) error {
	f.addCalls++
	return f.addErr
}

func (f *hybridTestClient) Reset(ctx context.Context, repo RepositoryHandle, path string) error {
	f.resetCalls++
	return f.resetErr
}

func (f *hybridTestClient) Commit(ctx context.Context, repo RepositoryHandle, msg string, opts CommitOptions) (string, error) {
	return "", nil
}

func (f *hybridTestClient) Diff(ctx context.Context, repo RepositoryHandle, staged bool) (string, error) {
	return "", nil
}

func (f *hybridTestClient) Fetch(ctx context.Context, repo RepositoryHandle, remote string) error {
	f.fetchCalls++
	return f.fetchErr
}

func (f *hybridTestClient) Push(ctx context.Context, repo RepositoryHandle, remote string) error {
	f.pushCalls++
	return f.pushErr
}

func (f *hybridTestClient) AheadBehind(ctx context.Context, repo RepositoryHandle, upstream string) (int, int, error) {
	return 0, 0, nil
}

func (f *hybridTestClient) CreateBranch(ctx context.Context, repo RepositoryHandle, name string, track bool, baseRef string) error {
	f.createBranchCalls++
	return f.createBranchErr
}

func (f *hybridTestClient) CheckoutBranch(ctx context.Context, repo RepositoryHandle, name string, create bool, force bool) error {
	f.checkoutBranchCalls++
	return f.checkoutBranchErr
}

func TestHybridMutatingOps_PropagatePrimaryErrors(t *testing.T) {
	repo := &hybridTestRepo{path: "/tmp/repo"}
	ctx := context.Background()
	primaryErr := errors.New("primary failure")

	tests := []struct {
		name          string
		setErr        func(c *hybridTestClient, err error)
		call          func(h *HybridClient) error
		fallbackCalls func(c *hybridTestClient) int
	}{
		{
			name: "Add",
			setErr: func(c *hybridTestClient, err error) {
				c.addErr = err
			},
			call: func(h *HybridClient) error {
				return h.Add(ctx, repo, ".")
			},
			fallbackCalls: func(c *hybridTestClient) int { return c.addCalls },
		},
		{
			name: "Reset",
			setErr: func(c *hybridTestClient, err error) {
				c.resetErr = err
			},
			call: func(h *HybridClient) error {
				return h.Reset(ctx, repo, ".")
			},
			fallbackCalls: func(c *hybridTestClient) int { return c.resetCalls },
		},
		{
			name: "Fetch",
			setErr: func(c *hybridTestClient, err error) {
				c.fetchErr = err
			},
			call: func(h *HybridClient) error {
				return h.Fetch(ctx, repo, "origin")
			},
			fallbackCalls: func(c *hybridTestClient) int { return c.fetchCalls },
		},
		{
			name: "Push",
			setErr: func(c *hybridTestClient, err error) {
				c.pushErr = err
			},
			call: func(h *HybridClient) error {
				return h.Push(ctx, repo, "origin")
			},
			fallbackCalls: func(c *hybridTestClient) int { return c.pushCalls },
		},
		{
			name: "CreateBranch",
			setErr: func(c *hybridTestClient, err error) {
				c.createBranchErr = err
			},
			call: func(h *HybridClient) error {
				return h.CreateBranch(ctx, repo, "feature", false, "")
			},
			fallbackCalls: func(c *hybridTestClient) int { return c.createBranchCalls },
		},
		{
			name: "CheckoutBranch",
			setErr: func(c *hybridTestClient, err error) {
				c.checkoutBranchErr = err
			},
			call: func(h *HybridClient) error {
				return h.CheckoutBranch(ctx, repo, "feature", false, false)
			},
			fallbackCalls: func(c *hybridTestClient) int { return c.checkoutBranchCalls },
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			primary := &hybridTestClient{}
			fallback := &hybridTestClient{}
			tc.setErr(primary, primaryErr)
			h := NewHybrid(primary, fallback)

			err := tc.call(h)
			if !errors.Is(err, primaryErr) {
				t.Fatalf("expected primary error, got: %v", err)
			}
			if tc.fallbackCalls(fallback) != 0 {
				t.Fatalf("fallback should not be called for real errors, got %d calls", tc.fallbackCalls(fallback))
			}
		})
	}
}

func TestHybridMutatingOps_FallbackOnNotImplemented(t *testing.T) {
	repo := &hybridTestRepo{path: "/tmp/repo"}
	ctx := context.Background()
	wrappedNotImplemented := fmt.Errorf("wrapped: %w", ErrNotImplemented)

	tests := []struct {
		name          string
		setErr        func(c *hybridTestClient, err error)
		call          func(h *HybridClient) error
		fallbackCalls func(c *hybridTestClient) int
	}{
		{
			name: "Add",
			setErr: func(c *hybridTestClient, err error) {
				c.addErr = err
			},
			call: func(h *HybridClient) error {
				return h.Add(ctx, repo, ".")
			},
			fallbackCalls: func(c *hybridTestClient) int { return c.addCalls },
		},
		{
			name: "Reset",
			setErr: func(c *hybridTestClient, err error) {
				c.resetErr = err
			},
			call: func(h *HybridClient) error {
				return h.Reset(ctx, repo, ".")
			},
			fallbackCalls: func(c *hybridTestClient) int { return c.resetCalls },
		},
		{
			name: "Fetch",
			setErr: func(c *hybridTestClient, err error) {
				c.fetchErr = err
			},
			call: func(h *HybridClient) error {
				return h.Fetch(ctx, repo, "origin")
			},
			fallbackCalls: func(c *hybridTestClient) int { return c.fetchCalls },
		},
		{
			name: "Push",
			setErr: func(c *hybridTestClient, err error) {
				c.pushErr = err
			},
			call: func(h *HybridClient) error {
				return h.Push(ctx, repo, "origin")
			},
			fallbackCalls: func(c *hybridTestClient) int { return c.pushCalls },
		},
		{
			name: "CreateBranch",
			setErr: func(c *hybridTestClient, err error) {
				c.createBranchErr = err
			},
			call: func(h *HybridClient) error {
				return h.CreateBranch(ctx, repo, "feature", false, "")
			},
			fallbackCalls: func(c *hybridTestClient) int { return c.createBranchCalls },
		},
		{
			name: "CheckoutBranch",
			setErr: func(c *hybridTestClient, err error) {
				c.checkoutBranchErr = err
			},
			call: func(h *HybridClient) error {
				return h.CheckoutBranch(ctx, repo, "feature", false, false)
			},
			fallbackCalls: func(c *hybridTestClient) int { return c.checkoutBranchCalls },
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			primary := &hybridTestClient{}
			fallback := &hybridTestClient{}
			tc.setErr(primary, wrappedNotImplemented)
			h := NewHybrid(primary, fallback)

			err := tc.call(h)
			if err != nil {
				t.Fatalf("expected nil fallback result, got: %v", err)
			}
			if tc.fallbackCalls(fallback) != 1 {
				t.Fatalf("expected fallback to be called once, got %d calls", tc.fallbackCalls(fallback))
			}
		})
	}
}

func TestHybridRemoteTrackingBranchExists_FallbackAndPropagation(t *testing.T) {
	ctx := context.Background()
	repo := &hybridTestRepo{path: "/tmp/repo"}

	t.Run("fallback on wrapped not implemented", func(t *testing.T) {
		primary := &hybridTestClient{
			remoteExistsErr: fmt.Errorf("wrapped: %w", ErrNotImplemented),
		}
		fallback := &hybridTestClient{
			remoteExists: true,
		}
		h := NewHybrid(primary, fallback)

		ok, err := h.RemoteTrackingBranchExists(ctx, repo, "origin", "feature")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !ok {
			t.Fatalf("expected fallback value true, got false")
		}
		if fallback.remoteExistsCalls != 1 {
			t.Fatalf("expected fallback call count 1, got %d", fallback.remoteExistsCalls)
		}
	})

	t.Run("propagate real errors", func(t *testing.T) {
		primaryErr := errors.New("primary remote lookup failed")
		primary := &hybridTestClient{
			remoteExistsErr: primaryErr,
		}
		fallback := &hybridTestClient{}
		h := NewHybrid(primary, fallback)

		_, err := h.RemoteTrackingBranchExists(ctx, repo, "origin", "feature")
		if !errors.Is(err, primaryErr) {
			t.Fatalf("expected primary error, got: %v", err)
		}
		if fallback.remoteExistsCalls != 0 {
			t.Fatalf("fallback should not run for real errors")
		}
	})
}
