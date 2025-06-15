package cmds

import (
	"context"
	"fmt"
	"github.com/go-go-golems/workspace-manager/pkg/wsm"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
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
	// Default to current directory if no paths specified
	if len(paths) == 0 {
		cwd, err := os.Getwd()
		if err != nil {
			return errors.Wrap(err, "failed to get current directory")
		}
		paths = []string{cwd}
	}

	// Expand and validate paths
	var expandedPaths []string
	for _, path := range paths {
		// Expand ~ to home directory
		if path[0] == '~' {
			home, err := os.UserHomeDir()
			if err != nil {
				return errors.Wrap(err, "failed to get home directory")
			}
			path = filepath.Join(home, path[1:])
		}

		// Convert to absolute path
		absPath, err := filepath.Abs(path)
		if err != nil {
			return errors.Wrapf(err, "failed to get absolute path for %s", path)
		}

		// Check if path exists
		if _, err := os.Stat(absPath); os.IsNotExist(err) {
			return errors.Errorf("path does not exist: %s", absPath)
		}

		expandedPaths = append(expandedPaths, absPath)
	}

	// Get registry path
	registryPath, err := getRegistryPath()
	if err != nil {
		return errors.Wrap(err, "failed to get registry path")
	}

	// Create discoverer and load existing registry
	discoverer := wsm.NewRepositoryDiscoverer(registryPath)
	if err := discoverer.LoadRegistry(); err != nil {
		return errors.Wrap(err, "failed to load registry")
	}

	// Discover repositories
	fmt.Printf("Discovering repositories in %v...\n", expandedPaths)
	if err := discoverer.DiscoverRepositories(ctx, expandedPaths, recursive, maxDepth); err != nil {
		return errors.Wrap(err, "discovery failed")
	}

	// Show results
	repos := discoverer.GetRepositories()
	fmt.Printf("Discovery complete! Found %d repositories.\n", len(repos))

	if len(repos) > 0 {
		fmt.Println("\nUse 'workspace-manager list repos' to see all discovered repositories.")
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
