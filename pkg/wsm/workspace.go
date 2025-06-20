package wsm

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/go-go-golems/workspace-manager/pkg/output"
	"github.com/pkg/errors"
)

// WorkspaceManager handles workspace creation and management
type WorkspaceManager struct {
	config       *WorkspaceConfig
	Discoverer   *RepositoryDiscoverer
	workspaceDir string
}

func getRegistryPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "workspace-manager", "registry.json"), nil
}

// NewWorkspaceManager creates a new workspace manager
func NewWorkspaceManager() (*WorkspaceManager, error) {
	config, err := loadConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to load config")
	}

	registryPath, err := getRegistryPath()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get registry path")
	}

	discoverer := NewRepositoryDiscoverer(registryPath)
	if err := discoverer.LoadRegistry(); err != nil {
		return nil, errors.Wrap(err, "failed to load registry")
	}

	return &WorkspaceManager{
		config:       config,
		Discoverer:   discoverer,
		workspaceDir: config.WorkspaceDir,
	}, nil
}

// CreateWorkspace creates a new multi-repository workspace
func (wm *WorkspaceManager) CreateWorkspace(ctx context.Context, name string, repoNames []string, branch string, agentSource string, dryRun bool) (*Workspace, error) {
	// Validate input
	if name == "" {
		return nil, errors.New("workspace name is required")
	}

	// Find repositories
	repos, err := wm.FindRepositories(repoNames)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find repositories")
	}

	// Create workspace directory path
	workspacePath := filepath.Join(wm.workspaceDir, name)

	workspace := &Workspace{
		Name:         name,
		Path:         workspacePath,
		Repositories: repos,
		Branch:       branch,
		Created:      time.Now(),
		GoWorkspace:  wm.shouldCreateGoWorkspace(repos),
		AgentMD:      agentSource,
	}

	if dryRun {
		return workspace, nil
	}

	// Create workspace
	if err := wm.createWorkspaceStructure(ctx, workspace); err != nil {
		return nil, errors.Wrap(err, "failed to create workspace structure")
	}

	// Save workspace configuration
	if err := wm.SaveWorkspace(workspace); err != nil {
		return nil, errors.Wrap(err, "failed to save workspace configuration")
	}

	return workspace, nil
}

// findRepositories finds repositories by name
func (wm *WorkspaceManager) FindRepositories(repoNames []string) ([]Repository, error) {
	allRepos := wm.Discoverer.GetRepositories()
	repoMap := make(map[string]Repository)

	for _, repo := range allRepos {
		repoMap[repo.Name] = repo
	}

	var repos []Repository
	var notFound []string

	for _, name := range repoNames {
		if repo, exists := repoMap[name]; exists {
			repos = append(repos, repo)
		} else {
			notFound = append(notFound, name)
		}
	}

	if len(notFound) > 0 {
		return nil, errors.Errorf("repositories not found: %s", strings.Join(notFound, ", "))
	}

	return repos, nil
}

// shouldCreateGoWorkspace determines if go.work should be created
func (wm *WorkspaceManager) shouldCreateGoWorkspace(repos []Repository) bool {
	for _, repo := range repos {
		for _, category := range repo.Categories {
			if category == "go" {
				return true
			}
		}
	}
	return false
}

// createWorkspaceStructure creates the physical workspace structure
func (wm *WorkspaceManager) createWorkspaceStructure(ctx context.Context, workspace *Workspace) error {
	output.LogInfo(
		fmt.Sprintf("Creating workspace structure for '%s'", workspace.Name),
		"Creating workspace structure",
		"workspace", workspace.Name,
	)

	// Create workspace directory
	if err := os.MkdirAll(workspace.Path, 0755); err != nil {
		return errors.Wrapf(err, "failed to create workspace directory: %s", workspace.Path)
	}

	// Track successfully created worktrees for rollback
	var createdWorktrees []WorktreeInfo

	// Create worktrees for each repository
	for _, repo := range workspace.Repositories {
		worktreeInfo := WorktreeInfo{
			Repository: repo,
			TargetPath: filepath.Join(workspace.Path, repo.Name),
			Branch:     workspace.Branch,
		}

		if err := wm.createWorktree(ctx, workspace, repo); err != nil {
			// Rollback any worktrees created so far
			output.LogError(
				fmt.Sprintf("Failed to create worktree for repository '%s'", repo.Name),
				"Failed to create worktree, rolling back",
				"repo", repo.Name,
				"createdWorktrees", len(createdWorktrees),
				"error", err,
			)

			wm.rollbackWorktrees(ctx, createdWorktrees)
			wm.cleanupWorkspaceDirectory(workspace.Path)
			return errors.Wrapf(err, "failed to create worktree for %s", repo.Name)
		}

		// Track successful creation
		createdWorktrees = append(createdWorktrees, worktreeInfo)
		output.LogInfo(
			fmt.Sprintf("Successfully created worktree for '%s'", repo.Name),
			"Successfully created worktree",
			"repo", repo.Name,
			"path", worktreeInfo.TargetPath,
		)
	}

	// Create go.work file if needed
	if workspace.GoWorkspace {
		if err := wm.CreateGoWorkspace(workspace); err != nil {
			output.LogError(
				"Failed to create go.work file",
				"Failed to create go.work file, rolling back worktrees",
				"error", err,
			)
			wm.rollbackWorktrees(ctx, createdWorktrees)
			wm.cleanupWorkspaceDirectory(workspace.Path)
			return errors.Wrap(err, "failed to create go.work file")
		}
	}

	// Copy AGENT.md if specified
	if workspace.AgentMD != "" {
		if err := wm.copyAgentMD(workspace); err != nil {
			output.LogError(
				"Failed to copy AGENT.md file",
				"Failed to copy AGENT.md, rolling back worktrees",
				"error", err,
			)
			wm.rollbackWorktrees(ctx, createdWorktrees)
			wm.cleanupWorkspaceDirectory(workspace.Path)
			return errors.Wrap(err, "failed to copy AGENT.md")
		}
	}

	output.LogInfo(
		fmt.Sprintf("Successfully created workspace structure for '%s' with %d worktrees", workspace.Name, len(createdWorktrees)),
		"Successfully created workspace structure",
		"workspace", workspace.Name,
		"worktrees", len(createdWorktrees),
	)

	return nil
}

