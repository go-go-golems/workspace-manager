package wsm

import (
	"context"
	"errors"
	"os/exec"
	"strconv"
	"strings"

	"github.com/go-go-golems/workspace-manager/pkg/wsm/branch"
	"github.com/go-go-golems/workspace-manager/pkg/wsm/gitclient"
	pkgerrors "github.com/pkg/errors"
)

// getGitCurrentBranch returns the current branch name.
func getGitCurrentBranch(ctx context.Context, path string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "branch", "--show-current")
	cmd.Dir = path
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// runGitCapture runs git in dir and returns stdout, capturing stderr in the
// error message on failure (unlike exec.Output, which discards stderr by
// default). Used by the status checks so a failed comparison carries a real
// reason instead of a bare exit code.
func runGitCapture(ctx context.Context, dir string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, pkgerrors.Wrapf(err, "git %s failed: %s", strings.Join(args, " "), strings.TrimSpace(string(out)))
	}
	return out, nil
}

// CheckBranchMerged checks whether HEAD has been merged into the configured
// base branch and returns a provenance-bearing BaseComparison.
//
// The base branch is first resolved by precedence (per-repo override > workspace
// base > discovered default > env > main) by the caller; here it is the
// already-resolved base branch name. The concrete comparison ref is chosen by
// branch.ResolveBaseRef: it prefers <remote>/<base> (remote-tracking) and falls
// back to the local <base>, so forked workspaces with a local-only base still
// produce a real comparison instead of a silent false.
//
// Status is BaseResolved when a real comparison ran; BaseUnknown when no usable
// ref exists (caller must treat as "could not compare", not as a negative);
// BaseError when git itself failed.
func CheckBranchMerged(
	ctx context.Context,
	gc gitclient.GitClient,
	path string,
	baseBranch string,
	remote string,
) (BaseComparison, error) {
	// Resolve the configured base through precedence (explicit > WSM_BASE_BRANCH
	// env > "main") so an empty workspace BaseBranch still compares against main.
	base := branch.ResolveBaseBranch(baseBranch)
	cmp := BaseComparison{
		ConfiguredBase: string(base),
		Remote:         remote,
	}

	res, err := branch.ResolveBaseRef(ctx, gc, path, base, branch.RemoteName(remote))
	if err != nil {
		// ResolveBaseRef already set Status/Reason on res; surface the error too.
		cmp.Status = res.Status
		cmp.Reason = res.Reason
		return cmp, err
	}
	cmp.ResolvedRef = res.Ref
	cmp.RefSource = res.Source
	cmp.Status = res.Status
	cmp.Reason = res.Reason
	if res.Status != branch.BaseResolved {
		// Unknown: no ref to compare against. Leave IsMerged false; the caller
		// must honor Status and not read IsMerged as a confident negative.
		return cmp, nil
	}

	currentBranch, branchErr := getGitCurrentBranch(ctx, path)
	if branchErr != nil {
		currentBranch = "unknown"
	}

	log.Debug().
		Str("path", path).
		Str("branch", currentBranch).
		Str("resolved_ref", cmp.ResolvedRef).
		Str("base_branch", baseBranch).
		Msg("Checking if branch is merged to resolved base")

	// merge-base --is-ancestor exits 0 iff HEAD is an ancestor of the base
	// (i.e. HEAD has been merged). Any failure here is a git error, not "no".
	if err := runGitCaptureNoOut(ctx, path, "merge-base", "--is-ancestor", "HEAD", res.Ref); err != nil {
		// exit 1 from merge-base means "not an ancestor" -> NOT merged (a real
		// negative result). Distinguish from a genuine failure (exit != 0/1).
		if isNotAncestorExit(err) {
			cmp.IsMerged = false
			return cmp, nil
		}
		cmp.Status = BaseError
		cmp.Reason = "merge-base --is-ancestor failed: " + err.Error()
		return cmp, err
	}
	cmp.IsMerged = true
	return cmp, nil
}

// CheckBranchNeedsRebase checks whether HEAD is behind the configured base
// branch and would benefit from a rebase. See CheckBranchMerged for the
// resolution semantics; the returned BaseComparison.NeedsRebase is meaningful
// only when Status == BaseResolved.
func CheckBranchNeedsRebase(
	ctx context.Context,
	gc gitclient.GitClient,
	path string,
	baseBranch string,
	remote string,
) (BaseComparison, error) {
	// Resolve the configured base through precedence (explicit > WSM_BASE_BRANCH
	// env > "main") so an empty workspace BaseBranch still compares against main.
	base := branch.ResolveBaseBranch(baseBranch)
	cmp := BaseComparison{
		ConfiguredBase: string(base),
		Remote:         remote,
	}

	res, err := branch.ResolveBaseRef(ctx, gc, path, base, branch.RemoteName(remote))
	if err != nil {
		cmp.Status = res.Status
		cmp.Reason = res.Reason
		return cmp, err
	}
	cmp.ResolvedRef = res.Ref
	cmp.RefSource = res.Source
	cmp.Status = res.Status
	cmp.Reason = res.Reason
	if res.Status != branch.BaseResolved {
		return cmp, nil
	}

	currentBranch, branchErr := getGitCurrentBranch(ctx, path)
	if branchErr != nil {
		currentBranch = "unknown"
	}

	// Skip rebase check if we're already on the configured base branch.
	if currentBranch == string(base) || (string(base) == "main" && currentBranch == "master") {
		log.Debug().
			Str("path", path).
			Str("branch", currentBranch).
			Str("base_branch", baseBranch).
			Msg("Skipping rebase check - already on base branch")
		cmp.NeedsRebase = false
		return cmp, nil
	}

	log.Debug().
		Str("path", path).
		Str("branch", currentBranch).
		Str("resolved_ref", cmp.ResolvedRef).
		Str("base_branch", baseBranch).
		Msg("Checking if branch needs rebase on resolved base")

	out, err := runGitCapture(ctx, path, "rev-list", "--count", "HEAD.."+res.Ref)
	if err != nil {
		cmp.Status = BaseError
		cmp.Reason = "rev-list --count failed: " + err.Error()
		return cmp, err
	}

	countStr := strings.TrimSpace(string(out))
	count, parseErr := strconv.Atoi(countStr)
	if parseErr != nil {
		cmp.Status = BaseError
		cmp.Reason = "unparseable rev-list count " + countStr + ": " + parseErr.Error()
		return cmp, parseErr
	}
	cmp.NeedsRebase = count > 0

	log.Debug().
		Str("path", path).
		Str("branch", currentBranch).
		Str("resolved_ref", cmp.ResolvedRef).
		Str("commits_behind", countStr).
		Bool("needs_rebase", cmp.NeedsRebase).
		Msg("Branch rebase check result")

	return cmp, nil
}

// runGitCaptureNoOut runs git and returns only an error (stdout discarded).
// Used for `merge-base --is-ancestor`, which signals via exit code.
func runGitCaptureNoOut(ctx context.Context, dir string, args ...string) error {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return pkgerrors.Wrapf(err, "git %s failed: %s", strings.Join(args, " "), strings.TrimSpace(string(out)))
	}
	return nil
}

// isNotAncestorExit reports whether an error from `merge-base --is-ancestor`
// represents the "not an ancestor" result (exit status 1) rather than a genuine
// git failure. merge-base returns 0 (ancestor) or 1 (not an ancestor); any
// other status is a real error.
func isNotAncestorExit(err error) bool {
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return ee.ExitCode() == 1
	}
	return false
}
