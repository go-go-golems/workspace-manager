package git

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// Register wires git-oriented commands into the root command.
func Register(root *cobra.Command) error {
	diffCmd, err := NewDiffCobraCommand()
	if err != nil {
		return errors.Wrap(err, "failed to build diff command")
	}

	logCmd, err := NewLogCobraCommand()
	if err != nil {
		return errors.Wrap(err, "failed to build log command")
	}

	branchCmd, err := NewBranchCobraCommand()
	if err != nil {
		return errors.Wrap(err, "failed to build branch command")
	}

	root.AddCommand(diffCmd, logCmd, branchCmd)
	return nil
}
