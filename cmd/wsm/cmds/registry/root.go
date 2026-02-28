package registry

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// Register wires registry-oriented commands into the root command.
func Register(root *cobra.Command) error {
	discoverCmd, err := NewDiscoverCobraCommand()
	if err != nil {
		return errors.Wrap(err, "failed to build discover command")
	}

	listReposCmd, err := NewListReposCobraCommand()
	if err != nil {
		return errors.Wrap(err, "failed to build list repos command")
	}

	listWorkspacesCmd, err := NewListWorkspacesCobraCommand()
	if err != nil {
		return errors.Wrap(err, "failed to build list workspaces command")
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List repositories and workspaces",
		Long:  "List discovered repositories and created workspaces.",
	}
	listCmd.AddCommand(listReposCmd, listWorkspacesCmd)

	root.AddCommand(discoverCmd, listCmd)
	return nil
}
