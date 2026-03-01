package branch

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// Register wires branch-oriented commands under git.
func Register(root *cobra.Command) error {
	branchCmd, err := NewBranchCobraCommand()
	if err != nil {
		return errors.Wrap(err, "failed to build branch command")
	}
	root.AddCommand(branchCmd)
	return nil
}
