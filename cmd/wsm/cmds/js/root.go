package js

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// Register wires JS-oriented commands into the root command.
func Register(root *cobra.Command) error {
	runnerCmd, err := NewRunnerCobraCommand()
	if err != nil {
		return errors.Wrap(err, "failed to build runner command")
	}

	root.AddCommand(runnerCmd)
	return nil
}