// createWorktree creates a git worktree for a repository
func (wm *WorkspaceManager) createWorktree(ctx context.Context, workspace *Workspace, repo Repository) error {
	targetPath := filepath.Join(workspace.Path, repo.Name)

	output.LogInfo(
		fmt.Sprintf("Creating worktree for '%s' on branch '%s'", repo.Name, workspace.Branch),
		"Creating worktree",
		"repo", repo.Name,
		"branch", workspace.Branch,
		"target", targetPath,
	)

	if workspace.Branch == "" {
		// No specific branch, create worktree from current branch
		return wm.ExecuteWorktreeCommand(ctx, repo.Path, "git", "worktree", "add", targetPath)
	}

	// Check if branch exists locally
	branchExists, err := wm.CheckBranchExists(ctx, repo.Path, workspace.Branch)
	if err != nil {
		return errors.Wrapf(err, "failed to check if branch %s exists", workspace.Branch)
	}

	// Check if branch exists remotely
	remoteBranchExists, err := wm.CheckRemoteBranchExists(ctx, repo.Path, workspace.Branch)
	if err != nil {
		output.LogWarn(
			fmt.Sprintf("Could not check if remote branch '%s' exists", workspace.Branch),
			"Could not check remote branch existence",
			"branch", workspace.Branch,
			"error", err,
		)
	}

	fmt.Printf("\nBranch status for %s:\n", repo.Name)
	fmt.Printf("  Local branch '%s' exists: %v\n", workspace.Branch, branchExists)
	fmt.Printf("  Remote branch 'origin/%s' exists: %v\n", workspace.Branch, remoteBranchExists)

	if branchExists {
		// Branch exists locally - ask user what to do using huh
		output.PrintWarning("Branch '%s' already exists in repository '%s'", workspace.Branch, repo.Name)

		var choice string
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("How would you like to handle the existing branch?").
					Options(
						huh.NewOption("Overwrite the existing branch (git worktree add -B)", "overwrite"),
						huh.NewOption("Use the existing branch as-is (git worktree add)", "use"),
						huh.NewOption("Cancel workspace creation", "cancel"),
					).
					Value(&choice),
			),
		)

		err := form.Run()
		if err != nil {
			// Check if user cancelled/aborted the form
			errMsg := strings.ToLower(err.Error())
			if strings.Contains(errMsg, "user aborted") ||
				strings.Contains(errMsg, "cancelled") ||
				strings.Contains(errMsg, "aborted") ||
				strings.Contains(errMsg, "interrupt") {
				return errors.New("workspace creation cancelled by user")
			}
			return errors.Wrap(err, "failed to get user choice")
		}

		switch choice {
		case "overwrite":
			output.PrintInfo("Overwriting branch '%s'...", workspace.Branch)
			if remoteBranchExists {
				return wm.ExecuteWorktreeCommand(ctx, repo.Path, "git", "worktree", "add", "-B", workspace.Branch, targetPath, "origin/"+workspace.Branch)
			} else {
				return wm.ExecuteWorktreeCommand(ctx, repo.Path, "git", "worktree", "add", "-B", workspace.Branch, targetPath)
			}
		case "use":
			output.PrintInfo("Using existing branch '%s'...", workspace.Branch)
			return wm.ExecuteWorktreeCommand(ctx, repo.Path, "git", "worktree", "add", targetPath, workspace.Branch)
		case "cancel":
			return errors.New("workspace creation cancelled by user")
		default:
			return errors.New("invalid choice, workspace creation cancelled")
		}
	} else {
		// Branch doesn't exist locally
		if remoteBranchExists {
			output.PrintInfo("Creating worktree from remote branch origin/%s...", workspace.Branch)
			return wm.ExecuteWorktreeCommand(ctx, repo.Path, "git", "worktree", "add", "-b", workspace.Branch, targetPath, "origin/"+workspace.Branch)
		} else {
			output.PrintInfo("Creating new branch '%s' and worktree...", workspace.Branch)
			return wm.ExecuteWorktreeCommand(ctx, repo.Path, "git", "worktree", "add", "-b", workspace.Branch, targetPath)
		}
	}
}

// checkBranchExists checks if a local branch exists
func (wm *WorkspaceManager) CheckBranchExists(ctx context.Context, repoPath, branch string) (bool, error) {
	cmd := exec.CommandContext(ctx, "git", "show-ref", "--verify", "--quiet", "refs/heads/"+branch)
	cmd.Dir = repoPath
	err := cmd.Run()
	return err == nil, nil
}

// checkRemoteBranchExists checks if a remote branch exists
func (wm *WorkspaceManager) CheckRemoteBranchExists(ctx context.Context, repoPath, branch string) (bool, error) {
	cmd := exec.CommandContext(ctx, "git", "show-ref", "--verify", "--quiet", "refs/remotes/origin/"+branch)
	cmd.Dir = repoPath
	err := cmd.Run()
	return err == nil, nil
}

