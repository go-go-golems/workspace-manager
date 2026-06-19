package registry

import (
	listcmd "github.com/go-go-golems/workspace-manager/cmd/wsm/cmds/registry/list"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// Register wires registry-oriented commands into the root command.
func Register(root *cobra.Command) error {
	discoverCmd, err := NewDiscoverCobraCommand()
	if err != nil {
		return errors.Wrap(err, "failed to build discover command")
	}

	validateCmd, err := NewValidateCobraCommand()
	if err != nil {
		return errors.Wrap(err, "failed to build validate command")
	}

	pruneCmd, err := NewPruneCobraCommand()
	if err != nil {
		return errors.Wrap(err, "failed to build prune command")
	}

	if err := listcmd.Register(root); err != nil {
		return errors.Wrap(err, "failed to register list commands")
	}

	root.AddCommand(discoverCmd, validateCmd, pruneCmd)
	return nil
}
