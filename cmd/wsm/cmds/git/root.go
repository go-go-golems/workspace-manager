package git

import (
	branchcmd "github.com/go-go-golems/workspace-manager/cmd/wsm/cmds/git/branch"
	rebasecmd "github.com/go-go-golems/workspace-manager/cmd/wsm/cmds/git/rebase"
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

	if err := branchcmd.Register(root); err != nil {
		return errors.Wrap(err, "failed to register branch commands")
	}

	if err := rebasecmd.Register(root); err != nil {
		return errors.Wrap(err, "failed to register rebase commands")
	}

	root.AddCommand(commitCmd, diffCmd, logCmd)
	return nil
}
