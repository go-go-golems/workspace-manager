package listcmd

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// Register wires list subcommands into registry root.
func Register(root *cobra.Command) error {
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

	root.AddCommand(listCmd)
	return nil
}
