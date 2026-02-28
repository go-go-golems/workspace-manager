package wsm

import (
	"context"
	branchsvc "github.com/go-go-golems/workspace-manager/pkg/wsm/branch"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func runGitInDirOrFail(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, string(out))
	}
	return strings.TrimSpace(string(out))
}

func createWorkspaceBranchFixture(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	remote := filepath.Join(tmp, "origin.git")
	seed := filepath.Join(tmp, "seed")
	client := filepath.Join(tmp, "client")

	runGitInDirOrFail(t, "", "init", "--bare", remote)
	runGitInDirOrFail(t, "", "clone", remote, seed)
	runGitInDirOrFail(t, seed, "config", "user.name", "WSM Test")
	runGitInDirOrFail(t, seed, "config", "user.email", "wsm-test@example.com")
	runGitInDirOrFail(t, seed, "checkout", "-b", "main")

	if err := os.WriteFile(filepath.Join(seed, "README.md"), []byte("seed\n"), 0o644); err != nil {
		t.Fatalf("write README: %v", err)
	}
	runGitInDirOrFail(t, seed, "add", "README.md")
	runGitInDirOrFail(t, seed, "commit", "-m", "seed commit")
	runGitInDirOrFail(t, seed, "push", "-u", "origin", "main")

	runGitInDirOrFail(t, seed, "checkout", "-b", "feature/remote-only")
	if err := os.WriteFile(filepath.Join(seed, "feature.txt"), []byte("feature\n"), 0o644); err != nil {
		t.Fatalf("write feature.txt: %v", err)
	}
	runGitInDirOrFail(t, seed, "add", "feature.txt")
	runGitInDirOrFail(t, seed, "commit", "-m", "remote branch commit")
	runGitInDirOrFail(t, seed, "push", "-u", "origin", "feature/remote-only")

	runGitInDirOrFail(t, "", "clone", "--branch", "main", remote, client)
	return client
}

func TestBranchServiceRemoteTrackingExists_BackendMatrix(t *testing.T) {
	repoPath := createWorkspaceBranchFixture(t)
	ctx := context.Background()

	for _, backend := range []string{"cli", "gogit", "hybrid"} {
		t.Run(backend, func(t *testing.T) {
			t.Setenv("WSM_GIT_BACKEND", backend)
			service := BuildBranchService(ctx)
			ok, err := service.RemoteTrackingExists(ctx, repoPath, branchsvc.DefaultRemoteName, branchsvc.BranchName("feature/remote-only"))
			if err != nil {
				t.Fatalf("check remote branch exists failed: %v", err)
			}
			if !ok {
				t.Fatalf("expected branch to exist in backend=%s", backend)
			}
		})
	}
}

func TestBranchServiceRemoteTrackingExists_MissingBranch(t *testing.T) {
	repoPath := createWorkspaceBranchFixture(t)
	ctx := context.Background()

	for _, backend := range []string{"cli", "gogit", "hybrid"} {
		t.Run(backend, func(t *testing.T) {
			t.Setenv("WSM_GIT_BACKEND", backend)
			service := BuildBranchService(ctx)
			ok, err := service.RemoteTrackingExists(ctx, repoPath, branchsvc.DefaultRemoteName, branchsvc.BranchName("feature/missing"))
			if err != nil {
				t.Fatalf("check remote branch exists failed: %v", err)
			}
			if ok {
				t.Fatalf("expected missing branch to return false in backend=%s", backend)
			}
		})
	}
}

func TestResolveBranchState_BranchSignals(t *testing.T) {
	repoPath := createWorkspaceBranchFixture(t)
	ctx := context.Background()
	wm := &WorkspaceManager{}

	plan, err := wm.resolveBranchPlan(ctx, repoPath, branchsvc.BranchResolutionRequest{
		Mode:         branchsvc.ResolutionModeCreateWorktree,
		TargetBranch: branchsvc.BranchName("feature/remote-only"),
		Remote:       branchsvc.DefaultRemoteName,
	})
	if err != nil {
		t.Fatalf("resolve branch plan failed: %v", err)
	}
	if plan.LocalExists {
		t.Fatalf("expected local branch to be false for remote-only branch")
	}
	if !plan.RemoteTrackingExists {
		t.Fatalf("expected remote branch to exist")
	}
	if plan.Strategy != branchsvc.ResolutionStrategyTrackRemote {
		t.Fatalf("expected track-remote strategy, got %v", plan.Strategy)
	}

	plan, err = wm.resolveBranchPlan(ctx, repoPath, branchsvc.BranchResolutionRequest{
		Mode:         branchsvc.ResolutionModeCreateWorktree,
		TargetBranch: branchsvc.BranchName("main"),
		Remote:       branchsvc.DefaultRemoteName,
	})
	if err != nil {
		t.Fatalf("resolve branch plan for main failed: %v", err)
	}
	if !plan.LocalExists {
		t.Fatalf("expected local main branch to exist")
	}
	if !plan.RemoteTrackingExists {
		t.Fatalf("expected remote main branch to exist")
	}
	if plan.Strategy != branchsvc.ResolutionStrategyUseLocal {
		t.Fatalf("expected use-local strategy, got %v", plan.Strategy)
	}

	plan, err = wm.resolveBranchPlan(ctx, repoPath, branchsvc.BranchResolutionRequest{
		Mode:         branchsvc.ResolutionModeCreateWorktree,
		TargetBranch: branchsvc.BranchName("feature/missing"),
		Remote:       branchsvc.DefaultRemoteName,
	})
	if err != nil {
		t.Fatalf("resolve branch plan for missing branch failed: %v", err)
	}
	if plan.LocalExists || plan.RemoteTrackingExists {
		t.Fatalf("expected missing branch to be false/false, got local=%v remote=%v", plan.LocalExists, plan.RemoteTrackingExists)
	}
	if plan.Strategy != branchsvc.ResolutionStrategyCreateFromHead {
		t.Fatalf("expected create-from-head strategy, got %v", plan.Strategy)
	}
}
