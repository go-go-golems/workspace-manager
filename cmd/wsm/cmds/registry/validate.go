package registry

import (
	"fmt"

	"github.com/go-go-golems/workspace-manager/pkg/output"
	"github.com/go-go-golems/workspace-manager/pkg/wsm"
	"github.com/spf13/cobra"
)

func NewValidateCobraCommand() (*cobra.Command, error) {
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate registry against disk",
		Long: `Check which repositories in the registry still exist on disk.

Shows repositories that are registered but no longer exist at their recorded paths.
Use 'wsm prune' to remove stale entries from the registry.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			registryPath, err := wsm.GetRegistryPath()
			if err != nil {
				return err
			}

			discoverer := wsm.NewRepositoryDiscoverer(registryPath)
			if err := discoverer.LoadRegistry(); err != nil {
				return err
			}

			result := discoverer.ValidateRegistry()

			if len(result.StaleRepos) == 0 {
				output.PrintSuccess("All %d registered repositories exist on disk", len(result.ValidRepos))
				return nil
			}

			output.PrintWarning("Found %d stale repositories (not on disk):", len(result.StaleRepos))
			fmt.Println()
			for _, repo := range result.StaleRepos {
				fmt.Printf("  %s\n", output.DimStyle.Render(repo.Path))
			}
			fmt.Println()
			output.PrintInfo("Run 'wsm prune' to remove stale entries from the registry")

			return nil
		},
	}

	return cmd, nil
}
