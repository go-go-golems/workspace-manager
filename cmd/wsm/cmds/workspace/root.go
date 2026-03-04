package workspace

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// Register wires workspace-oriented commands into the root command.
func Register(root *cobra.Command) error {
	infoCmd, err := NewInfoCobraCommand()
	if err != nil {
		return errors.Wrap(err, "failed to build info command")
	}

	addCmd, err := NewAddCobraCommand()
	if err != nil {
		return errors.Wrap(err, "failed to build add command")
	}

	removeCmd, err := NewRemoveCobraCommand()
	if err != nil {
		return errors.Wrap(err, "failed to build remove command")
	}

	statusCmd, err := NewStatusCobraCommand()
	if err != nil {
		return errors.Wrap(err, "failed to build status command")
	}

	createCmd, err := NewCreateCobraCommand()
	if err != nil {
		return errors.Wrap(err, "failed to build create command")
	}

	forkCmd, err := NewForkCobraCommand()
	if err != nil {
		return errors.Wrap(err, "failed to build fork command")
	}

	deleteCmd, err := NewDeleteCobraCommand()
	if err != nil {
		return errors.Wrap(err, "failed to build delete command")
	}

	mergeCmd, err := NewMergeCobraCommand()
	if err != nil {
		return errors.Wrap(err, "failed to build merge command")
	}

	pathCmd, err := NewPathCobraCommand()
	if err != nil {
		return errors.Wrap(err, "failed to build path command")
	}

	root.AddCommand(infoCmd, addCmd, removeCmd, statusCmd, createCmd, forkCmd, deleteCmd, mergeCmd, pathCmd)
	return nil
}