// executeWorktreeCommand executes a git worktree command with proper logging and error handling
func (wm *WorkspaceManager) ExecuteWorktreeCommand(ctx context.Context, repoPath string, args ...string) error {
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Dir = repoPath

	cmdStr := strings.Join(args, " ")
	fmt.Printf("Executing: %s (in %s)\n", cmdStr, repoPath)

	output.LogInfo(
		fmt.Sprintf("Executing git worktree command: %s", cmdStr),
		"Executing git worktree command",
		"command", cmdStr,
		"repoPath", repoPath,
	)

	cmdOutput, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("❌ Command failed: %s\n", cmdStr)
		fmt.Printf("   Error: %v\n", err)
		fmt.Printf("   Output: %s\n", string(cmdOutput))

		output.LogError(
			fmt.Sprintf("Git worktree command failed: %s", cmdStr),
			"Git worktree command failed",
			"error", err,
			"output", string(cmdOutput),
			"command", cmdStr,
		)

		return errors.Wrapf(err, "git command failed: %s", string(cmdOutput))
	}

	fmt.Printf("✓ Successfully executed: %s\n", cmdStr)
	if len(cmdOutput) > 0 {
		fmt.Printf("  Output: %s\n", string(cmdOutput))
	}

	output.LogInfo(
		fmt.Sprintf("Git worktree command succeeded: %s", cmdStr),
		"Git worktree command succeeded",
		"output", string(cmdOutput),
		"command", cmdStr,
	)

	return nil
}

// createGoWorkspace creates a go.work file
func (wm *WorkspaceManager) CreateGoWorkspace(workspace *Workspace) error {
	goWorkPath := filepath.Join(workspace.Path, "go.work")

	output.LogInfo(
		fmt.Sprintf("Creating go.work file at %s", goWorkPath),
		"Creating go.work file",
		"path", goWorkPath,
	)

	content := "go 1.23\n\nuse (\n"

	for _, repo := range workspace.Repositories {
		// Check if repo has go.mod
		goModPath := filepath.Join(workspace.Path, repo.Name, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			content += fmt.Sprintf("\t./%s\n", repo.Name)
		}
	}

	content += ")\n"

	if err := os.WriteFile(goWorkPath, []byte(content), 0644); err != nil {
		return errors.Wrapf(err, "failed to write go.work file")
	}

	return nil
}

// copyAgentMD copies AGENT.md file to workspace
func (wm *WorkspaceManager) copyAgentMD(workspace *Workspace) error {
	// Expand ~ in source path
	source := workspace.AgentMD
	if strings.HasPrefix(source, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return errors.Wrap(err, "failed to get home directory")
		}
		source = filepath.Join(home, source[1:])
	}

	target := filepath.Join(workspace.Path, "AGENT.md")

	output.LogInfo(
		fmt.Sprintf("Copying AGENT.md from %s to %s", source, target),
		"Copying AGENT.md",
		"source", source,
		"target", target,
	)

	data, err := os.ReadFile(source)
	if err != nil {
		return errors.Wrapf(err, "failed to read source file: %s", source)
	}

	if err := os.WriteFile(target, data, 0644); err != nil {
		return errors.Wrapf(err, "failed to write target file: %s", target)
	}

	return nil
}

// saveWorkspace saves workspace configuration
func (wm *WorkspaceManager) SaveWorkspace(workspace *Workspace) error {
	workspacesDir := filepath.Join(filepath.Dir(wm.config.RegistryPath), "workspaces")
	if err := os.MkdirAll(workspacesDir, 0755); err != nil {
		return errors.Wrap(err, "failed to create workspaces directory")
	}

	configPath := filepath.Join(workspacesDir, workspace.Name+".json")

	data, err := json.MarshalIndent(workspace, "", "  ")
	if err != nil {
		return errors.Wrap(err, "failed to marshal workspace configuration")
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return errors.Wrap(err, "failed to write workspace configuration")
	}

	return nil
}

// loadConfig loads workspace manager configuration
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

// LoadWorkspaces loads all workspace configurations
func LoadWorkspaces() ([]Workspace, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}

	workspacesDir := filepath.Join(configDir, "workspace-manager", "workspaces")

	if _, err := os.Stat(workspacesDir); os.IsNotExist(err) {
		return []Workspace{}, nil
	}

	entries, err := os.ReadDir(workspacesDir)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read workspaces directory")
	}

	var workspaces []Workspace
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			path := filepath.Join(workspacesDir, entry.Name())
			data, err := os.ReadFile(path)
			if err != nil {
				output.LogWarn(
					fmt.Sprintf("Failed to read workspace file: %s", path),
					"Failed to read workspace file",
					"path", path,
					"error", err,
				)
				continue
			}

			var workspace Workspace
			if err := json.Unmarshal(data, &workspace); err != nil {
				output.LogWarn(
					fmt.Sprintf("Failed to parse workspace file: %s", path),
					"Failed to parse workspace file",
					"path", path,
					"error", err,
				)
				continue
			}

			workspaces = append(workspaces, workspace)
		}
	}

	return workspaces, nil
}

// LoadWorkspace loads a specific workspace by name
func (wm *WorkspaceManager) LoadWorkspace(name string) (*Workspace, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}

	workspacePath := filepath.Join(configDir, "workspace-manager", "workspaces", name+".json")

	if _, err := os.Stat(workspacePath); os.IsNotExist(err) {
		return nil, errors.Errorf("workspace '%s' not found", name)
	}

	data, err := os.ReadFile(workspacePath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read workspace file: %s", workspacePath)
	}

	var workspace Workspace
	if err := json.Unmarshal(data, &workspace); err != nil {
		return nil, errors.Wrapf(err, "failed to parse workspace file: %s", workspacePath)
	}

	return &workspace, nil
}

