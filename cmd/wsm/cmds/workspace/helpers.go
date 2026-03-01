package workspace

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/go-go-golems/workspace-manager/pkg/output"
	"github.com/go-go-golems/workspace-manager/pkg/wsm"
	"github.com/pkg/errors"
)

func isUserCancelledError(err error) bool {
	if err == nil {
		return false
	}
	errMsg := strings.ToLower(err.Error())
	return strings.Contains(errMsg, "cancelled by user") ||
		strings.Contains(errMsg, "creation cancelled") ||
		strings.Contains(errMsg, "operation cancelled") ||
		strings.Contains(errMsg, "user aborted") ||
		strings.Contains(errMsg, "cancelled") ||
		strings.Contains(errMsg, "aborted") ||
		strings.Contains(errMsg, "interrupt")
}

func selectRepositoriesInteractively(wm *wsm.WorkspaceManager) ([]string, error) {
	repos := wm.Discoverer.GetRepositories()
	if len(repos) == 0 {
		return nil, errors.New("no repositories found. Run 'wsm discover' first")
	}

	output.PrintHeader("Select Repositories")

	options := make([]huh.Option[string], 0, len(repos))
	for _, repo := range repos {
		label := fmt.Sprintf("%s (%s)", repo.Name, strings.Join(repo.Categories, ", "))
		options = append(options, huh.NewOption(label, repo.Name))
	}

	var selected []string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Choose repositories to include:").
				Options(options...).
				Value(&selected),
		),
	)

	if err := form.Run(); err != nil {
		if isUserCancelledError(err) {
			return nil, errors.New("workspace creation cancelled by user")
		}
		return nil, errors.Wrap(err, "interactive form failed")
	}

	if len(selected) == 0 {
		return nil, errors.New("no repositories selected")
	}

	output.PrintInfo("Selected %d repositories: %s", len(selected), strings.Join(selected, ", "))
	return selected, nil
}

func showWorkspacePreview(workspace *wsm.Workspace) error {
	output.PrintHeader("Workspace Preview: %s", workspace.Name)
	fmt.Println()

	output.PrintInfo("Actions to be performed:")
	fmt.Printf("  1. Create directory structure at: %s\n", workspace.Path)
	fmt.Printf("  2. Create worktrees:\n")

	for _, repo := range workspace.Repositories {
		if workspace.Branch != "" {
			fmt.Printf("     git worktree add -B %s %s/%s\n", workspace.Branch, workspace.Path, repo.Name)
		} else {
			fmt.Printf("     git worktree add %s/%s\n", workspace.Path, repo.Name)
		}
	}

	stepNum := 3
	if workspace.GoWorkspace {
		fmt.Printf("  %d. Initialize go.work and add modules\n", stepNum)
		stepNum++
	}

	if workspace.AgentMD != "" {
		fmt.Printf("  %d. Copy AGENT.md from %s\n", stepNum, workspace.AgentMD)
		stepNum++
	}

	wm, err := wsm.NewWorkspaceManager()
	if err != nil {
		return errors.Wrap(err, "failed to create workspace manager for preview")
	}

	if err := wm.PreviewSetupScripts(workspace, stepNum); err != nil {
		return errors.Wrap(err, "failed to preview setup scripts")
	}

	fmt.Println()
	output.PrintInfo("Repositories to include:")
	for _, repo := range workspace.Repositories {
		fmt.Printf("  - %s (%s) [%s]\n", repo.Name, repo.Path, strings.Join(repo.Categories, ", "))
	}

	return nil
}

func getRepositoryNames(repos []wsm.Repository) []string {
	names := make([]string, len(repos))
	for i, repo := range repos {
		names[i] = repo.Name
	}
	return names
}
