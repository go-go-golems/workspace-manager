package gitclient

import (
	"bytes"
	"context"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

type cliRepo struct{ path string }

func (r *cliRepo) Path() string { return r.path }

// CliGitClient implements GitClient using the system git CLI.
type CliGitClient struct{}

func NewCli() *CliGitClient { return &CliGitClient{} }

func (c *CliGitClient) Open(ctx context.Context, repoPath string) (RepositoryHandle, error) {
	// Validate that repoPath is a git repo by checking .git (best-effort)
	// We do not fail if missing, deferring to commands.
	abs, _ := filepath.Abs(repoPath)
	return &cliRepo{path: abs}, nil
}

func runGit(ctx context.Context, dir string, args ...string) ([]byte, error) {
	// #nosec G204 -- git is invoked with a literal binary and program-derived args; no shell, no user-tainted input.
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, errors.Wrapf(err, "git %s failed: %s", strings.Join(args, " "), string(out))
	}
	return out, nil
}

func (c *CliGitClient) CurrentBranch(ctx context.Context, repo RepositoryHandle) (string, error) {
	out, err := runGit(ctx, repo.Path(), "branch", "--show-current")
	return strings.TrimSpace(string(out)), err
}

func (c *CliGitClient) RemoteURL(ctx context.Context, repo RepositoryHandle, remote string) (string, error) {
	if remote == "" {
		remote = "origin"
	}
	out, err := runGit(ctx, repo.Path(), "remote", "get-url", remote)
	return strings.TrimSpace(string(out)), err
}

func (c *CliGitClient) LocalBranchExists(ctx context.Context, repo RepositoryHandle, branch string) (bool, error) {
	if branch == "" {
		return false, nil
	}
	ref := "refs/heads/" + branch
	out, err := runGit(ctx, repo.Path(), "for-each-ref", "--format=%(refname)", ref)
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(string(out)) != "", nil
}

func (c *CliGitClient) ListLocalBranches(ctx context.Context, repo RepositoryHandle) ([]string, error) {
	out, err := runGit(ctx, repo.Path(), "for-each-ref", "--format=%(refname:short)", "refs/heads")
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var branches []string
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l == "" {
			continue
		}
		branches = append(branches, l)
	}
	return branches, nil
}

func (c *CliGitClient) ListRemoteTrackingBranches(ctx context.Context, repo RepositoryHandle, remote string) ([]string, error) {
	if remote == "" {
		remote = "origin"
	}
	out, err := runGit(ctx, repo.Path(), "for-each-ref", "--format=%(refname:short)", "refs/remotes/"+remote)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var branches []string
	prefix := remote + "/"
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l == "" || l == remote+"/HEAD" {
			continue
		}
		l = strings.TrimPrefix(l, prefix)
		if l == "" || l == "HEAD" {
			continue
		}
		branches = append(branches, l)
	}
	return branches, nil
}

func (c *CliGitClient) RemoteTrackingBranchExists(ctx context.Context, repo RepositoryHandle, remote string, branch string) (bool, error) {
	if remote == "" {
		remote = "origin"
	}
	if branch == "" {
		return false, nil
	}
	ref := "refs/remotes/" + remote + "/" + branch
	out, err := runGit(ctx, repo.Path(), "for-each-ref", "--format=%(refname)", ref)
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(string(out)) != "", nil
}

// DefaultBranch returns the remote's advertised default branch name via
// `git symbolic-ref refs/remotes/<remote>/HEAD`, stripped of the
// "refs/remotes/<remote>/" prefix (e.g. "develop", "main"). An unset HEAD is
// not an error: it returns ("", nil) so callers can fall back to probing
// candidates (main/master/develop).
func (c *CliGitClient) DefaultBranch(ctx context.Context, repo RepositoryHandle, remote string) (string, error) {
	if remote == "" {
		remote = "origin"
	}
	out, err := runGit(ctx, repo.Path(), "symbolic-ref", "refs/remotes/"+remote+"/HEAD")
	if err != nil {
		// Unset origin/HEAD -> not an error; caller probes candidates.
		return "", nil
	}
	ref := strings.TrimSpace(string(out)) // e.g. refs/remotes/origin/develop
	prefix := "refs/remotes/" + remote + "/"
	return strings.TrimPrefix(ref, prefix), nil // "develop"
}

func (c *CliGitClient) ListTags(ctx context.Context, repo RepositoryHandle) ([]string, error) {
	out, err := runGit(ctx, repo.Path(), "tag", "-l")
	if err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return []string{}, nil
	}
	return strings.Split(strings.TrimSpace(string(out)), "\n"), nil
}

func (c *CliGitClient) LastCommit(ctx context.Context, repo RepositoryHandle) (string, error) {
	out, err := runGit(ctx, repo.Path(), "log", "-1", "--pretty=format:%H %s")
	return strings.TrimSpace(string(out)), err
}