// DeleteWorkspace deletes a workspace and optionally removes its files
func (wm *WorkspaceManager) DeleteWorkspace(ctx context.Context, name string, removeFiles bool, forceWorktrees bool) error {
	output.LogInfo(
		fmt.Sprintf("Deleting workspace '%s' (removeFiles: %v, forceWorktrees: %v)", name, removeFiles, forceWorktrees),
		"Deleting workspace",
		"workspace", name,
		"removeFiles", removeFiles,
		"forceWorktrees", forceWorktrees,
	)

	// Load workspace to get its path
	workspace, err := wm.LoadWorkspace(name)
	if err != nil {
		return errors.Wrapf(err, "failed to load workspace '%s'", name)
	}

	// Remove worktrees first
	if err := wm.removeWorktrees(ctx, workspace, forceWorktrees); err != nil {
		return errors.Wrap(err, "failed to remove worktrees")
	}

	// Remove workspace directory and files if requested
	if removeFiles {
		if _, err := os.Stat(workspace.Path); err == nil {
			output.LogInfo(
				fmt.Sprintf("Removing workspace directory and files: %s", workspace.Path),
				"Removing workspace directory and files",
				"path", workspace.Path,
			)

			// Log what we're removing for transparency
			if err := wm.logWorkspaceFilesToRemove(workspace.Path); err != nil {
				output.LogWarn(
					"Failed to enumerate workspace files for logging",
					"Failed to enumerate workspace files for logging",
					"error", err,
				)
			}

			if err := os.RemoveAll(workspace.Path); err != nil {
				return errors.Wrapf(err, "failed to remove workspace directory: %s", workspace.Path)
			}

			output.LogInfo(
				fmt.Sprintf("Successfully removed workspace directory and all files: %s", workspace.Path),
				"Successfully removed workspace directory and all files",
				"path", workspace.Path,
			)
		}
	} else {
		// If not removing files, still clean up go.work and AGENT.md from workspace directory
		// as these are workspace-specific files that should be removed with workspace deletion
		if err := wm.cleanupWorkspaceSpecificFiles(workspace.Path); err != nil {
			output.LogWarn(
				"Failed to clean up workspace-specific files",
				"Failed to clean up workspace-specific files",
				"error", err,
			)
		}
	}

	// Remove workspace configuration
	configDir, err := os.UserConfigDir()
	if err != nil {
		return errors.Wrap(err, "failed to get config directory")
	}

	configPath := filepath.Join(configDir, "workspace-manager", "workspaces", name+".json")
	if err := os.Remove(configPath); err != nil && !os.IsNotExist(err) {
		return errors.Wrapf(err, "failed to remove workspace configuration: %s", configPath)
	}

	output.LogInfo(
		fmt.Sprintf("Workspace '%s' deleted successfully", name),
		"Workspace deleted successfully",
		"workspace", name,
	)
	return nil
}

// removeWorktrees removes git worktrees for a workspace
func (wm *WorkspaceManager) removeWorktrees(ctx context.Context, workspace *Workspace, force bool) error {
	var errs []error

	// First, let's list existing worktrees for debugging
	output.PrintHeader("Workspace Cleanup Debug Info")
	for _, repo := range workspace.Repositories {
		output.PrintInfo("Repository: %s (at %s)", repo.Name, repo.Path)

		// List existing worktrees
		listCmd := exec.CommandContext(ctx, "git", "worktree", "list")
		listCmd.Dir = repo.Path
		if cmdOutput, err := listCmd.CombinedOutput(); err != nil {
			output.PrintWarning("Failed to list worktrees: %v", err)
		} else {
			output.PrintInfo("Current worktrees:\n%s", string(cmdOutput))
		}
	}
	output.PrintHeader("Starting Worktree Removal")

	for _, repo := range workspace.Repositories {
		worktreePath := filepath.Join(workspace.Path, repo.Name)

		output.LogInfo(
			fmt.Sprintf("Removing worktree for '%s'", repo.Name),
			"Removing worktree",
			"repo", repo.Name,
			"worktree", worktreePath,
		)

		fmt.Printf("\n--- Processing %s ---\n", repo.Name)
		fmt.Printf("Workspace path: %s\n", workspace.Path)
		fmt.Printf("Expected worktree path: %s\n", worktreePath)

		// Check if worktree path exists
		if stat, err := os.Stat(worktreePath); os.IsNotExist(err) {
			fmt.Printf("⚠️  Worktree directory does not exist, skipping\n")
			continue
		} else if err != nil {
			fmt.Printf("⚠️  Error checking worktree path: %v\n", err)
			continue
		} else {
			fmt.Printf("✓ Worktree directory exists (type: %s)\n", map[bool]string{true: "directory", false: "file"}[stat.IsDir()])
		}

		// Remove worktree using git command
		var cmd *exec.Cmd
		var cmdStr string
		if force {
			cmd = exec.CommandContext(ctx, "git", "worktree", "remove", "--force", worktreePath)
			cmdStr = fmt.Sprintf("git worktree remove --force %s", worktreePath)
		} else {
			cmd = exec.CommandContext(ctx, "git", "worktree", "remove", worktreePath)
			cmdStr = fmt.Sprintf("git worktree remove %s", worktreePath)
		}
		cmd.Dir = repo.Path

		output.LogInfo(
			fmt.Sprintf("Executing git worktree remove command: %s", cmdStr),
			"Executing git worktree remove command",
			"repo", repo.Name,
			"repoPath", repo.Path,
			"worktreePath", worktreePath,
			"command", cmdStr,
		)

		fmt.Printf("Executing: %s (in %s)\n", cmdStr, repo.Path)

		if cmdOutput, err := cmd.CombinedOutput(); err != nil {
			output.LogError(
				fmt.Sprintf("Failed to remove worktree for repository '%s'", repo.Name),
				"Failed to remove worktree with git command",
				"error", err,
				"output", string(cmdOutput),
				"repo", repo.Name,
				"repoPath", repo.Path,
				"worktree", worktreePath,
				"command", cmdStr,
			)

			fmt.Printf("❌ Command failed: %s\n", cmdStr)
			fmt.Printf("   Error: %v\n", err)
			fmt.Printf("   Output: %s\n", string(cmdOutput))

			errs = append(errs, errors.Wrapf(err, "failed to remove worktree for %s: %s", repo.Name, string(cmdOutput)))
		} else {
			output.LogInfo(
				fmt.Sprintf("Successfully removed worktree for '%s'", repo.Name),
				"Successfully removed worktree",
				"output", string(cmdOutput),
				"repo", repo.Name,
				"command", cmdStr,
			)

			fmt.Printf("✓ Successfully executed: %s\n", cmdStr)
			if len(cmdOutput) > 0 {
				fmt.Printf("  Output: %s\n", string(cmdOutput))
			}
		}
	}

	// Verify worktrees were removed
	fmt.Printf("\n=== Verification: Final Worktree State ===\n")
	for _, repo := range workspace.Repositories {
		fmt.Printf("\nRepository: %s\n", repo.Name)

		// List remaining worktrees
		listCmd := exec.CommandContext(ctx, "git", "worktree", "list")
		listCmd.Dir = repo.Path
		if output, err := listCmd.CombinedOutput(); err != nil {
			fmt.Printf("  ⚠️  Failed to list worktrees: %v\n", err)
		} else {
			fmt.Printf("  Remaining worktrees:\n%s", string(output))
		}
	}

	if len(errs) > 0 {
		var errMsgs []string
		for _, err := range errs {
			errMsgs = append(errMsgs, err.Error())
		}
		return errors.New("failed to remove some worktrees: " + strings.Join(errMsgs, "; "))
	}

	fmt.Printf("=== Worktree cleanup completed ===\n\n")
	return nil
}

