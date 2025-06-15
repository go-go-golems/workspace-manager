package cmds

import (
	"context"
	"fmt"
	"github.com/go-go-golems/workspace-manager/pkg/wsm"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func NewDiffCommand() *cobra.Command {
	var (
		staged bool
		repo   string
	)

	cmd := &cobra.Command{
		Use:   "diff",
		Short: "Show diff across workspace repositories",
		Long: `Show unified diff of changes across all repositories in the workspace.
This provides a consolidated view of all modifications in your multi-repository development.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDiff(cmd.Context(), staged, repo)
		},
	}

	cmd.Flags().BoolVar(&staged, "staged", false, "Show staged changes only")
	cmd.Flags().StringVar(&repo, "repo", "", "Show diff for specific repository only")

	return cmd
}

func runDiff(ctx context.Context, staged bool, repoFilter string) error {
	workspace, err := detectCurrentWorkspace()
	if err != nil {
		return errors.Wrap(err, "failed to detect current workspace")
	}

	gitOps := wsm.NewGitOperations(workspace)

	fmt.Printf("📄 Showing diff for workspace: %s\n", workspace.Name)
	if staged {
		fmt.Println("   (staged changes only)")
	}
	if repoFilter != "" {
		fmt.Printf("   (repository: %s)\n", repoFilter)
	}
	fmt.Println()

	diff, err := gitOps.GetDiff(ctx, staged, repoFilter)
	if err != nil {
		return errors.Wrap(err, "failed to get diff")
	}

	if diff == "" || diff == "No changes found in workspace." {
		fmt.Println("No changes found in workspace.")
		return nil
	}

	fmt.Println(diff)
	return nil
}

func NewLogCommand() *cobra.Command {
	var (
		since   string
		oneline bool
		limit   int
	)

	cmd := &cobra.Command{
		Use:   "log",
		Short: "Show commit history across workspace repositories",
		Long: `Show commit history spanning multiple repositories in the workspace.
This provides a unified view of development activity across your projects.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLog(cmd.Context(), since, oneline, limit)
		},
	}

	cmd.Flags().StringVar(&since, "since", "", "Show commits since date (e.g., '1 week ago')")
	cmd.Flags().BoolVar(&oneline, "oneline", false, "Show one line per commit")
	cmd.Flags().IntVar(&limit, "limit", 10, "Limit number of commits per repository")

	return cmd
}

func runLog(ctx context.Context, since string, oneline bool, limit int) error {
	workspace, err := detectCurrentWorkspace()
	if err != nil {
		return errors.Wrap(err, "failed to detect current workspace")
	}

	syncOps := wsm.NewSyncOperations(workspace)

	fmt.Printf("📜 Commit history for workspace: %s\n", workspace.Name)
	if since != "" {
		fmt.Printf("   (since: %s)\n", since)
	}
	fmt.Println()

	logs, err := syncOps.GetWorkspaceLog(ctx, since, oneline, limit)
	if err != nil {
		return errors.Wrap(err, "failed to get workspace log")
	}

	if len(logs) == 0 {
		fmt.Println("No commits found in workspace.")
		return nil
	}

	for repoName, log := range logs {
		if log == "" {
			continue
		}

		fmt.Printf("=== Repository: %s ===\n", repoName)
		fmt.Println(log)
		fmt.Println()
	}

	return nil
}
