package scenarios

import (
	"strings"
	"testing"

	h "github.com/go-go-golems/workspace-manager/test/integration/helpers"
)

// TestForkDivergence_BaseBranchFlagSucceeds reproduces the user-reported error
// (wsm fork hard-failing when source repos are on different branches) and
// verifies the F1/F2 fix: fork succeeds when --base-branch is passed, and
// fails with a helpful error naming the branches when it is not (in
// non-interactive/glaze mode).
func TestForkDivergence_BaseBranchFlagSucceeds(t *testing.T) {
	s := h.NewSandbox(t)
	defer s.Cleanup()

	remote := s.InitBareRepo(t, "remote")
	s.InitRepo(t, "repo1", remote)
	s.InitRepo(t, "repo2", remote)

	res := s.RunWSM(t, nil, s.ReposDir, "discover", s.ReposDir)
	if res.ExitCode != 0 {
		t.Fatalf("discover failed: %s\n%s", res.Stdout, res.Stderr)
	}

	// Create a source workspace with both repos on task/base.
	wsName := "ws-fork-src"
	res = s.RunWSM(t, nil, "", "create", wsName, "--repos", "repo1,repo2", "--branch", "task/base")
	if res.ExitCode != 0 {
		t.Fatalf("create source failed: %s\n%s", res.Stdout, res.Stderr)
	}

	// Diverge: move repo2's worktree onto a different branch.
	wsPath := s.LoadWorkspacePath(t, wsName)
	repo2InWorkspace := wsPath + "/repo2"
	// Create the divergent branch off the current branch and switch to it.
	h.RunForTest(t, s, repo2InWorkspace, nil, "bash", "-lc",
		"git checkout -b task/divergent && echo x >> diverge.txt && git add diverge.txt && git commit -m diverge")

	// Sanity: repo1 is on task/base, repo2 is on task/divergent.

	// Fork WITHOUT --base-branch in glaze mode: should fail with a divergence
	// error that names both branches and mentions --base-branch.
	res = s.RunWSM(t, nil, wsPath, "fork", "ws-fork-dst", wsName, "--with-glaze-output", "--output", "json")
	if res.ExitCode == 0 {
		t.Fatalf("expected fork to fail without --base-branch on divergent source, but it succeeded:\n%s", res.Stdout)
	}
	combined := res.Stdout + "\n" + res.Stderr
	if !strings.Contains(combined, "--base-branch") {
		t.Fatalf("expected divergence error to mention --base-branch, got:\n%s", combined)
	}
	if !strings.Contains(combined, "task/base") || !strings.Contains(combined, "task/divergent") {
		t.Fatalf("expected divergence error to name both branches, got:\n%s", combined)
	}

	// Fork WITH --base-branch task/base: should succeed despite the divergence.
	res = s.RunWSM(t, nil, wsPath, "fork", "ws-fork-dst", wsName, "--base-branch", "task/base")
	if res.ExitCode != 0 {
		t.Fatalf("expected fork with --base-branch to succeed on divergent source, got:\n%s\n%s", res.Stdout, res.Stderr)
	}

	// Confirm the forked workspace exists and was created from the chosen base.
	dstPath := s.LoadWorkspacePath(t, "ws-fork-dst")
	if dstPath == "" {
		t.Fatalf("forked workspace ws-fork-dst not found")
	}
	// Both repos should exist in the new workspace.
	h.RunForTest(t, s, dstPath+"/repo1", nil, "git", "rev-parse", "HEAD")
	h.RunForTest(t, s, dstPath+"/repo2", nil, "git", "rev-parse", "HEAD")

	// The new workspace's repo2 should be on the fork's branch (not the
	// divergent branch), cut from the chosen base task/base.
	out := h.RunForTest(t, s, dstPath+"/repo2", nil, "git", "branch", "--show-current")
	if strings.TrimSpace(out) == "task/divergent" {
		t.Fatalf("forked repo2 is on the divergent branch %q; expected the fork's own branch", strings.TrimSpace(out))
	}
}

// TestForkDivergence_UniformSourceStillWorks is a regression guard: a source
// whose repos are all on the same branch must still fork without --base-branch
// (the divergence handling must not break the common case).
func TestForkDivergence_UniformSourceStillWorks(t *testing.T) {
	s := h.NewSandbox(t)
	defer s.Cleanup()

	remote := s.InitBareRepo(t, "remote")
	s.InitRepo(t, "repo1", remote)
	s.InitRepo(t, "repo2", remote)

	res := s.RunWSM(t, nil, s.ReposDir, "discover", s.ReposDir)
	if res.ExitCode != 0 {
		t.Fatalf("discover failed: %s\n%s", res.Stdout, res.Stderr)
	}

	res = s.RunWSM(t, nil, "", "create", "ws-uniform-src", "--repos", "repo1,repo2", "--branch", "task/base")
	if res.ExitCode != 0 {
		t.Fatalf("create source failed: %s\n%s", res.Stdout, res.Stderr)
	}
	wsPath := s.LoadWorkspacePath(t, "ws-uniform-src")

	// Uniform source: fork with no --base-branch must succeed.
	res = s.RunWSM(t, nil, wsPath, "fork", "ws-uniform-dst", "ws-uniform-src")
	if res.ExitCode != 0 {
		t.Fatalf("expected fork of uniform source to succeed without --base-branch, got:\n%s\n%s", res.Stdout, res.Stderr)
	}
}
