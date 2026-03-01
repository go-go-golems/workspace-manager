package wsm

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func runGitInDirExpectFail(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected git %s to fail", strings.Join(args, " "))
	}
	return string(out)
}

func createRebaseConflictWorktreeFixture(t *testing.T) (string, string) {
	t.Helper()
	tmp := t.TempDir()
	repoPath := filepath.Join(tmp, "repo")
	worktreePath := filepath.Join(tmp, "worktree-feature")

	runGitInDirOrFail(t, "", "init", repoPath)
	runGitInDirOrFail(t, repoPath, "config", "user.name", "WSM Test")
	runGitInDirOrFail(t, repoPath, "config", "user.email", "wsm-test@example.com")
	runGitInDirOrFail(t, repoPath, "checkout", "-b", "main")

	conflictFile := filepath.Join(repoPath, "conflict.txt")
	if err := os.WriteFile(conflictFile, []byte("base\n"), 0o644); err != nil {
		t.Fatalf("write base conflict file: %v", err)
	}
	runGitInDirOrFail(t, repoPath, "add", "conflict.txt")
	runGitInDirOrFail(t, repoPath, "commit", "-m", "base")

	runGitInDirOrFail(t, repoPath, "checkout", "-b", "feature/rebase")
	if err := os.WriteFile(conflictFile, []byte("feature\n"), 0o644); err != nil {
		t.Fatalf("write feature conflict file: %v", err)
	}
	runGitInDirOrFail(t, repoPath, "add", "conflict.txt")
	runGitInDirOrFail(t, repoPath, "commit", "-m", "feature change")

	runGitInDirOrFail(t, repoPath, "checkout", "main")
	if err := os.WriteFile(conflictFile, []byte("main\n"), 0o644); err != nil {
		t.Fatalf("write main conflict file: %v", err)
	}
	runGitInDirOrFail(t, repoPath, "add", "conflict.txt")
	runGitInDirOrFail(t, repoPath, "commit", "-m", "main change")

	runGitInDirOrFail(t, repoPath, "worktree", "add", worktreePath, "feature/rebase")
	runGitInDirExpectFail(t, worktreePath, "rebase", "main")

	return repoPath, worktreePath
}

func TestStatus_WorktreeRebaseStoppedConflicts(t *testing.T) {
	_, worktreePath := createRebaseConflictWorktreeFixture(t)

	state, conflicts, err := Status(context.Background(), worktreePath)
	if err != nil {
		t.Fatalf("status failed: %v", err)
	}
	if state != RebaseStateStoppedConflicts {
		t.Fatalf("expected stopped-conflicts, got %q", state)
	}
	if len(conflicts) == 0 {
		t.Fatalf("expected at least one conflict entry")
	}
}

func TestStatus_WorktreeRebaseInProgressWithoutConflicts(t *testing.T) {
	_, worktreePath := createRebaseConflictWorktreeFixture(t)

	if err := os.WriteFile(filepath.Join(worktreePath, "conflict.txt"), []byte("resolved\n"), 0o644); err != nil {
		t.Fatalf("write resolved conflict file: %v", err)
	}
	runGitInDirOrFail(t, worktreePath, "add", "conflict.txt")

	state, conflicts, err := Status(context.Background(), worktreePath)
	if err != nil {
		t.Fatalf("status failed: %v", err)
	}
	if state != RebaseStateInProgress {
		t.Fatalf("expected in-progress, got %q", state)
	}
	if len(conflicts) != 0 {
		t.Fatalf("expected no unresolved conflicts, got %d", len(conflicts))
	}
}
