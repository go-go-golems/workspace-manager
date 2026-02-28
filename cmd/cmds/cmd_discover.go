package cmds

import (
	"context"
	"github.com/go-go-golems/workspace-manager/pkg/output"
	"github.com/go-go-golems/workspace-manager/pkg/wsm/workflows"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func NewDiscoverCommand() *cobra.Command {
	var (
		recursive bool
		maxDepth  int
	)

	cmd := &cobra.Command{
		Use:   "discover [paths...]",
		Short: "Discover git repositories in specified directories",
		Long: `Discover git repositories in the specified directories and add them to the registry.
If no paths are specified, defaults to current directory.`,
		Args: cobra.MinimumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDiscover(cmd.Context(), args, recursive, maxDepth)
		},
	}

	cmd.Flags().BoolVarP(&recursive, "recursive", "r", true, "Recursively scan subdirectories")
	cmd.Flags().IntVar(&maxDepth, "max-depth", 3, "Maximum depth for recursive scanning")

	return cmd
}

func runDiscover(ctx context.Context, paths []string, recursive bool, maxDepth int) error {
	workflow, err := workflows.NewDiscoverWorkflow()
	if err != nil {
		return err
	}
	result, err := workflow.Discover(ctx, workflows.DiscoverRequest{
		Paths:     paths,
		Recursive: recursive,
		MaxDepth:  maxDepth,
	})
	if err != nil {
		return err
	}

	output.PrintInfo("Discovering repositories in %v", result.Paths)
	output.PrintSuccess("Discovery complete! Found %d repositories", result.RepositoryCount)

	if result.RepositoryCount > 0 {
		output.PrintInfo("Use 'workspace-manager list repos' to see all discovered repositories")
	}

	return nil
}

// getRegistryPath returns the path to the registry file
func getRegistryPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "workspace-manager", "registry.json"), nil
}
