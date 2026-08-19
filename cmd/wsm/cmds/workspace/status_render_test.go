package workspace

import (
	"fmt"
	"testing"

	"github.com/go-go-golems/workspace-manager/pkg/wsm"
	"github.com/stretchr/testify/assert"
)

// baseStatusCase is a table-driven helper for the three rendering functions.
func baseStatusCase(status wsm.BaseComparisonStatus, resolved, source, reason string, merged, needsRebase bool) wsm.RepositoryStatus {
	return wsm.RepositoryStatus{
		CurrentBranch: "feature/x",
		IsMerged:      merged,
		NeedsRebase:   needsRebase,
		Base: wsm.BaseComparison{
			ResolvedRef: resolved,
			RefSource:   wsm.RefSource(source),
			Status:      status,
			Reason:      reason,
		},
	}
}

func TestBaseString_ResolvedWithSource(t *testing.T) {
	s := baseStatusCase(wsm.BaseResolved, "origin/main", "remote-tracking", "", false, false)
	assert.Equal(t, "origin/main (remote-tracking)", baseString(s))
}

func TestBaseString_ResolvedLocalFallback(t *testing.T) {
	s := baseStatusCase(wsm.BaseResolved, "task/deploy-dev-indexer", "local", "", false, false)
	assert.Equal(t, "task/deploy-dev-indexer (local)", baseString(s))
}

func TestBaseString_UnknownShowsReason(t *testing.T) {
	s := baseStatusCase(wsm.BaseUnknown, "", "", "task/x is not a remote-tracking ref on origin and is not a local branch", false, false)
	assert.Contains(t, baseString(s), "?")
	assert.Contains(t, baseString(s), "task/x")
}

func TestBaseString_ErrorShowsReason(t *testing.T) {
	s := baseStatusCase(wsm.BaseError, "", "", "merge-base --is-ancestor failed: exit status 128", false, false)
	assert.Contains(t, baseString(s), "!")
	assert.Contains(t, baseString(s), "merge-base")
}

func TestGetMergedString_HonestByStatus(t *testing.T) {
	assert.Equal(t, "✓", getMergedString(baseStatusCase(wsm.BaseResolved, "origin/main", "remote-tracking", "", true, false)))
	assert.Equal(t, "-", getMergedString(baseStatusCase(wsm.BaseResolved, "origin/main", "remote-tracking", "", false, false)))
	assert.Equal(t, "?", getMergedString(baseStatusCase(wsm.BaseUnknown, "", "", "no ref", false, false)), "unknown must NOT render as a confident '-'")
	assert.Equal(t, "!", getMergedString(baseStatusCase(wsm.BaseError, "", "", "git failed", false, false)))
}

func TestGetRebaseString_HonestByStatus(t *testing.T) {
	assert.Equal(t, "⚠️", getRebaseString(baseStatusCase(wsm.BaseResolved, "origin/main", "remote-tracking", "", false, true)))
	assert.Equal(t, "✓", getRebaseString(baseStatusCase(wsm.BaseResolved, "origin/main", "remote-tracking", "", false, false)))
	assert.Equal(t, "?", getRebaseString(baseStatusCase(wsm.BaseUnknown, "", "", "no ref", false, false)), "unknown must NOT render as a confident '✓'")
	assert.Equal(t, "!", getRebaseString(baseStatusCase(wsm.BaseError, "", "", "git failed", false, false)))
}

// TestStatusToRows_IncludesBaseFields asserts the JSON row carries the new
// additive base provenance fields (Q1/Q2 surface).
func TestStatusToRows_IncludesBaseFields(t *testing.T) {
	wsStatus := &wsm.WorkspaceStatus{
		Workspace: wsm.Workspace{Name: "ws"},
		Repositories: []wsm.RepositoryStatus{
			baseStatusCase(wsm.BaseResolved, "origin/develop", "remote-tracking", "", false, false),
		},
	}
	rows := statusToRows(wsStatus, false)
	assert.Len(t, rows, 1)
	row := rows[0]
	assertRow := func(key, want string) {
		v, ok := row.Get(key)
		assert.True(t, ok, "missing key %q", key)
		assert.Equal(t, want, fmt.Sprintf("%v", v), "key %q", key)
	}
	assertRow("base", "origin/develop (remote-tracking)")
	assertRow("base_ref", "origin/develop")
	assertRow("base_source", "remote-tracking")
	assertRow("base_status", "resolved")
	assertRow("base_reason", "")
	// Existing fields still present.
	assertRow("is_merged", "false")
	assertRow("needs_rebase", "false")
}
