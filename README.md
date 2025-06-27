# Workspace Manager

**A powerful CLI tool for managing multi-repository workspaces with git worktrees**

Workspace Manager simplifies coordinated development across multiple related git repositories by automating workspace setup, git operations, and status tracking. Perfect for microservices, monorepos, or any project spanning multiple repositories.

## Features

- **🔍 Repository Discovery**: Automatically discover and catalog git repositories across your development environment
- **🏗️ Workspace Creation**: Create coordinated workspaces with git worktrees for multi-repo development
- **🔀 Fork & Merge Workflow**: Fork existing workspaces for feature development and merge back to parent branches
- **📊 Status Tracking**: Monitor git status across all repositories in a workspace simultaneously
- **🔄 Synchronized Operations**: Commit, push, and sync changes across multiple repositories with consistent messaging
- **🌿 Branch Management**: Coordinate branch operations across all workspace repositories
- **🔧 Go Integration**: Automatic `go.work` file generation for Go projects
- **🧹 Safe Cleanup**: Proper worktree removal and workspace cleanup

## Installation

### From Releases

Download the latest binary from the [releases page](https://github.com/go-go-golems/workspace-manager/releases):

```bash
# Linux/macOS
curl -L https://github.com/go-go-golems/workspace-manager/releases/latest/download/workspace-manager-$(uname -s)-$(uname -m) -o workspace-manager
chmod +x workspace-manager
sudo mv workspace-manager /usr/local/bin/
```

### Using Go Install

```bash
go install github.com/go-go-golems/workspace-manager/cmd/workspace-manager@latest
```

### Build from Source

```bash
git clone https://github.com/go-go-golems/workspace-manager.git
cd workspace-manager
go build ./cmd/workspace-manager
```

## Quick Start

### 1. Discover Repositories

First, let the workspace manager discover your existing git repositories:

```bash
# Discover repositories in your common development directories
workspace-manager discover ~/code ~/projects --recursive

# Discover with custom depth limit
workspace-manager discover ~/code --max-depth 2
```

### 2. Create a Workspace

Create a workspace with multiple repositories for coordinated development:

```bash
# Create workspace with automatic branch naming (task/my-feature)
workspace-manager create my-feature --repos app,lib,shared

# Create workspace with custom branch
workspace-manager create my-feature --repos app,lib,shared --branch feature/new-api

# Interactive repository selection
workspace-manager create my-feature --interactive
```

### 2a. Fork an Existing Workspace

Create a new workspace by forking an existing one:

```bash
# Fork the current workspace
workspace-manager fork my-feature-branch

# Fork a specific workspace
workspace-manager fork my-feature-branch source-workspace

# Fork with custom branch name
workspace-manager fork my-feature-branch --branch feature/custom-name
```

### 3. Check Status

Monitor the status of all repositories in your workspace:

```bash
workspace-manager status
```

### 4. Work with Your Code

Navigate to your workspace directory (default: `~/workspaces/YYYY-MM-DD/my-feature/`) and start coding. Each repository is available as a git worktree on your specified branch.

### 5. Merge and Clean Up

When you're done with your work, merge the fork back to its parent branch:

```bash
# Merge current fork back to base branch and delete workspace
workspace-manager merge

# Merge but keep the workspace
workspace-manager merge --keep-workspace

# Preview merge without executing
workspace-manager merge --dry-run
```

### 6. Interactive Mode



## Commands Reference

### Repository Discovery

```bash
# Discover repositories in specified paths
workspace-manager discover [paths...] [flags]

# Examples
workspace-manager discover ~/code ~/projects
workspace-manager discover . --recursive --max-depth 3
```

### Workspace Management

```bash
# Create a new workspace
workspace-manager create <workspace-name> --repos <repo1,repo2,repo3>

# Fork an existing workspace
workspace-manager fork <new-workspace-name> [source-workspace-name]

# Merge fork back to parent branch
workspace-manager merge [workspace-name]

# List all workspaces and repositories
workspace-manager list

# Get workspace information
workspace-manager info [workspace-name]

# Delete a workspace
workspace-manager delete <workspace-name>
```

### Repository Operations

```bash
# Add repository to existing workspace
workspace-manager add <workspace-name> <repo-name>

# Remove repository from workspace
workspace-manager remove <workspace-name> <repo-name>

# Show workspace status
workspace-manager status [workspace-name]
```

### Git Operations

```bash
# Commit changes across workspace repositories
workspace-manager commit -m "Your commit message"

# Push workspace branches
workspace-manager push [remote]

# Sync repositories (pull latest changes)
workspace-manager sync

# Show diff across repositories
workspace-manager diff

# Show commit history
workspace-manager log

# Manage branches
workspace-manager branch <operation>

# Rebase workspace repositories
workspace-manager rebase
```

### Pull Request Management

```bash
# Create pull requests for workspace branches
workspace-manager pr
```

## Configuration

Workspace Manager uses a configuration directory at `~/.config/workspace-manager/`:

- **Registry**: `registry.json` - Discovered repositories catalog
- **Workspaces**: `workspaces/` - Individual workspace configurations
- **Default Workspace Location**: `~/workspaces/YYYY-MM-DD/`

### Environment Variables

- `WORKSPACE_MANAGER_LOG_LEVEL`: Set logging level (trace, debug, info, warn, error, fatal)
- `WORKSPACE_MANAGER_WORKSPACE_DIR`: Override default workspace directory

## Examples

### Microservices Development

```bash
# Discover your microservices
workspace-manager discover ~/projects/microservices --recursive

# Create workspace for feature development
workspace-manager create user-auth-feature --repos api-gateway,user-service,auth-service --branch feature/oauth-integration

# Check status across all services
workspace-manager status

# Commit changes with consistent message
workspace-manager commit -m "Add OAuth integration across services"

# Push all branches
workspace-manager push origin

# Merge back to main when done
workspace-manager merge
```

### Go Project Development

```bash
# Create workspace for Go projects (automatically creates go.work)
workspace-manager create refactor-database --repos backend,shared-models,migration-tools

# The workspace will include a go.work file for module coordination
cd ~/workspaces/2025-01-15/refactor-database/
cat go.work
# go 1.21
# use ./backend
# use ./shared-models  
# use ./migration-tools
```

### Library and Application Development

```bash
# Work on library and its dependent applications simultaneously
workspace-manager create library-update --repos core-lib,web-app,cli-tool --branch feature/api-v2

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
workspace-manager create main-project --repos app,lib,api --branch main

# Fork for feature development
workspace-manager fork feature-auth main-project
# Creates: task/feature-auth branch from main

# Work on the feature...
cd ~/workspaces/2025-01-15/feature-auth/
# ... make changes ...

# When done, merge back
workspace-manager merge feature-auth
# Merges task/feature-auth → main and deletes feature-auth workspace
```

### Custom Branch Prefixes

```bash
# Use different branch prefixes for different types of work
workspace-manager create db-fix --repos backend,migrations --branch-prefix bug
# Creates branch: bug/db-fix

workspace-manager create new-api --repos api,client --branch-prefix feature  
# Creates branch: feature/new-api

# Fork with custom branch prefix
workspace-manager fork hotfix-issue --branch-prefix hotfix
# Creates branch: hotfix/hotfix-issue
```

### Agent Configuration

Copy an `AGENT.md` file to your workspace for AI coding assistants:

```bash
workspace-manager create my-workspace --repos app,lib --agent-source ~/templates/AGENT.md
```

### Dry Run Mode

Preview operations without making changes:

```bash
# Preview workspace creation
workspace-manager create test-workspace --repos app,lib --dry-run

# Preview fork operation
workspace-manager fork new-feature --dry-run

# Preview merge operation
workspace-manager merge --dry-run
```

## How It Works

Workspace Manager leverages **git worktrees** to create efficient multi-repository workspaces:

1. **Discovery**: Scans directories to build a registry of available repositories
2. **Workspace Creation**: Creates a workspace directory with git worktrees for each repository
3. **Fork & Merge**: Fork workspaces for feature development and merge back to parent branches
4. **Branch Coordination**: Ensures all repositories are on the same branch (creates if needed)
5. **Go Integration**: Automatically generates `go.work` files for Go projects
6. **Status Tracking**: Monitors git status across all workspace repositories
7. **Safe Operations**: Provides rollback mechanisms and proper cleanup

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

1. Create `cmd/cmd_<name>.go`
2. Implement `func New<Name>Command() *cobra.Command`
3. Add to root command in `cmd/root.go`
4. Add business logic to `WorkspaceManager` if needed

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Support

- 📖 [Implementation Guide](IMPLEMENTATION.md) - Detailed implementation documentation
- 🐛 [Issue Tracker](https://github.com/go-go-golems/workspace-manager/issues) - Report bugs or request features
- 💬 [Discussions](https://github.com/go-go-golems/workspace-manager/discussions) - Community discussions

---

**Made with ❤️ by the [go-go-golems](https://github.com/go-go-golems) team**
