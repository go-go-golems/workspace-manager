package branch

import (
	"context"
	"sort"

	"github.com/go-go-golems/workspace-manager/pkg/wsm/gitclient"
	"github.com/pkg/errors"
)

// RefSource describes how a base comparison ref was resolved.
type RefSource string

const (
	// RefSourceRemoteTracking means the ref is a remote-tracking branch (<remote>/<branch>).
	RefSourceRemoteTracking RefSource = "remote-tracking"
	// RefSourceLocal means the ref is a local branch (<branch>).
	RefSourceLocal RefSource = "local"
)

// BaseResolutionStatus classifies the outcome of resolving the base comparison ref.
type BaseResolutionStatus string

const (
	// BaseResolved means a concrete ref was found.
	BaseResolved BaseResolutionStatus = "resolved"
	// BaseUnknown means no usable ref exists (neither remote-tracking nor local).
	BaseUnknown BaseResolutionStatus = "unknown"
	// BaseError means git itself failed while probing for the ref.
	BaseError BaseResolutionStatus = "error"
)

// BaseRefResolution is the structured outcome of resolving which concrete git ref
// a merge/rebase status check should compare HEAD against.
//
// Preference order:
//  1. <remote>/<base> remote-tracking branch (refs/remotes/<remote>/<base>)
//  2. <base> local branch (refs/heads/<base>)
//  3. unknown (neither exists)
//
// On a genuine git failure (e.g. corrupt repository), Status is BaseError and
// Reason carries the captured stderr.
type BaseRefResolution struct {
	// Ref is the concrete git ref to compare against, e.g. "origin/main" or
	// "task/deploy-dev-indexer". Empty when Status != BaseResolved.
	Ref string
	// Source is how Ref was found (remote-tracking | local). Empty when
	// Status != BaseResolved.
	Source RefSource
	// Status classifies the outcome (resolved | unknown | error).
	Status BaseResolutionStatus
	// Reason is a human-readable explanation when Status != BaseResolved.
	Reason string
}

// ResolveBaseRef picks the concrete git ref to compare HEAD against for
// merge/rebase status, given a base branch and remote.
//
// It prefers the remote-tracking ref (reflects the shared/upstream truth) and
// falls back to the local branch (covers forked workspaces whose base branch
// was never pushed). If neither exists it returns BaseUnknown with a precise
// reason naming the missing refs; the caller must treat that as "could not
// compare", not as a confident negative.
func ResolveBaseRef(
	ctx context.Context,
	gc gitclient.GitClient,
	repoPath string,
	base BranchName,
	remote RemoteName,
) (BaseRefResolution, error) {
	res := BaseRefResolution{}

	if base == "" {
		res.Status = BaseUnknown
		res.Reason = "base branch is empty"
		return res, nil
	}

	r := remote
	if r == "" {
		r = DefaultRemoteName
	}

	h, err := gc.Open(ctx, repoPath)
	if err != nil {
		res.Status = BaseError
		res.Reason = "open repository: " + err.Error()
		return res, errors.Wrap(err, "open repository")
	}

	// 1) prefer remote-tracking ref <remote>/<base>
	remoteExists, err := gc.RemoteTrackingBranchExists(ctx, h, string(r), string(base))
	if err != nil {
		res.Status = BaseError
		res.Reason = "check remote-tracking branch: " + err.Error()
		return res, errors.Wrap(err, "check remote-tracking branch")
	}
	if remoteExists {
		res.Ref = string(r) + "/" + string(base)
		res.Source = RefSourceRemoteTracking
		res.Status = BaseResolved
		return res, nil
	}

	// 2) fall back to local branch <base>
	localExists, err := gc.LocalBranchExists(ctx, h, string(base))
	if err != nil {
		res.Status = BaseError
		res.Reason = "check local branch: " + err.Error()
		return res, errors.Wrap(err, "check local branch")
	}
	if localExists {
		res.Ref = string(base)
		res.Source = RefSourceLocal
		res.Status = BaseResolved
		return res, nil
	}

	// 3) nothing to compare against — give a precise reason
	res.Status = BaseUnknown
	res.Reason = string(base) + " is not a remote-tracking ref on " + string(r) +
		" and is not a local branch"
	return res, nil
}

// DefaultBaseBranchForRepo resolves a repository's effective default base branch
// using the git client: first the remote's advertised default (via
// `git symbolic-ref refs/remotes/<remote>/HEAD`), then a probe of common
// candidates (main, master, develop) via RemoteTrackingBranchExists. Returns
// "" if neither yields a result (caller falls back to env/main).
//
// This belongs in the branch layer so both discovery (persistence) and status
// (resolution) share one definition of "the repo's default".
func DefaultBaseBranchForRepo(
	ctx context.Context,
	gc gitclient.GitClient,
	repoPath string,
	remote RemoteName,
) (string, error) {
	r := remote
	if r == "" {
		r = DefaultRemoteName
	}
	h, err := gc.Open(ctx, repoPath)
	if err != nil {
		return "", errors.Wrap(err, "open repository")
	}

	// 1) remote-advertised default.
	if def, err := gc.DefaultBranch(ctx, h, string(r)); err == nil && def != "" {
		return def, nil
	}

	// 2) probe common candidates in order (documented heuristic).
	for _, cand := range []string{"main", "master", "develop"} {
		exists, err := gc.RemoteTrackingBranchExists(ctx, h, string(r), cand)
		if err != nil {
			return "", errors.Wrap(err, "probe candidate "+cand)
		}
		if exists {
			return cand, nil
		}
	}
	return "", nil
}

// Used by callers building divergence prompts.
func DistinctBranches(branches map[string]string) []string {
	seen := make(map[string]struct{}, len(branches))
	for _, b := range branches {
		seen[b] = struct{}{}
	}
	out := make([]string, 0, len(seen))
	for b := range seen {
		out = append(out, b)
	}
	sort.Strings(out)
	return out
}

// MostFrequentBranch returns the branch with the highest count in a repo->branch
// map, breaking ties by sorted order. Returns "" for an empty map.
func MostFrequentBranch(branches map[string]string) string {
	counts := make(map[string]int, len(branches))
	for _, b := range branches {
		counts[b]++
	}
	var best string
	bestCount := -1
	for _, b := range DistinctBranches(branches) { // deterministic order for ties
		if counts[b] > bestCount {
			best = b
			bestCount = counts[b]
		}
	}
	return best
}
