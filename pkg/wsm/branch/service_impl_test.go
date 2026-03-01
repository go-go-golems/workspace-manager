package branch

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-go-golems/workspace-manager/pkg/wsm/gitclient"
)

func runGitSvcTest(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, string(out))
	}
}

func createServiceFixture(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	remote := filepath.Join(tmp, "origin.git")
	seed := filepath.Join(tmp, "seed")
	client := filepath.Join(tmp, "client")

	runGitSvcTest(t, "", "init", "--bare", remote)
	runGitSvcTest(t, "", "clone", remote, seed)
	runGitSvcTest(t, seed, "config", "user.name", "WSM Branch Service")
	runGitSvcTest(t, seed, "config", "user.email", "wsm-branch@example.com")
	runGitSvcTest(t, seed, "checkout", "-b", "main")

	if err := os.WriteFile(filepath.Join(seed, "README.md"), []byte("seed\n"), 0o644); err != nil {
		t.Fatalf("write README: %v", err)
	}
	runGitSvcTest(t, seed, "add", "README.md")
	runGitSvcTest(t, seed, "commit", "-m", "seed")
	runGitSvcTest(t, seed, "push", "-u", "origin", "main")

	runGitSvcTest(t, seed, "checkout", "-b", "feature/remote-only")
	if err := os.WriteFile(filepath.Join(seed, "feature.txt"), []byte("feature\n"), 0o644); err != nil {
		t.Fatalf("write feature.txt: %v", err)
	}
	runGitSvcTest(t, seed, "add", "feature.txt")
	runGitSvcTest(t, seed, "commit", "-m", "feature")
	runGitSvcTest(t, seed, "push", "-u", "origin", "feature/remote-only")

	runGitSvcTest(t, "", "clone", "--branch", "main", remote, client)
	return client
}

func TestServiceResolve_RemoteTrackingBranch(t *testing.T) {
	repoPath := createServiceFixture(t)
	svc := NewService(gitclient.NewCli(), DefaultRemoteName)
	ctx := context.Background()

	plan, err := svc.Resolve(ctx, repoPath, BranchResolutionRequest{
		Mode:         ResolutionModeCreateWorktree,
		TargetBranch: BranchName("feature/remote-only"),
	})
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	if plan.Strategy != ResolutionStrategyTrackRemote {
		t.Fatalf("expected track-remote, got %v", plan.Strategy)
	}
	if plan.RemoteRefKind != RemoteRefKindRemoteTrackingBranch {
		t.Fatalf("expected remote tracking ref kind")
	}
	if plan.RemoteRef != "origin/feature/remote-only" {
		t.Fatalf("unexpected remote ref: %s", plan.RemoteRef)
	}
}

func TestServiceResolve_LocalBranchWins(t *testing.T) {
	repoPath := createServiceFixture(t)
	svc := NewService(gitclient.NewCli(), DefaultRemoteName)
	ctx := context.Background()

	plan, err := svc.Resolve(ctx, repoPath, BranchResolutionRequest{
		Mode:         ResolutionModeCreateWorktree,
		TargetBranch: BranchName("main"),
	})
	if err != nil {
		t.Fatalf("resolve failed: %v", err)
	}
	if plan.Strategy != ResolutionStrategyUseLocal {
		t.Fatalf("expected use-local, got %v", plan.Strategy)
	}
}