// logWorkspaceFilesToRemove logs the files that will be removed for transparency
func (wm *WorkspaceManager) logWorkspaceFilesToRemove(workspacePath string) error {
	entries, err := os.ReadDir(workspacePath)
	if err != nil {
		return err
	}

	var files []string
	var dirs []string

	for _, entry := range entries {
		if entry.IsDir() {
			dirs = append(dirs, entry.Name())
		} else {
			files = append(files, entry.Name())
		}
	}

	output.LogInfo(
		fmt.Sprintf("Workspace %s contains %d items to be removed", workspacePath, len(entries)),
		"Workspace contents to be removed",
		"workspacePath", workspacePath,
		"files", files,
		"directories", dirs,
		"totalItems", len(entries),
	)

	return nil
}

// cleanupWorkspaceSpecificFiles removes workspace-specific files (go.work, AGENT.md)
// even when not doing a full directory removal
func (wm *WorkspaceManager) cleanupWorkspaceSpecificFiles(workspacePath string) error {
	workspaceSpecificFiles := []string{"go.work", "go.work.sum", "AGENT.md"}

	for _, fileName := range workspaceSpecificFiles {
		filePath := filepath.Join(workspacePath, fileName)

		if _, err := os.Stat(filePath); err == nil {
			output.LogInfo(
				fmt.Sprintf("Removing workspace file %s", fileName),
				"Removing workspace-specific file",
				"file", filePath,
			)

			if err := os.Remove(filePath); err != nil {
				output.LogWarn(
					fmt.Sprintf("Failed to remove workspace-specific file: %s", filePath),
					"Failed to remove workspace-specific file",
					"file", filePath,
					"error", err,
				)
				return errors.Wrapf(err, "failed to remove %s", filePath)
			}

			output.LogInfo(
				fmt.Sprintf("Successfully removed %s", fileName),
				"Successfully removed workspace-specific file",
				"file", filePath,
			)
		} else if !os.IsNotExist(err) {
			output.LogWarn(
				fmt.Sprintf("Error checking workspace-specific file: %s", filePath),
				"Error checking workspace-specific file",
				"file", filePath,
				"error", err,
			)
		}
	}

	return nil
}

// rollbackWorktrees removes worktrees that were created during a failed workspace creation
func (wm *WorkspaceManager) rollbackWorktrees(ctx context.Context, worktrees []WorktreeInfo) {
	if len(worktrees) == 0 {
		return
	}

	fmt.Printf("\n🔄 Rolling back %d created worktrees...\n", len(worktrees))
	output.LogInfo(
		fmt.Sprintf("Rolling back %d created worktrees", len(worktrees)),
		"Rolling back created worktrees",
		"count", len(worktrees),
	)

	for i := len(worktrees) - 1; i >= 0; i-- {
		worktree := worktrees[i]

		fmt.Printf("Rolling back worktree: %s (at %s)\n", worktree.Repository.Name, worktree.TargetPath)

		output.LogInfo(
			fmt.Sprintf("Rolling back worktree for %s", worktree.Repository.Name),
			"Rolling back worktree",
			"repo", worktree.Repository.Name,
			"targetPath", worktree.TargetPath,
			"repoPath", worktree.Repository.Path,
		)

		// Use git worktree remove --force for rollback to ensure it works even with uncommitted changes
		cmd := exec.CommandContext(ctx, "git", "worktree", "remove", "--force", worktree.TargetPath)
		cmd.Dir = worktree.Repository.Path

		cmdStr := fmt.Sprintf("git worktree remove --force %s", worktree.TargetPath)
		fmt.Printf("  Executing: %s (in %s)\n", cmdStr, worktree.Repository.Path)

		if cmdOutput, err := cmd.CombinedOutput(); err != nil {
			fmt.Printf("  ⚠️  Failed to remove worktree: %v\n", err)
			fmt.Printf("      Output: %s\n", string(cmdOutput))

			output.LogWarn(
				fmt.Sprintf("Failed to remove worktree for '%s' during rollback", worktree.Repository.Name),
				"Failed to remove worktree during rollback",
				"error", err,
				"output", string(cmdOutput),
				"repo", worktree.Repository.Name,
				"targetPath", worktree.TargetPath,
			)
		} else {
			fmt.Printf("  ✓ Successfully removed worktree\n")

			output.LogInfo(
				fmt.Sprintf("Successfully removed worktree for %s", worktree.Repository.Name),
				"Successfully removed worktree during rollback",
				"repo", worktree.Repository.Name,
				"targetPath", worktree.TargetPath,
			)
		}
	}

	fmt.Printf("🔄 Rollback completed\n\n")
	output.LogInfo("Rollback completed", "Worktree rollback completed")
}

