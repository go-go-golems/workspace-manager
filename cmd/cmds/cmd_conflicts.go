package cmds

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/go-go-golems/workspace-manager/pkg/wsm"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
)

func NewConflictsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "conflicts",
		Short: "Conflict management helpers",
	}
	cmd.AddCommand(NewConflictsListCommand())
	cmd.AddCommand(NewConflictsOpenCommand())
	cmd.AddCommand(NewConflictsMarkResolvedCommand())
	return cmd
}

func NewConflictsListCommand() *cobra.Command {
	var (
		repository string
		jobs       int
	)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List conflicted files across repositories",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConflictsList(cmd.Context(), repository, jobs)
		},
	}
	cmd.Flags().StringVar(&repository, "repo", "", "Only list conflicts for a specific repository")
	cmd.Flags().IntVar(&jobs, "jobs", 1, "Maximum concurrent repositories to process")
	return cmd
}

func runConflictsList(ctx context.Context, repository string, jobs int) error {
	workspace, err := detectCurrentWorkspace()
	if err != nil { return errors.Wrap(err, "failed to detect current workspace") }

	repos := workspace.Repositories
	if repository != "" {
		filtered := []wsm.Repository{}
		for _, r := range repos { if r.Name == repository { filtered = append(filtered, r) } }
		repos = filtered
	}

	type row struct { repo string; count int; err string }
	rows := make([]row, len(repos))
	do := func(i int) {
		r := repos[i]
		conflicts, err := wsm.ListConflicts(ctx, filepath.Join(workspace.Path, r.Name))
		errStr := ""
		if err != nil { errStr = err.Error() }
		rows[i] = row{repo: r.Name, count: len(conflicts), err: errStr}
	}

	if jobs <= 1 || len(repos) <= 1 {
		for i := range repos { do(i) }
	} else {
		sem := semaphore.NewWeighted(int64(jobs))
		g, gctx := errgroup.WithContext(ctx)
		for i := range repos {
			i := i
			if err := sem.Acquire(gctx, 1); err != nil { return err }
			g.Go(func() error { defer sem.Release(1); do(i); return nil })
		}
		if err := g.Wait(); err != nil { return err }
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()
	fmt.Fprintln(w, "REPOSITORY\tCONFLICTS\tERROR")
	fmt.Fprintln(w, "----------\t---------\t-----")
	for _, r := range rows {
		errStr := r.err
		if len(errStr) > 60 { errStr = errStr[:57] + "..." }
		fmt.Fprintf(w, "%s\t%d\t%s\n", r.repo, r.count, errStr)
	}
	return nil
}

func NewConflictsOpenCommand() *cobra.Command {
	var (
		repository string
		tool       string
	)
	cmd := &cobra.Command{
		Use:   "open",
		Short: "Open mergetool/editor for conflicted files",
		RunE: func(cmd *cobra.Command, args []string) error {
			workspace, err := detectCurrentWorkspace()
			if err != nil { return errors.Wrap(err, "failed to detect current workspace") }
			if repository == "" { return errors.New("--repo is required") }
			repoPath := filepath.Join(workspace.Path, repository)
			return wsm.OpenMergetool(cmd.Context(), repoPath, tool, nil)
		},
	}
	cmd.Flags().StringVar(&repository, "repo", "", "Repository name")
	cmd.Flags().StringVar(&tool, "tool", "", "Mergetool to use (e.g., vimdiff, meld)")
	return cmd
}

func NewConflictsMarkResolvedCommand() *cobra.Command {
	var (
		repository string
		all       bool
	)
	cmd := &cobra.Command{
		Use:   "mark-resolved",
		Short: "Stage resolved files (or all if --all)",
		RunE: func(cmd *cobra.Command, args []string) error {
			workspace, err := detectCurrentWorkspace()
			if err != nil { return errors.Wrap(err, "failed to detect current workspace") }
			if repository == "" { return errors.New("--repo is required") }
			repoPath := filepath.Join(workspace.Path, repository)
			return wsm.StageResolved(cmd.Context(), repoPath, all, args)
		},
	}
	cmd.Flags().StringVar(&repository, "repo", "", "Repository name")
	cmd.Flags().BoolVar(&all, "all", false, "Stage all changes")
	return cmd
}
