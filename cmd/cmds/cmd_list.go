package cmds

import (
	"fmt"
	"github.com/go-go-golems/workspace-manager/pkg/wsm"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func NewListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List repositories and workspaces",
		Long:  "List discovered repositories and created workspaces.",
	}

	cmd.AddCommand(
		NewListReposCommand(),
		NewListWorkspacesCommand(),
	)

	return cmd
}

func NewListReposCommand() *cobra.Command {
	var (
		format string
		tags   []string
	)

	cmd := &cobra.Command{
		Use:   "repos",
		Short: "List discovered repositories",
		Long:  "List all discovered repositories with optional filtering by tags.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runListRepos(format, tags)
		},
	}

	cmd.Flags().StringVar(&format, "format", "table", "Output format: table, json")
	cmd.Flags().StringSliceVar(&tags, "tags", nil, "Filter by tags (comma-separated)")

	return cmd
}

func NewListWorkspacesCommand() *cobra.Command {
	var format string

	cmd := &cobra.Command{
		Use:   "workspaces",
		Short: "List created workspaces",
		Long:  "List all created workspaces, sorted by creation date (newest first).",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runListWorkspaces(format)
		},
	}

	cmd.Flags().StringVar(&format, "format", "table", "Output format: table, json")

	return cmd
}

func runListRepos(format string, tags []string) error {
	// Get registry path and load registry
	registryPath, err := getRegistryPath()
	if err != nil {
		return errors.Wrap(err, "failed to get registry path")
	}

	discoverer := wsm.NewRepositoryDiscoverer(registryPath)
	if err := discoverer.LoadRegistry(); err != nil {
		return errors.Wrap(err, "failed to load registry")
	}

	// Get repositories, optionally filtered by tags
	repos := discoverer.GetRepositoriesByTags(tags)

	if len(repos) == 0 {
		if len(tags) > 0 {
			fmt.Printf("No repositories found with tags: %s\n", strings.Join(tags, ", "))
		} else {
			fmt.Println("No repositories found. Run 'workspace-manager discover' to scan for repositories.")
		}
		return nil
	}

	switch format {
	case "table":
		return printReposTable(repos)
	case "json":
		return printReposJSON(repos)
	default:
		return errors.Errorf("unsupported format: %s", format)
	}
}

func runListWorkspaces(format string) error {
	workspaces, err := wsm.LoadWorkspaces()
	if err != nil {
		return errors.Wrap(err, "failed to load workspaces")
	}

	if len(workspaces) == 0 {
		fmt.Println("No workspaces found. Use 'workspace-manager create' to create a workspace.")
		return nil
	}

	// Sort workspaces by creation date descending (newest first)
	sort.Slice(workspaces, func(i, j int) bool {
		return workspaces[i].Created.After(workspaces[j].Created)
	})

	switch format {
	case "table":
		return printWorkspacesTable(workspaces)
	case "json":
		return printWorkspacesJSON(workspaces)
	default:
		return errors.Errorf("unsupported format: %s", format)
	}
}

func printReposTable(repos []wsm.Repository) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	fmt.Fprintln(w, "NAME\tPATH\tBRANCH\tTAGS\tREMOTE")
	fmt.Fprintln(w, "----\t----\t------\t----\t------")

	for _, repo := range repos {
		tags := strings.Join(repo.Categories, ",")
		if len(tags) > 30 {
			tags = tags[:27] + "..."
		}

		remote := repo.RemoteURL
		if len(remote) > 50 {
			remote = "..." + remote[len(remote)-47:]
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			repo.Name,
			repo.Path,
			repo.CurrentBranch,
			tags,
			remote,
		)
	}

	return nil
}

func printReposJSON(repos []wsm.Repository) error {
	return wsm.PrintJSON(repos)
}

func printWorkspacesTable(workspaces []wsm.Workspace) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

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

func printWorkspacesJSON(workspaces []wsm.Workspace) error {
	return wsm.PrintJSON(workspaces)
}
