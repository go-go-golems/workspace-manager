package registry

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/types"
	wsmcmdcommon "github.com/go-go-golems/workspace-manager/cmd/wsm/cmds/common"
	"github.com/go-go-golems/workspace-manager/pkg/output"
	"github.com/go-go-golems/workspace-manager/pkg/wsm"
	"github.com/go-go-golems/workspace-manager/pkg/wsm/workflows"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// ListWorkspacesCommand lists created workspaces.
type ListWorkspacesCommand struct {
	*cmds.CommandDescription
}

var _ cmds.BareCommand = &ListWorkspacesCommand{}

func NewListWorkspacesCommand() (*ListWorkspacesCommand, error) {
	desc, err := wsmcmdcommon.BuildDescription(
		"workspaces",
		cmds.WithShort("List created workspaces"),
		cmds.WithLong("List all created workspaces, sorted by creation date (newest first)."),
	)
	if err != nil {
		return nil, err
	}
	return &ListWorkspacesCommand{CommandDescription: desc}, nil
}

func (c *ListWorkspacesCommand) Run(ctx context.Context, vals *values.Values) error {
	workflow, err := workflows.NewListWorkflow()
	if err != nil {
		return err
	}

	workspaces, err := workflow.ListWorkspaces()
	if err != nil {
		return err
	}

	mode := wsmcmdcommon.ResolveOutputMode(vals)
	if wsmcmdcommon.ShouldOutputHuman(mode) {
		if err := printWorkspacesHuman(workspaces); err != nil {
			return err
		}
	}

	if wsmcmdcommon.ShouldOutputData(mode) {
		rows := make([]types.Row, 0, len(workspaces))
		for _, workspace := range workspaces {
			repoNames := make([]string, 0, len(workspace.Repositories))
			for _, repo := range workspace.Repositories {
				repoNames = append(repoNames, repo.Name)
			}

			rows = append(rows, types.NewRow(
				types.MRP("name", workspace.Name),
				types.MRP("path", workspace.Path),
				types.MRP("branch", workspace.Branch),
				types.MRP("base_branch", workspace.BaseBranch),
				types.MRP("repository_count", len(workspace.Repositories)),
				types.MRP("repositories", repoNames),
				types.MRP("created", workspace.Created),
			))
		}

		if err := wsmcmdcommon.EmitRows(ctx, vals, rows); err != nil {
			return errors.Wrap(err, "failed to emit workspace rows")
		}
	}

	if !wsmcmdcommon.ShouldOutputHuman(mode) && !wsmcmdcommon.ShouldOutputData(mode) {
		return wsmcmdcommon.ErrUnsupportedOutputMode(mode)
	}

	return nil
}

func printWorkspacesHuman(workspaces []wsm.Workspace) error {
	if len(workspaces) == 0 {
		output.PrintInfo("No workspaces found. Use 'wsm create' to create a workspace")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer func() {
		if err := w.Flush(); err != nil {
			output.PrintError("Failed to flush table writer: %v", err)
		}
	}()

	fmt.Fprintln(w, "NAME\tPATH\tREPOS\tBRANCH\tCREATED")
	fmt.Fprintln(w, "----\t----\t-----\t------\t-------")

	for _, workspace := range workspaces {
		repoNames := make([]string, len(workspace.Repositories))
		for i, repo := range workspace.Repositories {
			repoNames[i] = repo.Name
		}
		repos := strings.Join(repoNames, ",")
		if len(repos) > 30 {
			repos = repos[:27] + "..."
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			workspace.Name,
			workspace.Path,
			repos,
			workspace.Branch,
			workspace.Created.Format("2006-01-02 15:04"),
		)
	}

	return nil
}

func NewListWorkspacesCobraCommand() (*cobra.Command, error) {
	command, err := NewListWorkspacesCommand()
	if err != nil {
		return nil, fmt.Errorf("failed to build list workspaces command: %w", err)
	}
	return wsmcmdcommon.BuildCobraCommand(command)
}
