package gitclient

import (
	"context"
	stderrors "errors"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/pkg/errors"
)

// sentinel for hybrid fallback
var ErrNotImplemented = errors.New("not implemented in go-git backend")

type gogitRepo struct {
	path string
	repo *git.Repository
}

func (r *gogitRepo) Path() string { return r.path }

type GoGitClient struct{}

func NewGoGit() *GoGitClient { return &GoGitClient{} }

func (c *GoGitClient) Open(ctx context.Context, repoPath string) (RepositoryHandle, error) {
	abs, _ := filepath.Abs(repoPath)
	repo, err := git.PlainOpen(abs)
	if err != nil {
		return nil, errors.Wrap(err, "open repo")
	}
	return &gogitRepo{path: abs, repo: repo}, nil
}

func (c *GoGitClient) CurrentBranch(ctx context.Context, repo RepositoryHandle) (string, error) {
	gr := repo.(*gogitRepo)
	head, err := gr.repo.Head()
	if err != nil {
		return "", errors.Wrap(err, "head")
	}
	if head.Name().IsBranch() {
		return head.Name().Short(), nil
	}
	return head.Hash().String(), nil
}

func (c *GoGitClient) RemoteURL(ctx context.Context, repo RepositoryHandle, remote string) (string, error) {
	if remote == "" {
		remote = "origin"
	}
	gr := repo.(*gogitRepo)
	r, err := gr.repo.Remote(remote)
	if err != nil {
		return "", errors.Wrap(err, "remote")
	}
	urls := r.Config().URLs
	if len(urls) == 0 {
		return "", nil
	}
	return urls[0], nil
}

func (c *GoGitClient) LocalBranchExists(ctx context.Context, repo RepositoryHandle, branch string) (bool, error) {
	if branch == "" {
		return false, nil
	}
	gr := repo.(*gogitRepo)
	refName := plumbing.NewBranchReferenceName(branch)
	_, err := gr.repo.Reference(refName, true)
	if err == nil {
		return true, nil
	}
	if stderrors.Is(err, plumbing.ErrReferenceNotFound) {
		return false, nil
	}
	return false, errors.Wrap(err, "local branch reference")
}

func (c *GoGitClient) ListLocalBranches(ctx context.Context, repo RepositoryHandle) ([]string, error) {
	gr := repo.(*gogitRepo)
	it, err := gr.repo.Branches()
	if err != nil {
		return nil, errors.Wrap(err, "branches")
	}
	var out []string
	_ = it.ForEach(func(ref *plumbing.Reference) error {
		out = append(out, ref.Name().Short())
		return nil
	})
	return out, nil
}

func (c *GoGitClient) ListRemoteTrackingBranches(ctx context.Context, repo RepositoryHandle, remote string) ([]string, error) {
	if remote == "" {
		remote = "origin"
	}
	gr := repo.(*gogitRepo)
	it, err := gr.repo.References()
	if err != nil {
		return nil, errors.Wrap(err, "references")
	}
	var out []string
	prefix := "refs/remotes/" + remote + "/"
	_ = it.ForEach(func(ref *plumbing.Reference) error {
		name := ref.Name().String()
		if !strings.HasPrefix(name, prefix) {
			return nil
		}
		short := strings.TrimPrefix(name, prefix)
		if short == "" || short == "HEAD" {
			return nil
		}
		out = append(out, short)
		return nil
	})
	return out, nil
}

func (c *GoGitClient) RemoteTrackingBranchExists(ctx context.Context, repo RepositoryHandle, remote string, branch string) (bool, error) {
	if remote == "" {
		remote = "origin"
	}
	if branch == "" {
		return false, nil
	}
	gr := repo.(*gogitRepo)
	refName := plumbing.ReferenceName("refs/remotes/" + remote + "/" + branch)
	_, err := gr.repo.Reference(refName, true)
	if err == nil {
		return true, nil
	}
	if stderrors.Is(err, plumbing.ErrReferenceNotFound) {
		return false, nil
	}
	return false, errors.Wrap(err, "remote branch reference")
}

func (c *GoGitClient) ListTags(ctx context.Context, repo RepositoryHandle) ([]string, error) {
	gr := repo.(*gogitRepo)
	it, err := gr.repo.Tags()
	if err != nil {
		return nil, errors.Wrap(err, "tags")
	}
	var out []string
	_ = it.ForEach(func(ref *plumbing.Reference) error {
		out = append(out, ref.Name().Short())
		return nil
	})
	return out, nil
}

func (c *GoGitClient) LastCommit(ctx context.Context, repo RepositoryHandle) (string, error) {
	gr := repo.(*gogitRepo)
	head, err := gr.repo.Head()
	if err != nil {
		return "", errors.Wrap(err, "head")
	}
	commit, err := gr.repo.CommitObject(head.Hash())
	if err != nil {
		return head.Hash().String(), nil
	}
	return commit.Hash.String() + " " + commit.Message, nil
}

