# Workspace Manager Implementation Guide

This document provides comprehensive documentation for the workspace manager implementation, from architecture to detailed implementation details. It serves as a reference for new developers and interns to understand how workspace management works in this project.

## Table of Contents

1. [Overview](#overview)
2. [Architecture](#architecture)
3. [Core Components](#core-components)
4. [Data Models](#data-models)
5. [Command Implementation](#command-implementation)
6. [Git Operations](#git-operations)
7. [Workspace Lifecycle](#workspace-lifecycle)
8. [Configuration Management](#configuration-management)
9. [Error Handling](#error-handling)
10. [Development Guide](#development-guide)

## Overview

The Workspace Manager is a command-line tool designed to manage multi-repository workspaces using git worktrees. It allows developers to work on related repositories simultaneously while maintaining consistency across branches and managing dependencies.

### Key Features

- **Repository Discovery**: Automatically discovers and catalogs git repositories
- **Workspace Creation**: Creates workspaces with synchronized branches across multiple repositories
- **Git Worktree Management**: Uses git worktrees to avoid repository cloning overhead
- **Go Workspace Integration**: Automatically creates `go.work` files for Go projects
- **Status Tracking**: Monitors git status across all repositories in a workspace (logic in `pkg/wsm/status.go`)
- **Interactive TUI**: Provides a terminal-based user interface for easier management (located in `cmd/cmds/tui.go`)

## Architecture

The workspace manager follows a modular architecture with clear separation of concerns:

```
workspace-manager/
├── cmd/
│   ├── workspace-manager/
│   │   └── main.go           # Entry point
│   ├── cmds/                 # Command implementations
│   │   ├── cmd_*.go          # Individual command implementations
│   │   ├── tui.go           # Terminal UI implementation
│   │   └── tui_models.go    # TUI models
│   └── root.go              # Root command and CLI setup
├── pkg/
│   └── wsm/                 # Core workspace management package
│       ├── types.go         # Data structures and types
│       ├── workspace.go     # Core workspace management logic
│       ├── discovery.go     # Repository discovery logic
│       ├── status.go        # Status checking operations
│       ├── git_operations.go # Git operation utilities
│       ├── git_utils.go     # Additional git utilities
│       ├── sync_operations.go # Synchronization operations
│       └── utils.go         # General utilities
```

### Design Principles

1. **Single Responsibility**: Each component has a specific, well-defined purpose
2. **Dependency Injection**: Core components are injected to enable testing
3. **Error Propagation**: Errors are wrapped with context for better debugging
4. **Logging**: Structured logging using zerolog for observability
5. **CLI Conventions**: Uses Cobra for consistent command-line interface
6. **Package Separation**: Clear separation between CLI commands (`cmd/`) and business logic (`pkg/wsm/`)

### Architecture Benefits

The reorganized structure provides several advantages:
- **Reusability**: Core workspace management logic in `pkg/wsm/` can be imported by other tools
- **Testing**: Business logic can be unit tested independently from CLI commands
- **Maintainability**: Clear boundaries between user interface and core functionality
- **Go Conventions**: Follows standard Go project layout with `cmd/` for binaries and `pkg/` for libraries

## Core Components

All core workspace management functionality is now organized in the `pkg/wsm` package, providing clean separation between command-line interface (in `cmd/`) and business logic. Commands import the workspace manager functionality using:

```go
import "github.com/go-go-golems/workspace-manager/pkg/wsm"
```

### WorkspaceManager

The `WorkspaceManager` struct is the central component that orchestrates workspace operations:

```go
type WorkspaceManager struct {
    config       *WorkspaceConfig
    discoverer   *RepositoryDiscoverer
    workspaceDir string
}
```

**Key Methods:**
- `CreateWorkspace()`: Creates new multi-repository workspaces
- `AddRepositoryToWorkspace()`: Adds repositories to existing workspaces
- `RemoveRepositoryFromWorkspace()`: Removes repositories from workspaces
- `DeleteWorkspace()`: Deletes entire workspaces
- `LoadWorkspace()`: Loads workspace configurations

### RepositoryDiscoverer

Handles repository discovery and registry management:

```go
type RepositoryDiscoverer struct {
    registryPath string
    registry     *RepositoryRegistry
}
```

**Key Features:**
- Scans directories for git repositories
- Extracts metadata (branches, tags, remote URLs)
- Categorizes repositories (Go, Node.js, etc.)
- Maintains persistent registry

### Git Operations Layer

Provides abstraction over git commands with proper error handling and logging:

**Key Functions:**
- `executeWorktreeCommand()`: Executes git worktree operations
- `checkBranchExists()`: Verifies local branch existence
- `checkRemoteBranchExists()`: Verifies remote branch existence
- `createWorktree()`: Creates git worktrees with branch management

## Data Models

### Repository

Represents a discovered git repository:

```go
type Repository struct {
    Name          string    `json:"name"`
    Path          string    `json:"path"`
    RemoteURL     string    `json:"remote_url"`
    CurrentBranch string    `json:"current_branch"`
    Branches      []string  `json:"branches"`
    Tags          []string  `json:"tags"`
    LastCommit    string    `json:"last_commit"`
    LastUpdated   time.Time `json:"last_updated"`
    Categories    []string  `json:"categories"`
}
```

### Workspace

Represents a multi-repository workspace:

```go
type Workspace struct {
    Name         string       `json:"name"`
    Path         string       `json:"path"`
    Repositories []Repository `json:"repositories"`
    Branch       string       `json:"branch"`
    Created      time.Time    `json:"created"`
    GoWorkspace  bool         `json:"go_workspace"`
    AgentMD      string       `json:"agent_md"`
}
```

### WorkspaceConfig

Configuration for workspace management:

```go
type WorkspaceConfig struct {
    WorkspaceDir string `json:"workspace_dir"`
    TemplateDir  string `json:"template_dir"`
    RegistryPath string `json:"registry_path"`
}
```

## Command Implementation

### Command Structure

All commands follow a consistent pattern using Cobra:

1. **Command Definition**: Define command metadata and flags
2. **Argument Validation**: Validate required arguments
3. **Manager Initialization**: Create `WorkspaceManager` instance
4. **Operation Execution**: Call appropriate manager method
5. **Error Handling**: Wrap and return errors with context

### Example Command Implementation

Here's the structure of the `add` command in [`cmd_add.go`](file:///home/manuel/code/wesen/corporate-headquarters/workspace-manager/cmd/cmds/cmd_add.go):

```go
func NewAddCommand() *cobra.Command {
    var branchName string
    var forceOverwrite bool

    cmd := &cobra.Command{
        Use:   "add <workspace-name> <repo-name>",
        Short: "Add a repository to an existing workspace",
        // ... Long description and examples
        Args: cobra.ExactArgs(2),
        RunE: func(cmd *cobra.Command, args []string) error {
            workspaceName := args[0]
            repoName := args[1]

            wm, err := wsm.NewWorkspaceManager()
            if err != nil {
                return errors.Wrap(err, "failed to create workspace manager")
            }

            return wm.AddRepositoryToWorkspace(cmd.Context(), workspaceName, repoName, branchName, forceOverwrite)
        },
    }

    cmd.Flags().StringVarP(&branchName, "branch", "b", "", "Branch name to use")
    cmd.Flags().BoolVarP(&forceOverwrite, "force", "f", false, "Force overwrite if branch already exists")

    return cmd
}
```

### Remove Command Implementation

The newly implemented `remove` command in [`cmd_remove.go`](file:///home/manuel/code/wesen/corporate-headquarters/workspace-manager/cmd/cmds/cmd_remove.go) follows the same pattern:

```go
func (wm *WorkspaceManager) RemoveRepositoryFromWorkspace(ctx context.Context, workspaceName, repoName string, force, removeFiles bool) error {
    // 1. Load workspace configuration
    workspace, err := wm.LoadWorkspace(workspaceName)
    if err != nil {
        return errors.Wrapf(err, "failed to load workspace '%s'", workspaceName)
    }

    // 2. Find target repository in workspace
    var repoIndex = -1
    var targetRepo Repository
    for i, repo := range workspace.Repositories {
        if repo.Name == repoName {
            repoIndex = i
            targetRepo = repo
            break
        }
    }

    // 3. Remove git worktree
    worktreePath := filepath.Join(workspace.Path, repoName)
    if err := wm.removeWorktreeForRepo(ctx, targetRepo, worktreePath, force); err != nil {
        return errors.Wrapf(err, "failed to remove worktree for repository '%s'", repoName)
    }

    // 4. Update workspace configuration
    workspace.Repositories = append(workspace.Repositories[:repoIndex], workspace.Repositories[repoIndex+1:]...)

    // 5. Update go.work file if needed
    if workspace.GoWorkspace {
        if err := wm.createGoWorkspace(workspace); err != nil {
            log.Warn().Err(err).Msg("Failed to update go.work file, but continuing")
        }
    }

    // 6. Save updated configuration
    return wm.saveWorkspace(workspace)
}
```

## Git Operations

### Worktree Management

Git worktrees are the core technology enabling efficient multi-repository workflows. The workspace manager implements robust worktree management:

#### Creating Worktrees

The `createWorktree()` function in [`workspace.go`](file:///home/manuel/code/wesen/corporate-headquarters/workspace-manager/pkg/wsm/workspace.go) handles various scenarios:

1. **No Branch Specified**: Creates worktree from current branch
2. **Branch Exists Locally**: Prompts user to overwrite or use existing
3. **Branch Exists Remotely**: Creates local branch tracking remote
4. **New Branch**: Creates new branch and worktree

```go
if branchExists {
    // Branch exists locally - ask user what to do
    fmt.Printf("⚠️  Branch '%s' already exists in repository '%s'\n", workspace.Branch, repo.Name)
    fmt.Printf("What would you like to do?\n")
    fmt.Printf("  [o] Overwrite the existing branch (git worktree add -B)\n")
    fmt.Printf("  [u] Use the existing branch as-is (git worktree add)\n")
    fmt.Printf("  [c] Cancel workspace creation\n")
    // ... handle user choice
}
```

#### Removing Worktrees

The `removeWorktreeForRepo()` function provides safe worktree removal:

```go
func (wm *WorkspaceManager) removeWorktreeForRepo(ctx context.Context, repo Repository, worktreePath string, force bool) error {
    // Check if worktree exists
    if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
        return nil // Already removed
    }

    // Execute git worktree remove command
    var cmd *exec.Cmd
    if force {
        cmd = exec.CommandContext(ctx, "git", "worktree", "remove", "--force", worktreePath)
    } else {
        cmd = exec.CommandContext(ctx, "git", "worktree", "remove", worktreePath)
    }
    
    return wm.executeWorktreeCommand(ctx, repo.Path, cmd.Args...)
}
```

### Branch Operations

Branch management includes checking existence and creating tracking branches:

```go
// Check local branch existence
func (wm *WorkspaceManager) checkBranchExists(ctx context.Context, repoPath, branch string) (bool, error) {
    cmd := exec.CommandContext(ctx, "git", "show-ref", "--verify", "--quiet", "refs/heads/"+branch)
    cmd.Dir = repoPath
    err := cmd.Run()
    return err == nil, nil
}

// Check remote branch existence
func (wm *WorkspaceManager) checkRemoteBranchExists(ctx context.Context, repoPath, branch string) (bool, error) {
    cmd := exec.CommandContext(ctx, "git", "show-ref", "--verify", "--quiet", "refs/remotes/origin/"+branch)
    cmd.Dir = repoPath
    err := cmd.Run()
    return err == nil, nil
}
```

## Workspace Lifecycle

### Creation Process

1. **Repository Discovery**: Find requested repositories in registry
2. **Validation**: Ensure all repositories exist and are accessible
3. **Directory Creation**: Create workspace directory structure
4. **Worktree Creation**: Create git worktrees for each repository
5. **Go Workspace Setup**: Create `go.work` file if applicable
6. **Agent Configuration**: Copy `AGENT.md` file if specified
7. **Configuration Persistence**: Save workspace metadata

### Adding Repositories

1. **Workspace Loading**: Load existing workspace configuration
2. **Repository Lookup**: Find repository in registry
3. **Duplication Check**: Ensure repository not already in workspace
4. **Worktree Creation**: Create worktree for new repository
5. **Configuration Update**: Add repository to workspace and save
6. **Go Workspace Update**: Update `go.work` file if needed

### Removing Repositories

1. **Workspace Loading**: Load existing workspace configuration
2. **Repository Location**: Find repository in workspace
3. **Worktree Removal**: Remove git worktree
4. **Directory Cleanup**: Optionally remove repository directory
5. **Configuration Update**: Remove repository from workspace and save
6. **Go Workspace Update**: Update `go.work` file if needed

### Deletion Process

1. **Workspace Loading**: Load workspace configuration
2. **Worktree Cleanup**: Remove all git worktrees
3. **File Removal**: Optionally remove workspace directory
4. **Configuration Cleanup**: Remove workspace configuration file

## Configuration Management

### Storage Locations

Configuration files are stored in standard locations:

- **Registry**: `~/.config/workspace-manager/registry.json`
- **Workspaces**: `~/.config/workspace-manager/workspaces/`
- **Default Workspace Directory**: `~/workspaces/YYYY-MM-DD/`

### Configuration Loading

The `loadConfig()` function in [`workspace.go`](file:///home/manuel/code/wesen/corporate-headquarters/workspace-manager/pkg/wsm/workspace.go) establishes default configuration:

```go
func loadConfig() (*WorkspaceConfig, error) {
    home, err := os.UserHomeDir()
    if err != nil {
        return nil, err
    }

    configDir, err := os.UserConfigDir()
    if err != nil {
        return nil, err
    }

    config := &WorkspaceConfig{
        WorkspaceDir: filepath.Join(home, "workspaces", time.Now().Format("2006-01-02")),
        TemplateDir:  filepath.Join(home, "templates"),
        RegistryPath: filepath.Join(configDir, "workspace-manager", "registry.json"),
    }

    return config, nil
}
```

### Persistence

Workspace configurations are persisted as JSON files using the `saveWorkspace()` function:

```go
func (wm *WorkspaceManager) saveWorkspace(workspace *Workspace) error {
    workspacesDir := filepath.Join(filepath.Dir(wm.config.RegistryPath), "workspaces")
    if err := os.MkdirAll(workspacesDir, 0755); err != nil {
        return errors.Wrap(err, "failed to create workspaces directory")
    }

    configPath := filepath.Join(workspacesDir, workspace.Name+".json")
    
    data, err := json.MarshalIndent(workspace, "", "  ")
    if err != nil {
        return errors.Wrap(err, "failed to marshal workspace configuration")
    }

    return os.WriteFile(configPath, data, 0644)
}
```

## Error Handling

### Error Wrapping Strategy

The workspace manager uses the `github.com/pkg/errors` package for error wrapping:

```go
if err := wm.createWorktree(ctx, workspace, repo); err != nil {
    // Rollback any worktrees created so far
    wm.rollbackWorktrees(ctx, createdWorktrees)
    wm.cleanupWorkspaceDirectory(workspace.Path)
    return errors.Wrapf(err, "failed to create worktree for %s", repo.Name)
}
```

### Rollback Mechanisms

When operations fail, the workspace manager implements rollback mechanisms:

#### Worktree Rollback

The `rollbackWorktrees()` function in [`workspace.go`](file:///home/manuel/code/wesen/corporate-headquarters/workspace-manager/pkg/wsm/workspace.go) cleans up partially created workspaces:

```go
func (wm *WorkspaceManager) rollbackWorktrees(ctx context.Context, worktrees []WorktreeInfo) {
    for i := len(worktrees) - 1; i >= 0; i-- {
        worktree := worktrees[i]
        
        // Use --force to ensure removal even with uncommitted changes
        cmd := exec.CommandContext(ctx, "git", "worktree", "remove", "--force", worktree.TargetPath)
        cmd.Dir = worktree.Repository.Path
        
        if output, err := cmd.CombinedOutput(); err != nil {
            log.Warn().Err(err).Str("output", string(output)).Msg("Failed to remove worktree during rollback")
        }
    }
}
```

### Logging Strategy

Structured logging using zerolog provides detailed operation tracking:

```go
log.Info().
    Str("workspace", workspaceName).
    Str("repo", repoName).
    Bool("force", force).
    Bool("removeFiles", removeFiles).
    Msg("Removing repository from workspace")
```

## Development Guide

### Setting Up Development Environment

1. **Clone Repository**: `git clone <repository-url>`
2. **Install Dependencies**: `go mod download`
3. **Build Tool**: `go build ./cmd/workspace-manager`
4. **Run Tests**: `go test ./...`

### Adding New Commands

To add a new command:

1. **Create Command File**: `cmd/cmds/cmd_<name>.go`
2. **Implement Command Function**: `func New<Name>Command() *cobra.Command`
3. **Add to Root Command**: Add to `rootCmd.AddCommand()` in `cmd/root.go`
4. **Implement Business Logic**: Add methods to `WorkspaceManager` in `pkg/wsm/` if needed

### Testing Strategy

The codebase uses Go's built-in testing framework:

```go
func TestWorkspaceCreation(t *testing.T) {
    wm, err := wsm.NewWorkspaceManager()
    require.NoError(t, err)
    
    workspace, err := wm.CreateWorkspace(context.Background(), "test-workspace", []string{"repo1"}, "main", "", true)
    require.NoError(t, err)
    assert.Equal(t, "test-workspace", workspace.Name)
}
```

### Code Style Guidelines

1. **Error Handling**: Always wrap errors with context
2. **Logging**: Use structured logging for debugging
3. **Documentation**: Document exported functions and types
4. **Validation**: Validate inputs early in functions
5. **Context Propagation**: Pass context through operation chains

### Debugging Tips

1. **Enable Debug Logging**: Set log level to debug for detailed operation tracking
2. **Git Worktree List**: Use `git worktree list` to inspect worktree state
3. **Configuration Inspection**: Check `~/.config/workspace-manager/` for configuration issues
4. **Manual Cleanup**: Use `git worktree remove --force` for stuck worktrees

### Common Pitfalls

1. **Worktree Path Conflicts**: Ensure unique paths for each worktree
2. **Branch State**: Check both local and remote branch existence
3. **Permission Issues**: Ensure proper file permissions for configuration directories
4. **Git Repository State**: Verify repositories are clean before operations

This implementation guide provides the foundation for understanding and extending the workspace manager. For specific implementation details, refer to the source code files mentioned throughout this document.
