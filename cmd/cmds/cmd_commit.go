package cmds

import (
	"context"
	"fmt"
	"github.com/go-go-golems/workspace-manager/pkg/wsm"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func NewCommitCommand() *cobra.Command {
	var (
		message     string
		interactive bool
		addAll      bool
		push        bool
		dryRun      bool
		template    string
	)

	cmd := &cobra.Command{
		Use:   "commit",
		Short: "Commit changes across workspace repositories",
		Long: `Commit related changes across multiple repositories in the workspace.
Supports interactive file selection and consistent commit messaging.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommit(cmd.Context(), message, interactive, addAll, push, dryRun, template)
		},
	}

	cmd.Flags().StringVarP(&message, "message", "m", "", "Commit message")
	cmd.Flags().BoolVar(&interactive, "interactive", false, "Interactive file selection")
	cmd.Flags().BoolVar(&addAll, "add-all", false, "Add all changes")
	cmd.Flags().BoolVar(&push, "push", false, "Push changes after commit")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be committed")
	cmd.Flags().StringVar(&template, "template", "", "Use commit message template")

	return cmd
}

func runCommit(ctx context.Context, message string, interactive, addAll, push, dryRun bool, template string) error {
	// Detect current workspace
	workspace, err := detectCurrentWorkspace()
	if err != nil {
		return errors.Wrap(err, "failed to detect current workspace")
	}

	// Initialize git operations
	gitOps := wsm.NewGitOperations(workspace)

	// Get all changes in workspace
	allChanges, err := gitOps.GetWorkspaceChanges(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to get workspace changes")
	}

	if len(allChanges) == 0 {
		fmt.Println("No changes found in workspace.")
		return nil
	}

	// Handle commit message
	if message == "" && template != "" {
		message = getCommitMessageFromTemplate(template)
	}

	if message == "" && !interactive {
		return errors.New("commit message is required. Use -m flag or --interactive mode")
	}

	// Handle interactive mode
	var selectedChanges map[string][]wsm.FileChange
	if interactive {
		selectedChanges, message, err = selectChangesInteractively(allChanges, message)
		if err != nil {
			return errors.Wrap(err, "interactive selection failed")
		}
	} else {
		selectedChanges = allChanges
	}

	if len(selectedChanges) == 0 {
		fmt.Println("No files selected for commit.")
		return nil
	}

	// Create commit operation
	operation := &wsm.CommitOperation{
		Message: message,
		Files:   selectedChanges,
		DryRun:  dryRun,
		AddAll:  addAll,
		Push:    push,
	}

	// Execute commit
	if err := gitOps.CommitChanges(ctx, operation); err != nil {
		return errors.Wrap(err, "commit failed")
	}

	if !dryRun {
		fmt.Printf("✅ Successfully committed changes across %d repositories\n", len(selectedChanges))
		if push {
			fmt.Println("📤 Changes pushed to remote repositories")
		}
	}

	return nil
}

// detectCurrentWorkspace detects the current workspace
func detectCurrentWorkspace() (*wsm.Workspace, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get current directory")
	}

	// Try to find workspace by checking if we're in a workspace directory
	workspaces, err := wsm.LoadWorkspaces()
	if err != nil {
		return nil, errors.Wrap(err, "failed to load workspaces")
	}

	for _, workspace := range workspaces {
		if strings.HasPrefix(cwd, workspace.Path) {
			return &workspace, nil
		}
	}

	return nil, errors.New("not in a workspace directory. Run command from within a workspace")
}

// selectChangesInteractively allows user to select files interactively
func selectChangesInteractively(allChanges map[string][]wsm.FileChange, initialMessage string) (map[string][]wsm.FileChange, string, error) {
	fmt.Println("=== Interactive Commit ===")
	fmt.Println()

	// Show all changes
	fmt.Println("Changes found:")
	repoIndex := 0
	repoNames := make([]string, 0, len(allChanges))

	for repoName, changes := range allChanges {
		repoNames = append(repoNames, repoName)
		fmt.Printf("\n%d. Repository: %s (%d files)\n", repoIndex+1, repoName, len(changes))

		for i, change := range changes {
			status := wsm.GetStatusSymbol(change.Status)
			staged := ""
			if change.Staged {
				staged = " (staged)"
			}
			fmt.Printf("   %c. %s %s%s\n", 'a'+i, status, change.FilePath, staged)
		}
		repoIndex++
	}

	fmt.Println()

	// Get commit message if not provided
	message := initialMessage
	if message == "" {
		fmt.Print("Commit message: ")
		fmt.Scanln(&message)
		if message == "" {
			return nil, "", errors.New("commit message is required")
		}
	}

	// Simple selection - for now, include all changes
	// TODO: Implement more sophisticated interactive selection
	fmt.Println("\nProceeding with all changes...")

	return allChanges, message, nil
}

// getCommitMessageFromTemplate gets commit message from template
func getCommitMessageFromTemplate(template string) string {
	templates := map[string]string{
		"feature":  "feat: add new feature",
		"fix":      "fix: resolve issue",
		"docs":     "docs: update documentation",
		"style":    "style: formatting changes",
		"refactor": "refactor: code restructuring",
		"test":     "test: add or update tests",
		"chore":    "chore: maintenance tasks",
	}

	if msg, exists := templates[template]; exists {
		return msg
	}

	return template // Use template as-is if not found in predefined templates
}