// cleanupWorkspaceDirectory removes the workspace directory if it's empty or only contains expected files
func (wm *WorkspaceManager) cleanupWorkspaceDirectory(workspacePath string) {
	if workspacePath == "" {
		return
	}

	fmt.Printf("🧹 Cleaning up workspace directory: %s\n", workspacePath)
	output.LogInfo(
		fmt.Sprintf("Cleaning up workspace directory %s", workspacePath),
		"Cleaning up workspace directory",
		"path", workspacePath,
	)

	// Check if directory exists
	if _, err := os.Stat(workspacePath); os.IsNotExist(err) {
		fmt.Printf("  Directory doesn't exist, nothing to clean up\n")
		return
	}

	// Read directory contents
	entries, err := os.ReadDir(workspacePath)
	if err != nil {
		fmt.Printf("  ⚠️  Failed to read directory: %v\n", err)
		output.LogWarn(
			fmt.Sprintf("Failed to read workspace directory during cleanup: %s", workspacePath),
			"Failed to read workspace directory during cleanup",
			"path", workspacePath,
			"error", err,
		)
		return
	}

	// Check if directory is empty or only contains files we might have created
	isEmpty := len(entries) == 0
	onlyExpectedFiles := true
	expectedFiles := map[string]bool{
		"go.work":    true,
		"AGENT.md":   true,
		".gitignore": true,
	}

	if !isEmpty {
		for _, entry := range entries {
			if !expectedFiles[entry.Name()] {
				onlyExpectedFiles = false
				break
			}
		}
	}

	if isEmpty || onlyExpectedFiles {
		fmt.Printf("  Removing workspace directory (empty or only contains expected files)\n")
		if err := os.RemoveAll(workspacePath); err != nil {
			fmt.Printf("  ⚠️  Failed to remove workspace directory: %v\n", err)
			output.LogWarn(
				fmt.Sprintf("Failed to remove workspace directory during cleanup: %s", workspacePath),
				"Failed to remove workspace directory during cleanup",
				"path", workspacePath,
				"error", err,
			)
		} else {
			fmt.Printf("  ✓ Successfully removed workspace directory\n")
			output.LogInfo(
				fmt.Sprintf("Successfully removed workspace directory %s", workspacePath),
				"Successfully removed workspace directory during cleanup",
				"path", workspacePath,
			)
		}
	} else {
		fmt.Printf("  Directory contains unexpected files, leaving it intact\n")
		output.LogInfo(
			fmt.Sprintf("Workspace directory %s contains %d unexpected files", workspacePath, len(entries)),
			"Workspace directory contains unexpected files, not removing",
			"path", workspacePath,
			"entries", len(entries),
		)

		// List the unexpected files for debugging
		for _, entry := range entries {
			if !expectedFiles[entry.Name()] {
				fmt.Printf("    Unexpected file/directory: %s\n", entry.Name())
			}
		}
	}
}

// AddRepositoryToWorkspace adds a repository to an existing workspace
func (wm *WorkspaceManager) AddRepositoryToWorkspace(ctx context.Context, workspaceName, repoName, branchName string, forceOverwrite bool) error {
	output.LogInfo(
		fmt.Sprintf("Adding repository %s to workspace %s", repoName, workspaceName),
		"Adding repository to workspace",
		"workspace", workspaceName,
		"repo", repoName,
		"branch", branchName,
		"force", forceOverwrite,
	)

	// Load existing workspace
	workspace, err := wm.LoadWorkspace(workspaceName)
	if err != nil {
		return errors.Wrapf(err, "failed to load workspace '%s'", workspaceName)
	}

	// Check if repository is already in workspace
	for _, repo := range workspace.Repositories {
		if repo.Name == repoName {
			return errors.Errorf("repository '%s' is already in workspace '%s'", repoName, workspaceName)
		}
	}

	// Find the repository in the registry
	repos, err := wm.FindRepositories([]string{repoName})
	if err != nil {
		return errors.Wrapf(err, "failed to find repository '%s'", repoName)
	}

	if len(repos) == 0 {
		return errors.Errorf("repository '%s' not found in registry", repoName)
	}

	repo := repos[0]

	// Use the workspace's branch if no specific branch provided
	targetBranch := branchName
	if targetBranch == "" {
		targetBranch = workspace.Branch
	}

	// Create a temporary workspace with the new repository for worktree creation
	tempWorkspace := *workspace
	tempWorkspace.Branch = targetBranch
	tempWorkspace.Repositories = []Repository{repo}

	output.PrintInfo("Adding repository '%s' to workspace '%s'", repoName, workspaceName)
	output.PrintInfo("Target branch: %s", targetBranch)
	output.PrintInfo("Workspace path: %s", workspace.Path)

	// Create worktree for the new repository
	if err := wm.CreateWorktreeForAdd(ctx, workspace, repo, targetBranch, forceOverwrite); err != nil {
		return errors.Wrapf(err, "failed to create worktree for repository '%s'", repoName)
	}

	// Add repository to workspace configuration
	workspace.Repositories = append(workspace.Repositories, repo)

	// Update go.work file if this is a Go workspace and the new repo has go.mod
	if workspace.GoWorkspace {
		if err := wm.CreateGoWorkspace(workspace); err != nil {
			output.LogWarn(
				fmt.Sprintf("Failed to update go.work file: %v", err),
				"Failed to update go.work file, but continuing",
				"error", err,
			)
		}
	}

	// Save updated workspace configuration
	if err := wm.SaveWorkspace(workspace); err != nil {
		return errors.Wrap(err, "failed to save updated workspace configuration")
	}

	fmt.Printf("✓ Successfully added repository '%s' to workspace '%s'\n", repoName, workspaceName)
	return nil
}