func (c *GoGitClient) Status(ctx context.Context, repo RepositoryHandle) (Status, error) {
	gr := repo.(*gogitRepo)
	wt, err := gr.repo.Worktree()
	if err != nil {
		return Status{}, errors.Wrap(err, "worktree")
	}
	s, err := wt.Status()
	if err != nil {
		return Status{}, errors.Wrap(err, "status")
	}
	branch, _ := c.CurrentBranch(ctx, repo)
	st := Status{CurrentBranch: branch}
	for path, fs := range s {
		// staged changes are in fs.Staging, unstaged in fs.Worktree
		if fs.Staging != git.Unmodified {
			st.StagedFiles = append(st.StagedFiles, path)
		}
		if fs.Worktree != git.Unmodified {
			// Added by worktree status includes untracked
			if fs.Worktree == git.Untracked {
				st.UntrackedFiles = append(st.UntrackedFiles, path)
			} else {
				st.ModifiedFiles = append(st.ModifiedFiles, path)
			}
		}
	}
	return st, nil
}

func (c *GoGitClient) Add(ctx context.Context, repo RepositoryHandle, path string) error {
	gr := repo.(*gogitRepo)
	wt, err := gr.repo.Worktree()
	if err != nil {
		return errors.Wrap(err, "worktree")
	}
	_, e := wt.Add(path)
	return errors.Wrap(e, "add")
}

func (c *GoGitClient) Reset(ctx context.Context, repo RepositoryHandle, path string) error {
	// Path-specific mixed reset is not available in go-git; use fallback.
	return ErrNotImplemented
}

func (c *GoGitClient) Commit(ctx context.Context, repo RepositoryHandle, msg string, opts CommitOptions) (string, error) {
	gr := repo.(*gogitRepo)
	wt, err := gr.repo.Worktree()
	if err != nil {
		return "", errors.Wrap(err, "worktree")
	}
	hash, err := wt.Commit(msg, &git.CommitOptions{Author: &object.Signature{Name: opts.AuthorName, Email: opts.AuthorEmail}})
	if err != nil {
		return "", errors.Wrap(err, "commit")
	}
	return hash.String(), nil
}

func (c *GoGitClient) Diff(ctx context.Context, repo RepositoryHandle, staged bool) (string, error) {
	// Generating unified diffs is non-trivial in go-git; fall back
	return "", ErrNotImplemented
}

func (c *GoGitClient) Fetch(ctx context.Context, repo RepositoryHandle, remote string) error {
	if remote == "" {
		remote = "origin"
	}
	gr := repo.(*gogitRepo)
	return errors.Wrap(gr.repo.FetchContext(ctx, &git.FetchOptions{RemoteName: remote, Tags: git.AllTags}), "fetch")
}

func (c *GoGitClient) Push(ctx context.Context, repo RepositoryHandle, remote string) error {
	if remote == "" {
		remote = "origin"
	}
	gr := repo.(*gogitRepo)
	return errors.Wrap(gr.repo.PushContext(ctx, &git.PushOptions{RemoteName: remote}), "push")
}

func (c *GoGitClient) AheadBehind(ctx context.Context, repo RepositoryHandle, upstream string) (int, int, error) {
	// TODO: implement via commit graph walk comparing HEAD and upstream
	return 0, 0, ErrNotImplemented
}

func (c *GoGitClient) CreateBranch(ctx context.Context, repo RepositoryHandle, name string, track bool, baseRef string) error {
	gr := repo.(*gogitRepo)
	wt, err := gr.repo.Worktree()
	if err != nil {
		return errors.Wrap(err, "worktree")
	}
	// checkout -b name [base]
	opts := &git.CheckoutOptions{Branch: plumbing.NewBranchReferenceName(name), Create: true}
	if baseRef != "" {
		hash, err := resolveRevisionHash(gr.repo, baseRef)
		if err != nil {
			return errors.Wrapf(err, "resolve base ref %q", baseRef)
		}
		opts.Hash = hash
	}
	return errors.Wrap(wt.Checkout(opts), "checkout -b")
}

func resolveRevisionHash(repo *git.Repository, baseRef string) (plumbing.Hash, error) {
	candidates := []plumbing.Revision{
		plumbing.Revision(baseRef),
		plumbing.Revision("refs/heads/" + baseRef),
		plumbing.Revision("refs/remotes/" + baseRef),
		plumbing.Revision("refs/tags/" + baseRef),
	}
	for _, rev := range candidates {
		h, err := repo.ResolveRevision(rev)
		if err == nil && h != nil {
			return *h, nil
		}
	}
	return plumbing.ZeroHash, plumbing.ErrReferenceNotFound
}

func (c *GoGitClient) CheckoutBranch(ctx context.Context, repo RepositoryHandle, name string, create bool, force bool) error {
	gr := repo.(*gogitRepo)
	wt, err := gr.repo.Worktree()
	if err != nil {
		return errors.Wrap(err, "worktree")
	}
	opts := &git.CheckoutOptions{Branch: plumbing.NewBranchReferenceName(name), Create: create, Force: force}
	return errors.Wrap(wt.Checkout(opts), "checkout")
}
