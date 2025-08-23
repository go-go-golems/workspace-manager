package wsm

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

// RebaseState represents the rebase state for a repository.
type RebaseState string

const (
	RebaseStateNone            RebaseState = "none"
	RebaseStateInProgress      RebaseState = "in-progress"
	RebaseStateStoppedConflicts RebaseState = "stopped-conflicts"
	RebaseStateCompleted       RebaseState = "completed"
)

// ConflictInfo describes a single conflicted path.
type ConflictInfo struct {
	File  string
	Type  string // UU, AA, DD, etc.
	Hints []string
}

// RebaseOptions control rebase behavior.
type RebaseOptions struct {
	PreserveMerges bool
	Autosquash     bool
}

// DetectUpstream resolves the upstream ref for the current branch.
// Fallback: origin/<current-branch> when @{upstream} is not set.
func DetectUpstream(ctx context.Context, repoPath string) (string, error) {
	// Try HEAD@{upstream}
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--abbrev-ref", "@{upstream}")
	cmd.Dir = repoPath
	out, err := cmd.Output()
	if err == nil {
		up := strings.TrimSpace(string(out))
		if up != "" && up != "@{upstream}" {
			return up, nil
		}
	}

	// Fallback to origin/<current-branch>
	branchCmd := exec.CommandContext(ctx, "git", "rev-parse", "--abbrev-ref", "HEAD")
	branchCmd.Dir = repoPath
	bOut, bErr := branchCmd.Output()
	if bErr != nil {
		return "", errors.Wrap(bErr, "detect current branch")
	}
	branch := strings.TrimSpace(string(bOut))
	if branch == "" || branch == "HEAD" {
		return "", errors.New("cannot determine current branch for upstream fallback")
	}
	return fmt.Sprintf("origin/%s", branch), nil
}

// Start performs a rebase onto the given upstream. If upstream is empty, DetectUpstream is used.
func Start(ctx context.Context, repoPath string, upstream string, opts RebaseOptions) error {
	if upstream == "" {
		u, err := DetectUpstream(ctx, repoPath)
		if err != nil {
			return err
		}
		upstream = u
	}

	args := []string{"rebase"}
	if opts.PreserveMerges {
		args = append(args, "--rebase-merges")
	}
	if opts.Autosquash {
		args = append(args, "--autosquash")
	}
	args = append(args, upstream)

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = repoPath
	out, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "git rebase failed: %s", string(out))
	}
	return nil
}

// Continue continues an in-progress rebase.
func Continue(ctx context.Context, repoPath string) error {
	cmd := exec.CommandContext(ctx, "git", "rebase", "--continue")
	cmd.Dir = repoPath
	out, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "git rebase --continue failed: %s", string(out))
	}
	return nil
}

// Abort aborts an in-progress rebase.
func Abort(ctx context.Context, repoPath string) error {
	cmd := exec.CommandContext(ctx, "git", "rebase", "--abort")
	cmd.Dir = repoPath
	out, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "git rebase --abort failed: %s", string(out))
	}
	return nil
}

// Status returns current rebase state and any conflicts.
func Status(ctx context.Context, repoPath string) (RebaseState, []ConflictInfo, error) {
	// Check for rebase directories
	rebaseMergeDir := filepath.Join(repoPath, ".git", "rebase-merge")
	rebaseApplyDir := filepath.Join(repoPath, ".git", "rebase-apply")
	inProgress := false
	if _, err := os.Stat(rebaseMergeDir); err == nil { inProgress = true }
	if _, err := os.Stat(rebaseApplyDir); err == nil { inProgress = true }

	// Inspect porcelain for conflicts
	cmd := exec.CommandContext(ctx, "git", "status", "--porcelain")
	cmd.Dir = repoPath
	out, err := cmd.Output()
	if err != nil {
		return RebaseStateNone, nil, err
	}

	var conflicts []ConflictInfo
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		if len(line) < 2 { continue }
		xy := line[:2]
		if xy == "UU" || xy == "AA" || xy == "DD" || xy[0] == 'U' || xy[1] == 'U' {
			parts := strings.SplitN(line, " ", 2)
			file := ""
			if len(parts) == 2 { file = strings.TrimSpace(parts[1]) }
			conflicts = append(conflicts, ConflictInfo{File: file, Type: xy})
		}
	}

	if inProgress {
		if len(conflicts) > 0 {
			return RebaseStateStoppedConflicts, conflicts, nil
		}
		return RebaseStateInProgress, nil, nil
	}

	return RebaseStateNone, conflicts, nil
}

// ListConflicts lists conflicted files using porcelain parsing.
func ListConflicts(ctx context.Context, repoPath string) ([]ConflictInfo, error) {
	_, conflicts, err := Status(ctx, repoPath)
	return conflicts, err
}

// OpenMergetool opens a mergetool/editor for the given files (or all when files is empty).
func OpenMergetool(ctx context.Context, repoPath string, tool string, files []string) error {
	args := []string{"mergetool"}
	if tool != "" {
		args = append(args, "--tool", tool)
	}
	if len(files) > 0 {
		args = append(args, files...)
	}
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = repoPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// StageResolved stages resolved files. If all=true and files empty, it stages all.
func StageResolved(ctx context.Context, repoPath string, all bool, files []string) error {
	if all {
		cmd := exec.CommandContext(ctx, "git", "add", "-A")
		cmd.Dir = repoPath
		if out, err := cmd.CombinedOutput(); err != nil {
			return errors.Wrapf(err, "git add -A failed: %s", string(out))
		}
		return nil
	}
	if len(files) == 0 {
		return errors.New("no files specified; use --all to stage all")
	}
	args := append([]string{"add"}, files...)
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = repoPath
	if out, err := cmd.CombinedOutput(); err != nil {
		return errors.Wrapf(err, "git add failed: %s", string(out))
	}
	return nil
}
