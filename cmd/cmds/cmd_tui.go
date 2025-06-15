package cmds

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func NewTUICommand() *cobra.Command {
	var workspace string

	cmd := &cobra.Command{
		Use:   "tui",
		Short: "Launch interactive TUI",
		Long: `Launch the Terminal User Interface for visual workspace management.
This provides an interactive way to browse repositories, create workspaces,
and manage your development environment.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTUI(workspace)
		},
	}

	cmd.Flags().StringVar(&workspace, "workspace", "", "Start with specific workspace selected")

	return cmd
}

func runTUI(workspace string) error {
	// Create main model
	model, err := newMainModel()
	if err != nil {
		return errors.Wrap(err, "failed to initialize TUI")
	}

	// If specific workspace requested, navigate to it
	if workspace != "" {
		model.state = stateWorkspaces
		// TODO: Select specific workspace in list
	}

	// Create tea program
	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	// Run the program
	finalModel, err := p.Run()
	if err != nil {
		return errors.Wrap(err, "TUI program failed")
	}

	// Handle any final state
	if m, ok := finalModel.(mainModel); ok && m.message != "" {
		fmt.Println(m.message)
	}

	return nil
}
