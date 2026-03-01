package registry

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/middlewares"
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
var _ cmds.GlazeCommand = &ListWorkspacesCommand{}

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

func (c *ListWorkspacesCommand) Run(ctx context.Context, _ *values.Values) error {
	workspaces, err := c.execute(ctx)
	if err != nil {
		return err
	}

	return printWorkspacesHuman(workspaces)
}

func (c *ListWorkspacesCommand) RunIntoGlazeProcessor(
	ctx context.Context,
	_ *values.Values,
	gp middlewares.Processor,
) error {
	workspaces, err := c.execute(ctx)
	if err != nil {
		return err
	}

	for _, workspace := range workspaces {
		repoNames := make([]string, 0, len(workspace.Repositories))
		for _, repo := range workspace.Repositories {
			repoNames = append(repoNames, repo.Name)
		}

		row := types.NewRow(
			types.MRP("name", workspace.Name),
			types.MRP("path", workspace.Path),
			types.MRP("branch", workspace.Branch),
			types.MRP("base_branch", workspace.BaseBranch),
			types.MRP("repository_count", len(workspace.Repositories)),
			types.MRP("repositories", repoNames),
			types.MRP("created", workspace.Created),
		)
		if err := gp.AddRow(ctx, row); err != nil {
			return errors.Wrap(err, "failed to add workspace row")
		}
	}

	return nil
}

func (c *ListWorkspacesCommand) execute(_ context.Context) ([]wsm.Workspace, error) {
	workflow, err := workflows.NewListWorkflow()
	if err != nil {
		return nil, err
	}

	workspaces, err := workflow.ListWorkspaces()
	if err != nil {
		return nil, err
	}

	return workspaces, nil
}

func printWorkspacesHuman(workspaces []wsm.Workspace) error {
	if len(workspaces) == 0 {
		output.PrintInfo("No workspaces found. Use 'wsm create' to create a workspace")
		return nil
	}

	output.PrintHeader("Workspaces (%d)", len(workspaces))
	fmt.Println()

	for _, workspace := range workspaces {
		repoNames := make([]string, len(workspace.Repositories))
		for i, repo := range workspace.Repositories {
			repoNames[i] = repo.Name
		}
		repos := strings.Join(repoNames, ", ")
		if repos == "" {
			repos = "-"
		} else if len(repos) > 80 {
			repos = repos[:77] + "..."
		}

		branch := workspace.Branch
		if branch == "" {
			branch = "-"
		}

		baseBranch := workspace.BaseBranch
		if baseBranch == "" {
			baseBranch = "-"
		}

		fmt.Printf("- %s [%s]\n", workspace.Name, branch)
		fmt.Printf("  path: %s\n", workspace.Path)
		fmt.Printf("  repos: %d (%s)\n", len(workspace.Repositories), repos)
		fmt.Printf("  base: %s\n", baseBranch)
		fmt.Printf("  created: %s\n", workspace.Created.Format("2006-01-02 15:04"))
	}

	return nil
}

func NewListWorkspacesCobraCommand() (*cobra.Command, error) {
	command, err := NewListWorkspacesCommand()
	if err != nil {
		return nil, fmt.Errorf("failed to build list workspaces command: %w", err)
	}
	return wsmcmdcommon.BuildCobraCommandDualMode(command)
}
