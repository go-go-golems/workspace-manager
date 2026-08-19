package workflows

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrBranchDivergence_DistinctBranches(t *testing.T) {
	div := &ErrBranchDivergence{
		Branches: map[string]string{
			"coinvault": "task/deploy-dev-indexer",
			"geppetto":  "task/deploy-dev-indexer",
			"goldeneag": "task/deploy-image",
		},
		Expected: "task/deploy-dev-indexer",
		Source:   "deploy-dev-indexer",
	}
	assert.Equal(t, []string{"task/deploy-dev-indexer", "task/deploy-image"}, div.DistinctBranches())
}

func TestErrBranchDivergence_ErrorMentionsBranchesAndSource(t *testing.T) {
	div := &ErrBranchDivergence{
		Branches: map[string]string{"a": "main", "b": "develop"},
		Source:   "src-ws",
	}
	msg := div.Error()
	assert.Contains(t, msg, "src-ws")
	assert.Contains(t, msg, "develop")
	assert.Contains(t, msg, "main")
	assert.Contains(t, msg, "--base-branch")
}

func TestErrBranchDivergence_SingleBranchIsNotDivergent(t *testing.T) {
	// DistinctBranches on a uniform map still works (one entry); Plan only
	// constructs ErrBranchDivergence when len(distinct) > 1, but the helper
	// itself must not misbehave on a single branch.
	div := &ErrBranchDivergence{Branches: map[string]string{"a": "main", "b": "main"}}
	assert.Equal(t, []string{"main"}, div.DistinctBranches())
}

// TestForkRequest_BaseBranchField ensures the explicit base field is part of
// the request struct and defaults to empty (no override). This guards the F1
// contract at the type level.
func TestForkRequest_BaseBranchField(t *testing.T) {
	req := ForkRequest{NewWorkspaceName: "x"}
	assert.Empty(t, req.BaseBranch, "BaseBranch defaults to empty (no override)")
	req2 := ForkRequest{NewWorkspaceName: "x", BaseBranch: "task/deploy-dev-indexer"}
	assert.Equal(t, "task/deploy-dev-indexer", req2.BaseBranch)
}

// TestDistinctBranches_DeterministicOrder is a sanity check that the helper
// returns a stable, sorted order (used by prompt ordering and error messages).
func TestDistinctBranches_DeterministicOrder(t *testing.T) {
	got := (&ErrBranchDivergence{Branches: map[string]string{
		"z": "b-branch", "a": "a-branch", "m": "a-branch",
	}}).DistinctBranches()
	want := []string{"a-branch", "b-branch"}
	assert.True(t, sort.StringsAreSorted(got), "DistinctBranches must be sorted: got %v", got)
	assert.Equal(t, want, got)
}
