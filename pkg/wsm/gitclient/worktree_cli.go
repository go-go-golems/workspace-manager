package gitclient

import (
	"context"
	"path/filepath"
	"strings"
)

// CliWorktrees implements WorktreeManager using the git CLI.
type CliWorktrees struct{}

func NewCliWorktrees() *CliWorktrees { return &CliWorktrees{} }

func (w *CliWorktrees) Add(ctx context.Context, repoPath string, branch string, targetPath string, opts WorktreeAddOptions) error {
	args := []string{"worktree", "add"}
	if opts.Overwrite {
		args = append(args, "-B", branch)
	} else if branch != "" && !opts.UseExistingBranch {
		args = append(args, "-b", branch)
	}
	args = append(args, targetPath)
	if branch != "" && opts.UseExistingBranch {
		args = append(args, branch)
	} else if opts.RemoteBranch != "" {
		args = append(args, opts.RemoteBranch)
	} else if opts.BaseRef != "" && !opts.Overwrite {
		// for non-overwrite new branch creation from base ref
		args = append(args, opts.BaseRef)
	}
	_, err := runGit(ctx, repoPath, args...)
	return err
}

func (w *CliWorktrees) Remove(ctx context.Context, repoPath string, targetPath string, force bool) error {
	args := []string{"worktree", "remove"}
	if force {
		args = append(args, "--force")
	}
	args = append(args, targetPath)
	_, err := runGit(ctx, repoPath, args...)
	return err
}

func (w *CliWorktrees) List(ctx context.Context, repoPath string) ([]WorktreeInfo, error) {
	out, err := runGit(ctx, repoPath, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(out), "\n")
	infos := make([]WorktreeInfo, 0)
	current := WorktreeInfo{}
	inRecord := false

	flush := func() {
		if !inRecord {
			return
		}
		if current.Path != "" {
			current.Path = filepath.Clean(current.Path)
			infos = append(infos, current)
		}
		current = WorktreeInfo{}
		inRecord = false
	}

	for _, line := range lines {
		line = strings.TrimSuffix(line, "\r")
		if line == "" {
			flush()
			continue
		}

		if strings.HasPrefix(line, "worktree ") {
			flush()
			current.Path = strings.TrimPrefix(line, "worktree ")
			inRecord = true
			continue
		}

		if !inRecord {
			continue
		}
		if strings.HasPrefix(line, "branch ") {
			ref := strings.TrimPrefix(line, "branch ")
			current.Branch = strings.TrimPrefix(ref, "refs/heads/")
		}
	}

	flush()
	return infos, nil
}