func (c *CliGitClient) Status(ctx context.Context, repo RepositoryHandle) (Status, error) {
	out, err := runGit(ctx, repo.Path(), "status", "--porcelain", "-z")
	if err != nil {
		return Status{}, err
	}
	var st Status
	st.CurrentBranch, _ = c.CurrentBranch(ctx, repo)
	if len(out) == 0 {
		return st, nil
	}

	records := bytes.Split(out, []byte{0})
	for i := 0; i < len(records); i++ {
		rec := records[i]
		if len(rec) == 0 || len(rec) < 3 {
			continue
		}

		idx := rec[0]
		wt := rec[1]
		pathStart := 2
		if len(rec) > 2 && rec[2] == ' ' {
			pathStart = 3
		}
		if len(rec) <= pathStart {
			continue
		}
		path := string(rec[pathStart:])

		if idx != ' ' && idx != '?' {
			st.StagedFiles = append(st.StagedFiles, path)
		}
		if wt != ' ' && wt != '?' {
			st.ModifiedFiles = append(st.ModifiedFiles, path)
		}
		if idx == '?' && wt == '?' {
			st.UntrackedFiles = append(st.UntrackedFiles, path)
		}

		// In porcelain -z format, rename/copy entries include an extra NUL-separated path record.
		// We keep the first path as the working path and skip the paired record.
		if idx == 'R' || idx == 'C' || wt == 'R' || wt == 'C' {
			if i+1 < len(records) {
				i++
			}
		}
	}
	return st, nil
}

func (c *CliGitClient) Add(ctx context.Context, repo RepositoryHandle, path string) error {
	_, err := runGit(ctx, repo.Path(), "add", path)
	return err
}

func (c *CliGitClient) Reset(ctx context.Context, repo RepositoryHandle, path string) error {
	_, err := runGit(ctx, repo.Path(), "reset", "HEAD", path)
	return err
}

func (c *CliGitClient) Commit(ctx context.Context, repo RepositoryHandle, msg string) error {
	_, err := runGit(ctx, repo.Path(), "commit", "-m", msg)
	return err
}

func (c *CliGitClient) Diff(ctx context.Context, repo RepositoryHandle, staged bool) (string, error) {
	args := []string{"diff"}
	if staged {
		args = append(args, "--cached")
	}
	out, err := runGit(ctx, repo.Path(), args...)
	return string(out), err
}

func (c *CliGitClient) Fetch(ctx context.Context, repo RepositoryHandle, remote string) error {
	if remote == "" {
		remote = "origin"
	}
	_, err := runGit(ctx, repo.Path(), "fetch", remote)
	return err
}

func (c *CliGitClient) Push(ctx context.Context, repo RepositoryHandle, remote string) error {
	if remote == "" {
		remote = "origin"
	}
	_, err := runGit(ctx, repo.Path(), "push", remote)
	if err == nil {
		return nil
	}

	// First push of a freshly created branch can fail under push.default=simple when no upstream exists.
	// In that case, retry with explicit upstream setup.
	lowerErr := strings.ToLower(err.Error())
	if strings.Contains(lowerErr, "has no upstream branch") ||
		strings.Contains(lowerErr, "no upstream branch") {
		branch, branchErr := c.CurrentBranch(ctx, repo)
		if branchErr != nil || branch == "" {
			return err
		}
		_, upErr := runGit(ctx, repo.Path(), "push", "--set-upstream", remote, branch)
		if upErr == nil {
			return nil
		}
		return upErr
	}

	return err
}

func (c *CliGitClient) AheadBehind(ctx context.Context, repo RepositoryHandle, upstream string) (int, int, error) {
	if upstream == "" {
		// Use HEAD...@{upstream} if configured
		if _, err := runGit(ctx, repo.Path(), "rev-parse", "--abbrev-ref", "@{upstream}"); err != nil {
			return 0, 0, nil
		}
		out, err := runGit(ctx, repo.Path(), "rev-list", "--left-right", "--count", "HEAD...@{upstream}")
		if err != nil {
			return 0, 0, err
		}
		fields := strings.Fields(strings.TrimSpace(string(out)))
		if len(fields) != 2 {
			return 0, 0, errors.Errorf("unexpected rev-list output: %s", string(out))
		}
		ahead, _ := strconv.Atoi(fields[0])
		behind, _ := strconv.Atoi(fields[1])
		return ahead, behind, nil
	}
	out, err := runGit(ctx, repo.Path(), "rev-list", "--left-right", "--count", "HEAD..."+upstream)
	if err != nil {
		return 0, 0, err
	}
	fields := strings.Fields(strings.TrimSpace(string(out)))
	if len(fields) != 2 {
		return 0, 0, errors.Errorf("unexpected rev-list output: %s", string(out))
	}
	ahead, _ := strconv.Atoi(fields[0])
	behind, _ := strconv.Atoi(fields[1])
	return ahead, behind, nil
}

func (c *CliGitClient) CreateBranch(ctx context.Context, repo RepositoryHandle, name string, track bool, baseRef string) error {
	args := []string{"checkout", "-b", name}
	if track {
		args = append(args, "--track")
	}
	if baseRef != "" {
		args = append(args, baseRef)
	}
	_, err := runGit(ctx, repo.Path(), args...)
	return err
}

func (c *CliGitClient) CheckoutBranch(ctx context.Context, repo RepositoryHandle, name string, create bool, force bool) error {
	if create {
		return c.CreateBranch(ctx, repo, name, false, "")
	}
	args := []string{"checkout"}
	if force {
		args = append(args, "-f")
	}
	args = append(args, name)
	_, err := runGit(ctx, repo.Path(), args...)
	return err
}
