package git

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// Register wires git-oriented commands into the root command.
func Register(root *cobra.Command) error {
	commitCmd, err := NewCommitCobraCommand()
	if err != nil {
		return errors.Wrap(err, "failed to build commit command")
	}

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

	rebaseCmd, err := NewRebaseCobraCommand()
	if err != nil {
		return errors.Wrap(err, "failed to build rebase command")
	}

	root.AddCommand(commitCmd, diffCmd, logCmd, branchCmd, rebaseCmd)
	return nil
}
