package registry

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
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
	settings_ := &ListReposSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, settings_); err != nil {
		return errors.Wrap(err, "failed to decode list repos settings")
	}

	workflow, err := workflows.NewListWorkflow()
	if err != nil {
		return err
	}
	repos, err := workflow.ListRepositories(settings_.Tags)
	if err != nil {
		return err
	}

	mode := wsmcmdcommon.ResolveOutputMode(vals)
	if wsmcmdcommon.ShouldOutputHuman(mode) {
		if err := printReposHuman(repos, settings_.Tags); err != nil {
			return err
		}
	}

	if wsmcmdcommon.ShouldOutputData(mode) {
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
		if err := wsmcmdcommon.EmitRows(ctx, vals, rows); err != nil {
			return errors.Wrap(err, "failed to emit repository rows")
		}
	}

	if !wsmcmdcommon.ShouldOutputHuman(mode) && !wsmcmdcommon.ShouldOutputData(mode) {
		return wsmcmdcommon.ErrUnsupportedOutputMode(mode)
	}

	return nil
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

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer func() {
		if err := w.Flush(); err != nil {
			output.PrintError("Failed to flush table writer: %v", err)
		}
	}()

	fmt.Fprintln(w, "NAME\tPATH\tBRANCH\tTAGS\tREMOTE")
	fmt.Fprintln(w, "----\t----\t------\t----\t------")

	for _, repo := range repos {
		tagsJoined := strings.Join(repo.Categories, ",")
		if len(tagsJoined) > 30 {
			tagsJoined = tagsJoined[:27] + "..."
		}

		remote := repo.RemoteURL
		if len(remote) > 50 {
			remote = "..." + remote[len(remote)-47:]
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			repo.Name,
			repo.Path,
			repo.CurrentBranch,
			tagsJoined,
			remote,
		)
	}

	return nil
}

func NewListReposCobraCommand() (*cobra.Command, error) {
	command, err := NewListReposCommand()
	if err != nil {
		return nil, fmt.Errorf("failed to build list repos command: %w", err)
	}
	return wsmcmdcommon.BuildCobraCommand(command)
}
