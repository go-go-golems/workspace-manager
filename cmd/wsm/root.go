package main

import (
	"github.com/go-go-golems/glazed/pkg/cmds/logging"
	gitcmds "github.com/go-go-golems/workspace-manager/cmd/wsm/cmds/git"
	registrycmds "github.com/go-go-golems/workspace-manager/cmd/wsm/cmds/registry"
	workspacecmds "github.com/go-go-golems/workspace-manager/cmd/wsm/cmds/workspace"
	"github.com/go-go-golems/workspace-manager/pkg/output"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/carapace-sh/carapace"
	clay "github.com/go-go-golems/clay/pkg"
)

var rootCmd = &cobra.Command{
	Use:   "wsm",
	Short: "A tool for managing multi-repository workspaces",
	Long: `Workspace Manager helps you work with multiple related git repositories 
simultaneously by automating workspace setup, git operations, and status tracking.

Features:
- Discover and catalog git repositories across your development environment
- Create workspaces with git worktrees for coordinated multi-repo development
- Track status across all repositories in a workspace
- Commit changes across multiple repositories with consistent messaging
- Coordinate repository branches and rebases across a workspace

- Safe workspace cleanup with proper worktree removal

Examples:
  # Discover repositories in your code directories
  wsm discover ~/code ~/projects --recursive

  # Create a workspace for feature development
  wsm create my-feature --repos app,lib,shared --branch feature/new-api

  # Check status across all workspace repositories
  wsm status

  # Interactive mode
  `,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return logging.InitLoggerFromCobra(cmd)
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	err := clay.InitGlazed("workspace-manager", rootCmd)
	if err != nil {
		output.PrintError("Failed to initialize configuration: %v", err)
		log.Fatal().Err(err).Msg("Failed to initialize Glazed")
	}

	if err := registrycmds.Register(rootCmd); err != nil {
		output.PrintError("Failed to register registry commands: %v", err)
		log.Fatal().Err(err).Msg("Failed to register registry commands")
	}

	if err := workspacecmds.Register(rootCmd); err != nil {
		output.PrintError("Failed to register workspace commands: %v", err)
		log.Fatal().Err(err).Msg("Failed to register workspace commands")
	}

	if err := gitcmds.Register(rootCmd); err != nil {
		output.PrintError("Failed to register git commands: %v", err)
		log.Fatal().Err(err).Msg("Failed to register git commands")
	}

	carapace.Gen(rootCmd)
}
