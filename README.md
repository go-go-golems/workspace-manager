# WSM (Workspace Manager)

**A powerful CLI tool for managing multi-repository workspaces with git worktrees**

WSM simplifies coordinated development across multiple related git repositories by automating workspace setup, git operations, and status tracking. Perfect for microservices, monorepos, or any project spanning multiple repositories.

## Features

- **🔍 Repository Discovery**: Automatically discover and catalog git repositories across your development environment
- **🏗️ Workspace Creation**: Create coordinated workspaces with git worktrees for multi-repo development
- **🔀 Fork & Merge Workflow**: Fork existing workspaces for feature development and merge back to parent branches
- **📊 Status Tracking**: Monitor git status across all repositories in a workspace simultaneously
- **🔄 Synchronized Operations**: Commit, push, and sync changes across multiple repositories with consistent messaging
- **🌿 Branch Management**: Coordinate branch operations across all workspace repositories
- **🔧 Go Integration**: Automatic `go.work` file generation for Go projects
- **🧹 Safe Cleanup**: Proper worktree removal and workspace cleanup
- **💻 Tmux Integration**: Create and manage tmux sessions with profile-based configuration for each workspace
- **⚙️ Setup Scripts**: Automatic execution of setup scripts from `.wsm/setup.sh` and `.wsm/setup.d/` directories
- **📝 Metadata Files**: Automatic creation of `.wsm/wsm.json` with workspace information and environment variables

## Branch Resolution Model

WSM now uses an explicit branch abstraction layer in `pkg/wsm/branch`:

- `BranchService` is the single policy engine for branch decisions.
- Resolution uses typed enums: `ResolutionMode`, `ResolutionStrategy`, and `RemoteRefKind`.
- Git backend APIs are explicit primitives (`ListLocalBranches`, `ListRemoteTrackingBranches`, `LocalBranchExists`, `RemoteTrackingBranchExists`).
- Branch policy no longer relies on ambiguous mixed branch lists or string-only heuristics.

This behavior is intentionally breaking compared to older internal branch helper semantics.

## Installation

### From Releases

