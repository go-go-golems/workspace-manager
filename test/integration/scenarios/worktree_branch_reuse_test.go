package scenarios

import (
	"path/filepath"
	"strings"
	"testing"

	h "github.com/go-go-golems/workspace-manager/test/integration/helpers"
)

func TestWorktreeBranchReuse_CreateAndAddOnMain(t *testing.T) {
	s := h.NewSandbox(t)
	defer s.Cleanup()

	remote := s.InitBareRepo(t, "remote")
	repo1 := s.InitRepo(t, "repo1", remote)
	repo2 := s.InitRepo(t, "repo2", remote)

	// Prepare an existing local branch that is not currently checked out.
	h.RunForTest(t, s, repo1, nil, "bash", "-lc", "git checkout -b feature/existing && git checkout main")
	h.RunForTest(t, s, repo2, nil, "bash", "-lc", "git checkout -b feature/existing && git checkout main")

	res := s.RunWSM(t, nil, s.ReposDir, "discover", s.ReposDir)
	if res.ExitCode != 0 {
		t.Fatalf("discover failed: %s\n%s", res.Stdout, res.Stderr)
	}

	wsName := "ws-branch-reuse"
	res = s.RunWSM(t, nil, "", "create", wsName, "--repos", "repo1", "--branch", "feature/existing")
	if res.ExitCode != 0 {
		t.Fatalf("create on existing local branch failed: %s\n%s", res.Stdout, res.Stderr)
	}
	wsPath := s.LoadWorkspacePath(t, wsName)

	res = s.RunWSM(t, nil, wsPath, "add", wsName, "repo2", "--branch", "feature/existing")
	if res.ExitCode != 0 {
		t.Fatalf("add on existing local branch failed: %s\n%s", res.Stdout, res.Stderr)
	}

	repo2Path := filepath.Join(wsPath, "repo2")
	currentBranch := strings.TrimSpace(h.RunForTest(t, s, repo2Path, nil, "git", "rev-parse", "--abbrev-ref", "HEAD"))
	if currentBranch != "feature/existing" {
		t.Fatalf("expected repo2 worktree on feature/existing, got %q", currentBranch)
	}
}

func TestWorktreeAdd_CreatesMissingBranchFromWorkspaceBase(t *testing.T) {
	s := h.NewSandbox(t)
	defer s.Cleanup()

	remote := s.InitBareRepo(t, "remote")
	_ = s.InitRepo(t, "repo1", remote)
	_ = s.InitRepo(t, "repo2", remote)

	res := s.RunWSM(t, nil, s.ReposDir, "discover", s.ReposDir)
	if res.ExitCode != 0 {
		t.Fatalf("discover failed: %s\n%s", res.Stdout, res.Stderr)
	}

	wsName := "ws-add-missing-branch"
	targetBranch := "task/add-os-doc-browser"

	res = s.RunWSM(t, nil, "", "create", wsName, "--repos", "repo1", "--branch", targetBranch)
	if res.ExitCode != 0 {
		t.Fatalf("create on missing target branch failed: %s\n%s", res.Stdout, res.Stderr)
	}
	wsPath := s.LoadWorkspacePath(t, wsName)

	// repo2 does not have targetBranch locally or on origin; add should create it from workspace base branch.
	res = s.RunWSM(t, nil, wsPath, "add", wsName, "repo2")
	if res.ExitCode != 0 {
		t.Fatalf("add on missing target branch failed: %s\n%s", res.Stdout, res.Stderr)
	}

	repo2Path := filepath.Join(wsPath, "repo2")
	currentBranch := strings.TrimSpace(h.RunForTest(t, s, repo2Path, nil, "git", "rev-parse", "--abbrev-ref", "HEAD"))
	if currentBranch != targetBranch {
		t.Fatalf("expected repo2 worktree on %q, got %q", targetBranch, currentBranch)
	}
}