// CreateWorktreeForAdd creates a worktree for adding a repository to an existing workspace
func (wm *WorkspaceManager) CreateWorktreeForAdd(ctx context.Context, workspace *Workspace, repo Repository, branch string, forceOverwrite bool) error {
	targetPath := filepath.Join(workspace.Path, repo.Name)

	output.LogInfo(
		fmt.Sprintf("Creating worktree for %s at %s", repo.Name, targetPath),
		"Creating worktree for add operation",
		"repo", repo.Name,
		"branch", branch,
		"target", targetPath,
		"force", forceOverwrite,
	)

	// Check if target path already exists
	if _, err := os.Stat(targetPath); err == nil {
		return errors.Errorf("target path '%s' already exists", targetPath)
	}

	if branch == "" {
		// No specific branch, create worktree from current branch
		return wm.ExecuteWorktreeCommand(ctx, repo.Path, "git", "worktree", "add", targetPath)
	}

	// Check if branch exists locally
	branchExists, err := wm.CheckBranchExists(ctx, repo.Path, branch)
	if err != nil {
		return errors.Wrapf(err, "failed to check if branch %s exists", branch)
	}

	// Check if branch exists remotely
	remoteBranchExists, err := wm.CheckRemoteBranchExists(ctx, repo.Path, branch)
	if err != nil {
		output.LogWarn(
			fmt.Sprintf("Could not check remote branch existence for '%s': %v", branch, err),
			"Could not check remote branch existence",
			"error", err,
			"branch", branch,
		)
	}

	fmt.Printf("\nBranch status for %s:\n", repo.Name)
	fmt.Printf("  Local branch '%s' exists: %v\n", branch, branchExists)
	fmt.Printf("  Remote branch 'origin/%s' exists: %v\n", branch, remoteBranchExists)

	if branchExists {
		if forceOverwrite {
			fmt.Printf("Force overwriting branch '%s'...\n", branch)
			if remoteBranchExists {
				return wm.ExecuteWorktreeCommand(ctx, repo.Path, "git", "worktree", "add", "-B", branch, targetPath, "origin/"+branch)
			} else {
				return wm.ExecuteWorktreeCommand(ctx, repo.Path, "git", "worktree", "add", "-B", branch, targetPath)
			}
		} else {
			// Branch exists locally - ask user what to do unless force is specified
			fmt.Printf("\n⚠️  Branch '%s' already exists in repository '%s'\n", branch, repo.Name)
			fmt.Printf("What would you like to do?\n")
			fmt.Printf("  [o] Overwrite the existing branch (git worktree add -B)\n")
			fmt.Printf("  [u] Use the existing branch as-is (git worktree add)\n")
			fmt.Printf("  [c] Cancel operation\n")
			fmt.Printf("Choice [o/u/c]: ")

			var choice string
			if _, err := fmt.Scanln(&choice); err != nil {
				// If input fails, default to cancel to be safe
				choice = "c"
			}

			switch strings.ToLower(choice) {
			case "o", "overwrite":
				fmt.Printf("Overwriting branch '%s'...\n", branch)
				if remoteBranchExists {
					return wm.ExecuteWorktreeCommand(ctx, repo.Path, "git", "worktree", "add", "-B", branch, targetPath, "origin/"+branch)
				} else {
					return wm.ExecuteWorktreeCommand(ctx, repo.Path, "git", "worktree", "add", "-B", branch, targetPath)
				}
			case "u", "use":
				fmt.Printf("Using existing branch '%s'...\n", branch)
				return wm.ExecuteWorktreeCommand(ctx, repo.Path, "git", "worktree", "add", targetPath, branch)
			case "c", "cancel":
				return errors.New("operation cancelled by user")
			default:
				return errors.New("invalid choice, operation cancelled")
			}
		}
	} else {
		// Branch doesn't exist locally
		if remoteBranchExists {
			fmt.Printf("Creating worktree from remote branch origin/%s...\n", branch)
			return wm.ExecuteWorktreeCommand(ctx, repo.Path, "git", "worktree", "add", "-b", branch, targetPath, "origin/"+branch)
		} else {
			fmt.Printf("Creating new branch '%s' and worktree...\n", branch)
			return wm.ExecuteWorktreeCommand(ctx, repo.Path, "git", "worktree", "add", "-b", branch, targetPath)
		}
	}
}

