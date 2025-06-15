package cmd

import (
	"github.com/go-go-golems/glazed/pkg/cmds/logging"
	"github.com/go-go-golems/workspace-manager/cmd/cmds"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	clay "github.com/go-go-golems/clay/pkg"
)

var rootCmd = &cobra.Command{
	Use:   "workspace-manager",
	Short: "A tool for managing multi-repository workspaces",
	Long: `Workspace Manager helps you work with multiple related git repositories 
simultaneously by automating workspace setup, git operations, and status tracking.

Features:
- Discover and catalog git repositories across your development environment
- Create workspaces with git worktrees for coordinated multi-repo development
- Track status across all repositories in a workspace
- Commit changes across multiple repositories with consistent messaging
- Synchronize repositories (pull, push, branch operations)
- Interactive TUI for visual repository and workspace management
- Safe workspace cleanup with proper worktree removal

Examples:
  # Discover repositories in your code directories
  workspace-manager discover ~/code ~/projects --recursive

  # Create a workspace for feature development
  workspace-manager create my-feature --repos app,lib,shared --branch feature/new-api

  # Check status across all workspace repositories
  workspace-manager status

  # Interactive mode
  workspace-manager tui`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return logging.InitLoggerFromViper()
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	err := clay.InitViper("workspace-manager", rootCmd)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize Viper")
	}

	// Add all subcommands
	rootCmd.AddCommand(
		cmds.NewDiscoverCommand(),
		cmds.NewListCommand(),
		cmds.NewCreateCommand(),
		cmds.NewAddCommand(),
		cmds.NewRemoveCommand(),
		cmds.NewDeleteCommand(),
		cmds.NewInfoCommand(),
		cmds.NewStatusCommand(),
		cmds.NewPRCommand(),
		cmds.NewPushCommand(),
		cmds.NewTUICommand(),
		cmds.NewCommitCommand(),
		cmds.NewSyncCommand(),
		cmds.NewBranchCommand(),
		cmds.NewRebaseCommand(),
		cmds.NewDiffCommand(),
		cmds.NewLogCommand(),
	)
}