Download the latest binary from the [releases page](https://github.com/go-go-golems/workspace-manager/releases):

```bash
# Linux/macOS
curl -L https://github.com/go-go-golems/workspace-manager/releases/latest/download/wsm-$(uname -s)-$(uname -m) -o wsm
chmod +x wsm
sudo mv wsm /usr/local/bin/
```

### Using Go Install

```bash
go install github.com/go-go-golems/workspace-manager/cmd/wsm@latest
```

### Build from Source

```bash
git clone https://github.com/go-go-golems/workspace-manager.git
cd workspace-manager
go build ./cmd/workspace-manager
```

## Shell Completion

WSM supports intelligent shell completion via [carapace](https://github.com/carapace-sh/carapace), providing context-aware suggestions for commands, workspace names, repository names, and tags.

### Setup

**Prerequisites**: Install [carapace](https://github.com/carapace-sh/carapace) for your shell.

**Enable completion:**
```bash
# For permanent setup, add to your shell configuration
source <(wsm _carapace)
```

### Features

- **Workspace Names**: Auto-complete workspace names for `info`, `delete`, `add`, `remove`, etc.
- **Repository Names**: 
  - For `add` command: Complete from all available repositories in registry
  - For `remove` command: Complete from repositories currently in the workspace
- **Tags**: Auto-complete repository tags for the `--tags` flag in `list repos`
- **Dynamic Context**: Completions adapt based on your actual data (workspaces, repositories, tags)

## Quick Start

### 1. Discover Repositories

First, let WSM discover your existing git repositories:

```bash
# Discover repositories in your common development directories
wsm discover ~/code ~/projects --recursive

# Discover with custom depth limit
wsm discover ~/code --max-depth 2
```

### 2. Create a Workspace

Create a workspace with multiple repositories for coordinated development:

```bash
# Create workspace with automatic branch naming (task/my-feature)
wsm create my-feature --repos app,lib,shared

# Create workspace with custom branch
wsm create my-feature --repos app,lib,shared --branch feature/new-api

# Interactive repository selection
wsm create my-feature --interactive
```

### 2a. Fork an Existing Workspace

Create a new workspace by forking an existing one:

```bash
# Fork the current workspace
wsm fork my-feature-branch

# Fork a specific workspace
wsm fork my-feature-branch source-workspace

# Fork with custom branch name
wsm fork my-feature-branch --branch feature/custom-name
```

### 3. Check Status

Monitor the status of all repositories in your workspace:

```bash
wsm status
```

### 4. Work with Your Code

Navigate to your workspace directory (default: `~/workspaces/YYYY-MM-DD/my-feature/`) and start coding. Each repository is available as a git worktree on your specified branch.

### 5. Start a Tmux Session

Create or attach to a tmux session for your workspace:

```bash
# Create/attach to tmux session for current workspace
wsm tmux

# Create/attach to tmux session for specific workspace
wsm tmux my-feature

# Use a specific profile configuration
wsm tmux my-feature --profile development
```

### 6. Merge and Clean Up

When you're done with your work, merge the fork back to its parent branch:

```bash
# Merge current fork back to base branch and delete workspace
wsm merge

# Merge but keep the workspace
wsm merge --keep-workspace

# Preview merge without executing
wsm merge --dry-run
```

### 6. Interactive Mode



## Commands Reference

### Repository Discovery

```bash
# Discover repositories in specified paths
wsm discover [paths...] [flags]

# Examples
wsm discover ~/code ~/projects
wsm discover . --recursive --max-depth 3
```

### Workspace Management

```bash
# Create a new workspace
wsm create <workspace-name> --repos <repo1,repo2,repo3>

# Fork an existing workspace
wsm fork <new-workspace-name> [source-workspace-name]

# Merge fork back to parent branch
wsm merge [workspace-name]

# List all workspaces and repositories
wsm list
wsm list repos [--tags tag1,tag2]
wsm list workspaces

# Get workspace information
wsm info [workspace-name]

# Delete a workspace
wsm delete <workspace-name>
```

### Repository Operations

```bash
# Add repository to existing workspace
wsm add <workspace-name> <repo-name>

# Remove repository from workspace
wsm remove <workspace-name> <repo-name>

# Show workspace status
wsm status [workspace-name]
```

### Tmux Integration

```bash
# Create or attach to tmux session for current workspace
wsm tmux

# Create or attach to tmux session for specific workspace
wsm tmux <workspace-name>

# Use specific profile configuration
wsm tmux [workspace-name] --profile <profile-name>
```

### Git Operations

```bash
# Commit changes across workspace repositories
wsm commit -m "Your commit message"

# Push workspace branches
wsm push [remote]

# Sync repositories (pull latest changes)
wsm sync
wsm sync all
wsm sync pull
wsm sync push

# Show diff across repositories
wsm diff

# Show commit history
wsm log

# Manage branches
wsm branch <operation>
wsm branch create <branch-name>
wsm branch switch <branch-name>
wsm branch list

# Rebase workspace repositories
wsm rebase
```

### Pull Request Management

```bash
# Create pull requests for workspace branches
wsm pr
```

## Configuration

WSM uses a configuration directory at `~/.config/workspace-manager/`:

- **Registry**: `registry.json` - Discovered repositories catalog
- **Workspaces**: `workspaces/` - Individual workspace configurations
- **Default Workspace Location**: `~/workspaces/YYYY-MM-DD/`

### Workspace Structure

Each workspace includes:

- **`.wsm/wsm.json`**: Metadata file with workspace information and environment variables
- **`.wsm/setup.sh`**: Optional setup script executed after workspace creation/fork
- **`.wsm/setup.d/`**: Directory for multiple setup scripts (executed in lexical order)
- **`.wsm/tmux.conf`**: Default tmux configuration for the workspace
- **`.wsm/profiles/PROFILE/tmux.conf`**: Profile-specific tmux configurations

### Environment Variables

**Global Configuration:**
- `WORKSPACE_MANAGER_LOG_LEVEL`: Set logging level (trace, debug, info, warn, error, fatal)
- `WORKSPACE_MANAGER_WORKSPACE_DIR`: Override default workspace directory

**Setup Script Environment:**
These variables are automatically provided to setup scripts:
- `WSM_WORKSPACE_NAME`: Name of the workspace
- `WSM_WORKSPACE_PATH`: Absolute path to the workspace
- `WSM_WORKSPACE_BRANCH`: Current workspace branch
- `WSM_WORKSPACE_BASE_BRANCH`: Base branch (for forks)
- `WSM_WORKSPACE_REPOS`: Comma-separated list of repository names

## Examples

### Microservices Development

```bash
# Discover your microservices
wsm discover ~/projects/microservices --recursive

# Create workspace for feature development
wsm create user-auth-feature --repos api-gateway,user-service,auth-service --branch feature/oauth-integration

# Start tmux session with development profile
wsm tmux user-auth-feature --profile development

# Check status across all services
wsm status

# Commit changes with consistent message
wsm commit -m "Add OAuth integration across services"

# Push all branches
wsm push origin

# Merge back to main when done
wsm merge
```

### Go Project Development

```bash
# Create workspace for Go projects (automatically creates go.work)
wsm create refactor-database --repos backend,shared-models,migration-tools

# The workspace will include a go.work file for module coordination
cd ~/workspaces/2025-01-15/refactor-database/
cat go.work
# go 1.21
# use ./backend
# use ./shared-models  
# use ./migration-tools

# Start tmux session for the workspace
wsm tmux refactor-database
```

### Library and Application Development

```bash
# Work on library and its dependent applications simultaneously
wsm create library-update --repos core-lib,web-app,cli-tool --branch feature/api-v2

# Start tmux session
wsm tmux library-update

# Make changes to library
cd ~/workspaces/2025-01-15/library-update/core-lib
# ... make changes ...

# Test changes in applications
cd ../web-app
go test ./...

cd ../cli-tool  
go build ./cmd/cli
```

## Advanced Usage

### Fork-Based Development Workflow

```bash
# Start with a main workspace
wsm create main-project --repos app,lib,api --branch main

# Fork for feature development
wsm fork feature-auth main-project
# Creates: task/feature-auth branch from main

# Start tmux session for the feature
wsm tmux feature-auth

# Work on the feature...
cd ~/workspaces/2025-01-15/feature-auth/
# ... make changes ...

# When done, merge back
wsm merge feature-auth
# Merges task/feature-auth → main and deletes feature-auth workspace
```

### Custom Branch Prefixes

```bash
# Use different branch prefixes for different types of work
wsm create db-fix --repos backend,migrations --branch-prefix bug
# Creates branch: bug/db-fix

wsm create new-api --repos api,client --branch-prefix feature  
# Creates branch: feature/new-api

# Fork with custom branch prefix
wsm fork hotfix-issue --branch-prefix hotfix
# Creates branch: hotfix/hotfix-issue
```

### Setup Scripts and Automation

WSM automatically executes setup scripts after workspace creation:

```bash
# Create a workspace - setup scripts run automatically
wsm create my-workspace --repos app,lib

# Setup scripts are executed in this order:
# 1. .wsm/setup.sh (workspace root)
# 2. .wsm/setup.d/*.sh (workspace root, lexical order)
# 3. Each repo's .wsm/setup.sh
# 4. Each repo's .wsm/setup.d/*.sh (lexical order)
```

Example setup script (`.wsm/setup.sh`):
```bash
#!/bin/bash
# Environment variables are available:
echo "Setting up workspace: $WSM_WORKSPACE_NAME"
echo "Workspace path: $WSM_WORKSPACE_PATH"
echo "Repositories: $WSM_WORKSPACE_REPOS"

# Install dependencies, setup environment, etc.
npm install
```

### Tmux Profiles and Configuration

Create profile-specific tmux configurations:

```bash
# Use development profile
wsm tmux my-workspace --profile development

# Use testing profile
wsm tmux my-workspace --profile testing
```

Example tmux.conf (`.wsm/profiles/development/tmux.conf`):
```
# Development profile tmux configuration
new-window -n "editor" "vim ."
new-window -n "server" "npm run dev"
new-window -n "tests" "npm run test:watch"
split-window -h "tail -f logs/app.log"
```

### Agent Configuration

Copy an `AGENT.md` file to your workspace for AI coding assistants:

```bash
wsm create my-workspace --repos app,lib --agent-source ~/templates/AGENT.md
```

### Dry Run Mode

Preview operations without making changes:

```bash
# Preview workspace creation
wsm create test-workspace --repos app,lib --dry-run

# Preview fork operation
wsm fork new-feature --dry-run

# Preview merge operation
wsm merge --dry-run
```

## How It Works

WSM leverages **git worktrees** to create efficient multi-repository workspaces:

1. **Discovery**: Scans directories to build a registry of available repositories
2. **Workspace Creation**: Creates a workspace directory with git worktrees for each repository
3. **Setup Automation**: Executes setup scripts and creates metadata files automatically
4. **Fork & Merge**: Fork workspaces for feature development and merge back to parent branches
5. **Branch Coordination**: Ensures all repositories are on the same branch (creates if needed)
6. **Go Integration**: Automatically generates `go.work` files for Go projects
7. **Tmux Integration**: Creates and manages tmux sessions with profile-based configurations
8. **Status Tracking**: Monitors git status across all workspace repositories
9. **Safe Operations**: Provides rollback mechanisms and proper cleanup

### Why Git Worktrees?

- **No Cloning Overhead**: Share objects with main repository
- **Independent Working Directories**: Each worktree has its own working directory and index
- **Branch Isolation**: Different worktrees can be on different branches
- **Shared History**: All worktrees share the same git history and objects

## Contributing

We welcome contributions! Please see our [contributing guidelines](CONTRIBUTING.md) for details.

### Development Setup

```bash
git clone https://github.com/go-go-golems/workspace-manager.git
cd workspace-manager
go mod download
go build ./cmd/workspace-manager
go test ./...
```

### Adding New Commands

1. Create `cmd/cmds/cmd_<name>.go`
2. Implement `func New<Name>Command() *cobra.Command`
3. Add to root command in `cmd/cmds/root.go`
4. Add business logic to `WorkspaceManager` if needed

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Support

- 📖 [Implementation Guide](IMPLEMENTATION.md) - Detailed implementation documentation
- 🐛 [Issue Tracker](https://github.com/go-go-golems/workspace-manager/issues) - Report bugs or request features
- 💬 [Discussions](https://github.com/go-go-golems/workspace-manager/discussions) - Community discussions

---

**Made with ❤️ by the [go-go-golems](https://github.com/go-go-golems) team**