// RemoveRepositoryFromWorkspace removes a repository from an existing workspace
func (wm *WorkspaceManager) RemoveRepositoryFromWorkspace(ctx context.Context, workspaceName, repoName string, force, removeFiles bool) error {
	output.LogInfo(
		fmt.Sprintf("Removing repository %s from workspace %s", repoName, workspaceName),
		"Removing repository from workspace",
		"workspace", workspaceName,
		"repo", repoName,
		"force", force,
		"removeFiles", removeFiles,
	)

	// Load existing workspace
	workspace, err := wm.LoadWorkspace(workspaceName)
	if err != nil {
		return errors.Wrapf(err, "failed to load workspace '%s'", workspaceName)
	}

	// Find the repository in the workspace
	var repoIndex = -1
	var targetRepo Repository
	for i, repo := range workspace.Repositories {
		if repo.Name == repoName {
			repoIndex = i
			targetRepo = repo
			break
		}
	}

	if repoIndex == -1 {
		return errors.Errorf("repository '%s' not found in workspace '%s'", repoName, workspaceName)
	}

	fmt.Printf("Removing repository '%s' from workspace '%s'\n", repoName, workspaceName)
	fmt.Printf("Repository path: %s\n", targetRepo.Path)
	fmt.Printf("Workspace path: %s\n", workspace.Path)

	// Remove the worktree
	worktreePath := filepath.Join(workspace.Path, repoName)
	if err := wm.removeWorktreeForRepo(ctx, targetRepo, worktreePath, force); err != nil {
		return errors.Wrapf(err, "failed to remove worktree for repository '%s'", repoName)
	}

	// Remove repository directory if requested
	if removeFiles {
		if _, err := os.Stat(worktreePath); err == nil {
			fmt.Printf("Removing repository directory: %s\n", worktreePath)
			if err := os.RemoveAll(worktreePath); err != nil {
				return errors.Wrapf(err, "failed to remove repository directory: %s", worktreePath)
			}
			fmt.Printf("✓ Successfully removed repository directory\n")
		}
	}

	// Remove repository from workspace configuration
	workspace.Repositories = append(workspace.Repositories[:repoIndex], workspace.Repositories[repoIndex+1:]...)

	// Update go.work file if this is a Go workspace
	if workspace.GoWorkspace {
		if err := wm.CreateGoWorkspace(workspace); err != nil {
			output.LogWarn(
				fmt.Sprintf("Failed to update go.work file: %v", err),
				"Failed to update go.work file, but continuing",
				"error", err,
			)
		}
	}

	// Save updated workspace configuration
	if err := wm.SaveWorkspace(workspace); err != nil {
		return errors.Wrap(err, "failed to save updated workspace configuration")
	}

	fmt.Printf("✓ Successfully removed repository '%s' from workspace '%s'\n", repoName, workspaceName)
	return nil
}

// removeWorktreeForRepo removes a worktree for a specific repository
func (wm *WorkspaceManager) removeWorktreeForRepo(ctx context.Context, repo Repository, worktreePath string, force bool) error {
	output.LogInfo(
		fmt.Sprintf("Removing worktree for %s at %s", repo.Name, worktreePath),
		"Removing worktree for repository",
		"repo", repo.Name,
		"worktree", worktreePath,
		"force", force,
	)

	fmt.Printf("\n--- Removing worktree for %s ---\n", repo.Name)
	fmt.Printf("Worktree path: %s\n", worktreePath)

	// Check if worktree path exists
	if stat, err := os.Stat(worktreePath); os.IsNotExist(err) {
		fmt.Printf("⚠️  Worktree directory does not exist, skipping worktree removal\n")
		return nil
	} else if err != nil {
		return errors.Wrapf(err, "error checking worktree path: %s", worktreePath)
	} else {
		fmt.Printf("✓ Worktree directory exists (type: %s)\n", map[bool]string{true: "directory", false: "file"}[stat.IsDir()])
	}

	// First, list current worktrees for debugging
	fmt.Printf("\nCurrent worktrees for %s:\n", repo.Name)
	listCmd := exec.CommandContext(ctx, "git", "worktree", "list")
	listCmd.Dir = repo.Path
	if output, err := listCmd.CombinedOutput(); err != nil {
		fmt.Printf("⚠️  Failed to list worktrees: %v\n", err)
	} else {
		fmt.Printf("%s", string(output))
	}

	// Remove worktree using git command
	var cmd *exec.Cmd
	var cmdStr string
	if force {
		cmd = exec.CommandContext(ctx, "git", "worktree", "remove", "--force", worktreePath)
		cmdStr = fmt.Sprintf("git worktree remove --force %s", worktreePath)
	} else {
		cmd = exec.CommandContext(ctx, "git", "worktree", "remove", worktreePath)
		cmdStr = fmt.Sprintf("git worktree remove %s", worktreePath)
	}
	cmd.Dir = repo.Path

	output.LogInfo(
		fmt.Sprintf("Executing: %s (in %s)", cmdStr, repo.Path),
		"Executing git worktree remove command",
		"repo", repo.Name,
		"repoPath", repo.Path,
		"worktreePath", worktreePath,
		"command", cmdStr,
	)

	fmt.Printf("Executing: %s (in %s)\n", cmdStr, repo.Path)

	cmdOutput, err := cmd.CombinedOutput()
	if err != nil {
		output.LogError(
			fmt.Sprintf("Failed to remove worktree for '%s': %v", repo.Name, err),
			"Failed to remove worktree with git command",
			"error", err,
			"output", string(cmdOutput),
			"repo", repo.Name,
			"repoPath", repo.Path,
			"worktree", worktreePath,
			"command", cmdStr,
		)

		return errors.Wrapf(err, "failed to remove worktree: %s", string(cmdOutput))
	}

	output.LogInfo(
		fmt.Sprintf("Successfully removed worktree for '%s'", repo.Name),
		"Successfully removed worktree",
		"output", string(cmdOutput),
		"repo", repo.Name,
		"command", cmdStr,
	)

	fmt.Printf("✓ Successfully executed: %s\n", cmdStr)
	if len(cmdOutput) > 0 {
		fmt.Printf("  Output: %s\n", string(cmdOutput))
	}

	// Verify worktree was removed
	fmt.Printf("\nVerification: Remaining worktrees for %s:\n", repo.Name)
	listCmd = exec.CommandContext(ctx, "git", "worktree", "list")
	listCmd.Dir = repo.Path
	if output, err := listCmd.CombinedOutput(); err != nil {
		fmt.Printf("⚠️  Failed to list worktrees: %v\n", err)
	} else {
		fmt.Printf("%s", string(output))
	}

	return nil
}
