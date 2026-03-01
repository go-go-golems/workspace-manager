package rebase

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// Register wires rebase-oriented commands under git.
func Register(root *cobra.Command) error {
	rebaseCmd, err := NewRebaseCobraCommand()
	if err != nil {
		return errors.Wrap(err, "failed to build rebase command")
	}
	root.AddCommand(rebaseCmd)
	return nil
}
