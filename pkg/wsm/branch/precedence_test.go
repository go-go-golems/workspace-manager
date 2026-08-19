package branch

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveBaseBranchForRepo_WorkspaceOverrideWins(t *testing.T) {
	in := RepoBaseInput{
		BaseBranchWorkspace: "task/local",
		BaseBranchGlobal:    "task/global",
		WorkspaceBase:       "main",
		DefaultBaseBranch:   "develop",
	}
	b, r := ResolveBaseBranchForRepo(in)
	assert.Equal(t, BranchName("task/local"), b)
	assert.Equal(t, DefaultRemoteName, r)
}

func TestResolveBaseBranchForRepo_GlobalWhenNoWorkspace(t *testing.T) {
	in := RepoBaseInput{
		BaseBranchGlobal:  "task/global",
		WorkspaceBase:     "main",
		DefaultBaseBranch: "develop",
	}
	b, _ := ResolveBaseBranchForRepo(in)
	assert.Equal(t, BranchName("task/global"), b)
}

func TestResolveBaseBranchForRepo_WorkspaceBaseWhenNoOverride(t *testing.T) {
	in := RepoBaseInput{
		WorkspaceBase:     "task/deploy-dev-indexer",
		DefaultBaseBranch: "develop",
	}
	b, _ := ResolveBaseBranchForRepo(in)
	assert.Equal(t, BranchName("task/deploy-dev-indexer"), b)
}

func TestResolveBaseBranchForRepo_DiscoveredDefaultWhenNoWorkspaceBase(t *testing.T) {
	in := RepoBaseInput{
		DefaultBaseBranch: "develop",
	}
	b, _ := ResolveBaseBranchForRepo(in)
	assert.Equal(t, BranchName("develop"), b, "discovered default beats env/main")
}

func TestResolveBaseBranchForRepo_EnvWhenNoDiscovered(t *testing.T) {
	t.Setenv("WSM_BASE_BRANCH", "task/from-env")
	in := RepoBaseInput{}
	b, _ := ResolveBaseBranchForRepo(in)
	assert.Equal(t, BranchName("task/from-env"), b)
}

func TestResolveBaseBranchForRepo_MainFallback(t *testing.T) {
	// Ensure no env override leaks in from the environment.
	t.Setenv("WSM_BASE_BRANCH", "")
	in := RepoBaseInput{}
	b, r := ResolveBaseBranchForRepo(in)
	assert.Equal(t, DefaultBaseBranch, b)
	assert.Equal(t, DefaultRemoteName, r)
}

func TestResolveBaseBranchForRepo_OverrideRemoteUsed(t *testing.T) {
	in := RepoBaseInput{
		BaseBranchWorkspace: "task/local",
		BaseRemoteWorkspace: "upstream",
	}
	_, r := ResolveBaseBranchForRepo(in)
	assert.Equal(t, RemoteName("upstream"), r, "override BaseRemote should win, not default origin")
}

func TestResolveBaseBranchForRepo_OverrideRemoteDefaultsToOriginWhenEmpty(t *testing.T) {
	in := RepoBaseInput{
		BaseBranchWorkspace: "task/local",
		BaseRemoteWorkspace: "", // empty -> origin
	}
	_, r := ResolveBaseBranchForRepo(in)
	assert.Equal(t, DefaultRemoteName, r)
}

func TestResolveBaseBranchForRepo_EnvSetButDiscoveredPresent_DiscoversWins(t *testing.T) {
	// Precedence: discovered default (4) beats env (5).
	t.Setenv("WSM_BASE_BRANCH", "task/env")
	in := RepoBaseInput{DefaultBaseBranch: "develop"}
	b, _ := ResolveBaseBranchForRepo(in)
	assert.Equal(t, BranchName("develop"), b)
}

// Ensure the test process does not inherit a WSM_BASE_BRANCH that would make
// MainFallback flaky.
func TestMain(m *testing.M) {
	_ = os.Unsetenv("WSM_BASE_BRANCH")
	os.Exit(m.Run())
}
