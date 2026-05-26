package listcmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
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

// ListReposCommand lists discovered repositories.
type ListReposCommand struct {
	*cmds.CommandDescription
}

// ListReposSettings stores parsed list repos settings.
type ListReposSettings struct {
	Tags []string `glazed:"tags"`
}

var _ cmds.BareCommand = &ListReposCommand{}
var _ cmds.GlazeCommand = &ListReposCommand{}

func NewListReposCommand() (*ListReposCommand, error) {
	desc, err := wsmcmdcommon.BuildDescription(
		"repos",
		cmds.WithShort("List discovered repositories"),
		cmds.WithLong("List all discovered repositories with optional filtering by tags."),
		cmds.WithFlags(
			fields.New(
				"tags",
				fields.TypeStringList,
				fields.WithHelp("Filter by tags (comma-separated)"),
			),
		),
	)
	if err != nil {
		return nil, err
	}
	return &ListReposCommand{CommandDescription: desc}, nil
}

func (c *ListReposCommand) Run(ctx context.Context, vals *values.Values) error {
	repos, tags, err := c.execute(ctx, vals)
	if err != nil {
		return err
	}

	return printReposHuman(repos, tags)
}

func (c *ListReposCommand) RunIntoGlazeProcessor(
	ctx context.Context,
	vals *values.Values,
	gp middlewares.Processor,
) error {
	repos, _, err := c.execute(ctx, vals)
	if err != nil {
		return err
	}

	for _, row := range reposToRows(repos) {
		if err := gp.AddRow(ctx, row); err != nil {
			return errors.Wrap(err, "failed to add repository row")
		}
	}

	return nil
}

func (c *ListReposCommand) execute(_ context.Context, vals *values.Values) ([]wsm.Repository, []string, error) {
	settings_ := &ListReposSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, settings_); err != nil {
		return nil, nil, errors.Wrap(err, "failed to decode list repos settings")
	}

	workflow, err := workflows.NewListWorkflow()
	if err != nil {
		return nil, nil, err
	}
	repos, err := workflow.ListRepositories(settings_.Tags)
	if err != nil {
		return nil, nil, err
	}

	return repos, settings_.Tags, nil
}

func reposToRows(repos []wsm.Repository) []types.Row {
	rows := make([]types.Row, 0, len(repos))
	for _, repo := range repos {
		rows = append(rows, types.NewRow(
			types.MRP("name", repo.Name),
			types.MRP("path", repo.Path),
			types.MRP("current_branch", repo.CurrentBranch),
			types.MRP("remote_url", repo.RemoteURL),
			types.MRP("categories", repo.Categories),
			types.MRP("last_updated", repo.LastUpdated),
		))
	}
	return rows
}

func printReposHuman(repos []wsm.Repository, tags []string) error {
	if len(repos) == 0 {
		if len(tags) > 0 {
			output.PrintInfo("No repositories found with tags: %s", strings.Join(tags, ", "))
		} else {
			output.PrintInfo("No repositories found. Run 'wsm discover' to scan for repositories")
		}
		return nil
	}

	var md strings.Builder
	_, _ = fmt.Fprintf(&md, "# Repositories (%d)\n\n", len(repos))
	if len(tags) > 0 {
		_, _ = fmt.Fprintf(&md, "> Filter tags: `%s`\n\n", strings.Join(tags, ", "))
	}

	for _, repo := range repos {
		tagsJoined := strings.Join(repo.Categories, ", ")
		if tagsJoined == "" {
			tagsJoined = "-"
		} else if len(tagsJoined) > 64 {
			tagsJoined = tagsJoined[:61] + "..."
		}

		branch := repo.CurrentBranch
		if branch == "" {
			branch = "-"
		}

		remote := repo.RemoteURL
		if remote == "" {
			remote = "-"
		} else if len(remote) > 96 {
			remote = remote[:93] + "..."
		}

		_, _ = fmt.Fprintf(&md, "## %s\n\n", repo.Name)
		_, _ = fmt.Fprintf(&md, "- **Branch:** `%s`\n", branch)
		_, _ = fmt.Fprintf(&md, "- **Path:** `%s`\n", repo.Path)
		_, _ = fmt.Fprintf(&md, "- **Tags:** %s\n", tagsJoined)
		_, _ = fmt.Fprintf(&md, "- **Remote:** `%s`\n\n", remote)
	}

	output.PrintMarkdown(md.String())
	return nil
}

func NewListReposCobraCommand() (*cobra.Command, error) {
	command, err := NewListReposCommand()
	if err != nil {
		return nil, fmt.Errorf("failed to build list repos command: %w", err)
	}
	return wsmcmdcommon.BuildCobraCommandDualMode(command)
}
