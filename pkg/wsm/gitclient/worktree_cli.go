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
    } else if branch != "" {
        args = append(args, "-b", branch)
    }
    args = append(args, targetPath)
    if opts.RemoteBranch != "" {
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
    out, err := runGit(ctx, repoPath, "worktree", "list")
    if err != nil { return nil, err }
    lines := strings.Split(strings.TrimSpace(string(out)), "\n")
    var infos []WorktreeInfo
    for _, l := range lines {
        if l == "" { continue }
        // Formats vary; common: "/path/to/wt  <hash> [branch]"
        // We'll parse path and try to derive branch from trailing
        parts := strings.Fields(l)
        if len(parts) == 0 { continue }
        wtPath := parts[0]
        var branch string
        if strings.HasSuffix(l, "]") {
            // find last '[' and ']'
            lb := strings.LastIndex(l, "[")
            rb := strings.LastIndex(l, "]")
            if lb >= 0 && rb > lb {
                branch = l[lb+1:rb]
            }
        }
        infos = append(infos, WorktreeInfo{ Path: filepath.Clean(wtPath), Branch: branch })
    }
    return infos, nil
}


