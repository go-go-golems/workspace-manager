package cmds

import (
	"github.com/go-go-golems/workspace-manager/pkg/wsm/workflows"
	"github.com/spf13/cobra"
)

func NewMergeCommand() *cobra.Command {
	var (
		dryRun        bool
		force         bool
		workspace     string
		keepWorkspace bool
	)

	cmd := &cobra.Command{
		Use:   "merge [workspace-name]",
		Short: "Merge a forked workspace back into its base branch and delete the workspace",
		Long: `Merge a forked workspace back into its base branch and optionally delete the workspace.

This command:
1. Detects the current workspace (if not specified)
2. Verifies the workspace is a fork (has a base branch)
3. Checks if a workspace exists for the base branch and enforces running from within it
4. Checks that all repositories are clean before merging
5. For each repository:
   - Switches to the base branch
   - Merges the workspace branch into the base branch
   - Pushes the merged changes
6. Optionally deletes the workspace after successful merge`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			workspaceName := workspace
			if len(args) > 0 {
				workspaceName = args[0]
			}
			workflow := workflows.NewMergeWorkflow()
			return workflow.Execute(cmd.Context(), workflows.MergeRequest{
				WorkspaceName: workspaceName,
				DryRun:        dryRun,
				Force:         force,
				KeepWorkspace: keepWorkspace,
			})
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be merged without executing")
	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompts")
	cmd.Flags().StringVar(&workspace, "workspace", "", "Workspace name")
	cmd.Flags().BoolVar(&keepWorkspace, "keep-workspace", false, "Keep the workspace after merge (don't delete it)")

	return cmd
}
