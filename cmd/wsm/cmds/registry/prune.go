package registry

import (
	"fmt"

	"github.com/go-go-golems/workspace-manager/pkg/output"
	"github.com/go-go-golems/workspace-manager/pkg/wsm"
	"github.com/spf13/cobra"
)

func NewPruneCobraCommand() (*cobra.Command, error) {
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "prune",
		Short: "Remove stale repositories from registry",
		Long: `Remove repositories from the registry that no longer exist on disk.

Use --dry-run to preview what would be removed without making changes.`,
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
				output.PrintSuccess("Registry is clean - no stale repositories to remove")
				return nil
			}

			if dryRun {
				output.PrintInfo("Would remove %d stale repositories:", len(result.StaleRepos))
				fmt.Println()
				for _, repo := range result.StaleRepos {
					fmt.Printf("  %s\n", output.DimStyle.Render(repo.Path))
				}
				return nil
			}

			discoverer.RemoveRepositories(result.StaleRepos)
			if err := discoverer.SaveRegistry(); err != nil {
				return err
			}

			output.PrintSuccess("Removed %d stale repositories from registry", len(result.StaleRepos))
			for _, repo := range result.StaleRepos {
				fmt.Printf("  %s\n", output.DimStyle.Render(repo.Path))
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "Show what would be removed without making changes")

	return cmd, nil
}
