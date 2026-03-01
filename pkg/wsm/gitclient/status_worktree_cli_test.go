package gitclient

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func createLocalRepoFixture(t *testing.T) string {
	t.Helper()
	repo := filepath.Join(t.TempDir(), "repo")
	runGitOrFail(t, "", "init", repo)
	runGitOrFail(t, repo, "config", "user.name", "WSM Test")
	runGitOrFail(t, repo, "config", "user.email", "wsm-test@example.com")
	runGitOrFail(t, repo, "checkout", "-b", "main")

	if err := os.WriteFile(filepath.Join(repo, "a.txt"), []byte("base\n"), 0o644); err != nil {
		t.Fatalf("write a.txt: %v", err)
	}
	runGitOrFail(t, repo, "add", "a.txt")
	runGitOrFail(t, repo, "commit", "-m", "base commit")

	return repo
}

func contains(ss []string, v string) bool {
	for _, s := range ss {
		if s == v {
			return true
		}
	}
	return false
}

func TestCliStatus_ParsesPorcelainZ(t *testing.T) {
	repo := createLocalRepoFixture(t)
	runGitOrFail(t, repo, "mv", "a.txt", "renamed file.txt")
	if err := os.WriteFile(filepath.Join(repo, "renamed file.txt"), []byte("renamed\nwith unstaged change\n"), 0o644); err != nil {
		t.Fatalf("write renamed file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repo, "new file.txt"), []byte("new\n"), 0o644); err != nil {
		t.Fatalf("write new file: %v", err)
	}

	c := NewCli()
	ctx := context.Background()
	h, err := c.Open(ctx, repo)
	if err != nil {
		t.Fatalf("open repo: %v", err)
	}
	st, err := c.Status(ctx, h)
	if err != nil {
		t.Fatalf("status failed: %v", err)
	}

	if st.CurrentBranch != "main" {
		t.Fatalf("expected main branch, got %q", st.CurrentBranch)
	}
	if !contains(st.StagedFiles, "renamed file.txt") {
		t.Fatalf("expected staged rename target, got %#v", st.StagedFiles)
	}
	if !contains(st.ModifiedFiles, "renamed file.txt") {
		t.Fatalf("expected unstaged modification for renamed file, got %#v", st.ModifiedFiles)
	}
	if !contains(st.UntrackedFiles, "new file.txt") {
		t.Fatalf("expected untracked file, got %#v", st.UntrackedFiles)
	}
	if contains(st.StagedFiles, "a.txt") || contains(st.ModifiedFiles, "a.txt") {
		t.Fatalf("did not expect old rename source path in status arrays: staged=%#v modified=%#v", st.StagedFiles, st.ModifiedFiles)
	}
}

func TestCliWorktreeList_PreservesPathsWithSpaces(t *testing.T) {
	repo := createLocalRepoFixture(t)
	targetPath := filepath.Join(filepath.Dir(repo), "worktrees", "feature space")

	w := NewCliWorktrees()
	ctx := context.Background()
	if err := w.Add(ctx, repo, "feature/space", targetPath, WorktreeAddOptions{}); err != nil {
		t.Fatalf("worktree add failed: %v", err)
	}

	infos, err := w.List(ctx, repo)
	if err != nil {
		t.Fatalf("worktree list failed: %v", err)
	}

	wantPath := filepath.Clean(targetPath)
	found := false
	for _, info := range infos {
		if filepath.Clean(info.Path) == wantPath {
			found = true
			if info.Branch != "feature/space" {
				t.Fatalf("expected branch feature/space, got %q (all=%#v)", info.Branch, infos)
			}
		}
	}
	if !found {
		var paths []string
		for _, info := range infos {
			paths = append(paths, info.Path)
		}
		t.Fatalf("worktree path with spaces not found. want=%q got=%s", wantPath, strings.Join(paths, ", "))
	}
}

func TestCliWorktreeAdd_UsesExistingBranchWhenRequested(t *testing.T) {
	repo := createLocalRepoFixture(t)
	runGitOrFail(t, repo, "checkout", "-b", "feature/existing")
	runGitOrFail(t, repo, "checkout", "main")

	targetPath := filepath.Join(filepath.Dir(repo), "worktrees", "feature existing")

	w := NewCliWorktrees()
	ctx := context.Background()
	if err := w.Add(ctx, repo, "feature/existing", targetPath, WorktreeAddOptions{UseExistingBranch: true}); err != nil {
		t.Fatalf("worktree add (existing branch) failed: %v", err)
	}

	current := runGitOrFail(t, targetPath, "rev-parse", "--abbrev-ref", "HEAD")
	if current != "feature/existing" {
		t.Fatalf("expected worktree on feature/existing, got %q", current)
	}
}
