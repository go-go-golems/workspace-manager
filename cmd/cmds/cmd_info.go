package cmds

import (
	"context"
	"fmt"
	"github.com/go-go-golems/workspace-manager/pkg/output"
	"github.com/go-go-golems/workspace-manager/pkg/wsm"
	"github.com/go-go-golems/workspace-manager/pkg/wsm/workflows"

	"github.com/carapace-sh/carapace"
	"github.com/spf13/cobra"
)

func NewInfoCommand() *cobra.Command {
	var (
		outputFormat string
		outputField  string
		workspace    string
	)

	cmd := &cobra.Command{
		Use:   "info [workspace-name]",
		Short: "Display workspace information",
		Long: `Display information about a workspace.

By default, shows all workspace information. Use --field to get a specific piece of information.

Available fields:
  - path: workspace directory path
  - name: workspace name  
  - branch: workspace branch
  - repositories: number of repositories
  - created: creation date and time (YYYY-MM-DD HH:MM:SS)
  - date: creation date only (YYYY-MM-DD)
  - time: creation time only (HH:MM:SS)

Examples:
  # Show all workspace info
  workspace-manager info my-workspace

  # Get just the path (useful for cd $(wsm info my-workspace --field path))
  workspace-manager info my-workspace --field path

  # Get workspace name
  workspace-manager info --field name

  # JSON output
  workspace-manager info my-workspace --output json`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			workspaceName := workspace
			if len(args) > 0 {
				workspaceName = args[0]
			}
			return runInfo(cmd.Context(), workspaceName, outputFormat, outputField)
		},
	}

	cmd.Flags().StringVarP(&outputFormat, "output", "o", "table", "Output format (table, json)")
	cmd.Flags().StringVar(&outputField, "field", "", "Output specific field only (path, name, branch, repositories, created, date, time)")
	cmd.Flags().StringVar(&workspace, "workspace", "", "Workspace name")

	carapace.Gen(cmd).PositionalCompletion(WorkspaceNameCompletion())

	return cmd
}

func runInfo(ctx context.Context, workspaceName string, outputFormat, outputField string) error {
	workflow := workflows.NewInfoWorkflow()
	workspace, err := workflow.ResolveWorkspace(workspaceName)
	if err != nil {
		return err
	}

	// Handle field-specific output
	if outputField != "" {
		value, err := workflow.FieldValue(workspace, outputField)
		if err != nil {
			return err
		}
		fmt.Println(value)
		return nil
	}

	// Handle JSON output
	if outputFormat == "json" {
		return wsm.PrintJSON(workspace)
	}

	// Default table output
	return printInfoTable(workspace)
}

func printInfoTable(workspace *wsm.Workspace) error {
	output.PrintHeader("Workspace Information")
	fmt.Printf("  Name:         %s\n", workspace.Name)
	fmt.Printf("  Path:         %s\n", workspace.Path)
	fmt.Printf("  Branch:       %s\n", workspace.Branch)
	fmt.Printf("  Repositories: %d\n", len(workspace.Repositories))
	fmt.Printf("  Created:      %s\n", workspace.Created.Format("2006-01-02 15:04:05"))
	fmt.Printf("  Go Workspace: %t\n", workspace.GoWorkspace)

	if len(workspace.Repositories) > 0 {
		output.PrintHeader("\nRepositories")
		for _, repo := range workspace.Repositories {
			fmt.Printf("  - %s (%s)\n", repo.Name, repo.RemoteURL)
		}
	}

	return nil
}
