package wsm

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/go-go-golems/workspace-manager/pkg/output"
	branchsvc "github.com/go-go-golems/workspace-manager/pkg/wsm/branch"
	"github.com/go-go-golems/workspace-manager/pkg/wsm/gitclient"
	"github.com/pkg/errors"
)

// WorkspaceManager handles workspace creation and management
type WorkspaceManager struct {
	config       *WorkspaceConfig
	Discoverer   *RepositoryDiscoverer
	workspaceDir string
	branches     branchsvc.Service
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
		branches:     BuildBranchService(context.Background()),
	}, nil
}

// CreateWorkspace creates a new multi-repository workspace
func (wm *WorkspaceManager) CreateWorkspace(ctx context.Context, name string, repoNames []string, branch string, baseBranch string, agentSource string, dryRun bool) (*Workspace, error) {
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
		BaseBranch:   baseBranch,
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

	// Create wsm.json metadata file
	if err := wm.createWorkspaceMetadata(workspace); err != nil {
		output.LogWarn(
			fmt.Sprintf("Failed to create wsm.json metadata file for workspace '%s'", workspace.Name),
			"Failed to create workspace metadata file",
			"workspace", workspace.Name,
			"error", err,
		)
		// Don't fail workspace creation if metadata file creation fails
	}

	output.LogInfo(
		fmt.Sprintf("Successfully created workspace structure for '%s' with %d worktrees", workspace.Name, len(createdWorktrees)),
		"Successfully created workspace structure",
		"workspace", workspace.Name,
		"worktrees", len(createdWorktrees),
	)

	// Execute setup scripts if they exist
	if err := wm.executeSetupScripts(ctx, workspace); err != nil {
		output.LogWarn(
			fmt.Sprintf("Failed to execute setup scripts for workspace '%s'", workspace.Name),
			"Setup scripts failed, workspace created but setup incomplete",
			"workspace", workspace.Name,
			"error", err,
		)
		// Don't fail workspace creation if setup scripts fail
	}

	return nil
}

// createWorktree creates a git worktree for a repository
func (wm *WorkspaceManager) createWorktree(ctx context.Context, workspace *Workspace, repo Repository) error {
	targetPath := filepath.Join(workspace.Path, repo.Name)
	_, wtm := BuildGitBackends(ctx)

	output.LogInfo(
		fmt.Sprintf("Creating worktree for '%s' on branch '%s'", repo.Name, workspace.Branch),
		"Creating worktree",
		"repo", repo.Name,
		"branch", workspace.Branch,
		"target", targetPath,
	)

	// No specific branch, create a detached worktree from current branch
	if workspace.Branch == "" {
		return wtm.Add(ctx, repo.Path, "", targetPath, gitclient.WorktreeAddOptions{})
	}

	plan, err := wm.resolveBranchPlan(ctx, repo.Path, branchsvc.BranchResolutionRequest{
		TargetBranch: branchsvc.BranchName(workspace.Branch),
		BaseBranch:   branchsvc.BranchName(workspace.BaseBranch),
		Remote:       branchsvc.DefaultRemoteName,
		Mode:         branchsvc.ResolutionModeCreateWorktree,
	})
	if err != nil {
		return errors.Wrap(err, "resolve branch plan")
	}

	opts := gitclient.WorktreeAddOptions{}
	if plan.Strategy == branchsvc.ResolutionStrategyUseLocal {
		// Use existing branch as-is (non-destructive default)
		return wtm.Add(ctx, repo.Path, workspace.Branch, targetPath, opts)
	}

	if plan.Strategy == branchsvc.ResolutionStrategyTrackRemote {
		opts.RemoteBranch = plan.RemoteRef
		return wtm.Add(ctx, repo.Path, workspace.Branch, targetPath, opts)
	}
	if plan.Strategy == branchsvc.ResolutionStrategyCreateFromBase {
		opts.BaseRef = plan.StartPoint
		return wtm.Add(ctx, repo.Path, workspace.Branch, targetPath, opts)
	}

	// Create new local branch from HEAD
	return wtm.Add(ctx, repo.Path, workspace.Branch, targetPath, opts)
}

func (wm *WorkspaceManager) branchService(ctx context.Context) branchsvc.Service {
	if wm.branches == nil {
		wm.branches = BuildBranchService(ctx)
	}
	return wm.branches
}

func (wm *WorkspaceManager) resolveBranchPlan(ctx context.Context, repoPath string, req branchsvc.BranchResolutionRequest) (*branchsvc.BranchResolutionPlan, error) {
	plan, err := wm.branchService(ctx).Resolve(ctx, repoPath, req)
	if err != nil {
		return nil, errors.Wrap(err, "resolve branch")
	}
	return plan, nil
}

// checkBranchExists checks if a local branch exists
func (wm *WorkspaceManager) CheckBranchExists(ctx context.Context, repoPath, branch string) (bool, error) {
	if branch == "" {
		return false, nil
	}
	localExists, err := wm.branchService(ctx).LocalExists(ctx, repoPath, branchsvc.BranchName(branch))
	if err != nil {
		return false, errors.Wrap(err, "check local branch")
	}
	return localExists, nil
}

// checkRemoteBranchExists checks if a remote branch exists
func (wm *WorkspaceManager) CheckRemoteBranchExists(ctx context.Context, repoPath, branch string) (bool, error) {
	if branch == "" {
		return false, nil
	}
	remoteExists, err := wm.branchService(ctx).RemoteTrackingExists(
		ctx,
		repoPath,
		branchsvc.DefaultRemoteName,
		branchsvc.BranchName(branch),
	)
	if err != nil {
		return false, errors.Wrap(err, "check remote branch")
	}
	return remoteExists, nil
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
	_, wtm := BuildGitBackends(ctx)

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

		// Check for untracked files that would preclude removal
		untrackedFiles, err := wm.getUntrackedFiles(ctx, worktreePath)
		if err != nil {
			output.LogWarn(
				fmt.Sprintf("Failed to check for untracked files in %s: %v", repo.Name, err),
				"Unable to check for untracked files",
				"repo", repo.Name,
				"error", err,
			)
		} else if len(untrackedFiles) > 0 && !force {
			fmt.Printf("\n⚠️  Found untracked files in %s that would prevent worktree removal (use --force-worktrees to override):\n", repo.Name)
			for _, file := range untrackedFiles {
				fmt.Printf("  - %s\n", file)
			}
			errs = append(errs, fmt.Errorf("untracked files present in %s - use --force-worktrees to override", repo.Name))
			continue
		}

		// Remove worktree using WorktreeManager (no interactive prompt)
		if err := wtm.Remove(ctx, repo.Path, worktreePath, force); err != nil {
			output.LogError(
				fmt.Sprintf("Failed to remove worktree for repository '%s'", repo.Name),
				"Failed to remove worktree",
				"error", err,
				"repo", repo.Name,
				"repoPath", repo.Path,
				"worktree", worktreePath,
			)
			errs = append(errs, errors.Wrapf(err, "failed to remove worktree for %s", repo.Name))
		} else {
			output.LogInfo(
				fmt.Sprintf("Successfully removed worktree for '%s'", repo.Name),
				"Successfully removed worktree",
				"repo", repo.Name,
			)
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

	// Update wsm.json metadata file
	if err := wm.createWorkspaceMetadata(workspace); err != nil {
		output.LogWarn(
			fmt.Sprintf("Failed to update wsm.json metadata file for workspace '%s'", workspace.Name),
			"Failed to update workspace metadata file",
			"workspace", workspace.Name,
			"error", err,
		)
		// Don't fail add operation if metadata file update fails
	}

	// Execute setup scripts for the newly added repository
	if err := wm.executeSetupScriptsForRepo(ctx, workspace, repo); err != nil {
		output.LogWarn(
			fmt.Sprintf("Failed to execute setup scripts for newly added repository '%s'", repo.Name),
			"Setup scripts failed for new repository",
			"workspace", workspace.Name,
			"repo", repo.Name,
			"error", err,
		)
		// Don't fail add operation if setup scripts fail
	}

	fmt.Printf("✓ Successfully added repository '%s' to workspace '%s'\n", repoName, workspaceName)
	return nil
}

// CreateWorktreeForAdd creates a worktree for adding a repository to an existing workspace
func (wm *WorkspaceManager) CreateWorktreeForAdd(ctx context.Context, workspace *Workspace, repo Repository, branch string, forceOverwrite bool) error {
	targetPath := filepath.Join(workspace.Path, repo.Name)
	_, wtm := BuildGitBackends(ctx)

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

	// No specific branch, create worktree from current branch
	if branch == "" {
		return wtm.Add(ctx, repo.Path, "", targetPath, gitclient.WorktreeAddOptions{})
	}

	plan, err := wm.resolveBranchPlan(ctx, repo.Path, branchsvc.BranchResolutionRequest{
		TargetBranch: branchsvc.BranchName(branch),
		BaseBranch:   branchsvc.BranchName(workspace.Branch),
		Remote:       branchsvc.DefaultRemoteName,
		Mode:         branchsvc.ResolutionModeAddRepository,
	})
	if err != nil {
		return errors.Wrap(err, "resolve branch plan")
	}

	opts := gitclient.WorktreeAddOptions{}
	if plan.Strategy == branchsvc.ResolutionStrategyUseLocal {
		if forceOverwrite {
			opts.Overwrite = true
			return wtm.Add(ctx, repo.Path, branch, targetPath, opts)
		}
		return wtm.Add(ctx, repo.Path, branch, targetPath, opts)
	}

	if plan.Strategy == branchsvc.ResolutionStrategyTrackRemote {
		opts.RemoteBranch = plan.RemoteRef
		return wtm.Add(ctx, repo.Path, branch, targetPath, opts)
	}

	if plan.Strategy == branchsvc.ResolutionStrategyCreateFromBase {
		opts.BaseRef = plan.StartPoint
	}
	return wtm.Add(ctx, repo.Path, branch, targetPath, opts)
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

	// Check for untracked files that would preclude removal
	untrackedFiles, err := wm.getUntrackedFiles(ctx, worktreePath)
	if err != nil {
		output.LogWarn(
			fmt.Sprintf("Failed to check for untracked files: %v", err),
			"Unable to check for untracked files",
			"error", err,
		)
	} else if len(untrackedFiles) > 0 && !force {
		fmt.Printf("\n⚠️  Found untracked files that would prevent worktree removal (use --force to override):\n")
		for _, file := range untrackedFiles {
			fmt.Printf("  - %s\n", file)
		}
		return errors.New("untracked files present - use --force to override")
	}

	// Remove worktree using WorktreeManager (no interactive prompt)
	_, wtm := BuildGitBackends(ctx)
	if err := wtm.Remove(ctx, repo.Path, worktreePath, force); err != nil {
		output.LogError(
			fmt.Sprintf("Failed to remove worktree for '%s': %v", repo.Name, err),
			"Failed to remove worktree",
			"error", err,
			"repo", repo.Name,
			"repoPath", repo.Path,
			"worktree", worktreePath,
		)
		return errors.Wrapf(err, "failed to remove worktree")
	}

	output.LogInfo(
		fmt.Sprintf("Successfully removed worktree for '%s'", repo.Name),
		"Successfully removed worktree",
		"repo", repo.Name,
	)

	// Verify worktree was removed
	fmt.Printf("\nVerification: Remaining worktrees for %s:\n", repo.Name)
	_, wtm = BuildGitBackends(ctx)
	if infos, err := wtm.List(ctx, repo.Path); err != nil {
		fmt.Printf("⚠️  Failed to list worktrees: %v\n", err)
	} else {
		if len(infos) == 0 {
			fmt.Printf("(none)\n")
		} else {
			for _, wt := range infos {
				fmt.Printf("- %s (%s)\n", wt.Path, wt.Branch)
			}
		}
	}

	return nil
}

// getUntrackedFiles gets untracked files in a repository path
func (wm *WorkspaceManager) getUntrackedFiles(ctx context.Context, repoPath string) ([]string, error) {
	cmd := exec.CommandContext(ctx, "git", "ls-files", "--others", "--exclude-standard")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	if len(output) == 0 {
		return []string{}, nil
	}

	files := strings.Split(strings.TrimSpace(string(output)), "\n")
	return files, nil
}

// executeSetupScripts executes setup scripts after workspace creation
func (wm *WorkspaceManager) executeSetupScripts(ctx context.Context, workspace *Workspace) error {
	// Prepare environment variables
	env := os.Environ()
	env = append(env, fmt.Sprintf("WSM_WORKSPACE_NAME=%s", workspace.Name))
	env = append(env, fmt.Sprintf("WSM_WORKSPACE_PATH=%s", workspace.Path))
	env = append(env, fmt.Sprintf("WSM_WORKSPACE_BRANCH=%s", workspace.Branch))
	if workspace.BaseBranch != "" {
		env = append(env, fmt.Sprintf("WSM_WORKSPACE_BASE_BRANCH=%s", workspace.BaseBranch))
	}

	// Add repository names as comma-separated list
	repoNames := make([]string, len(workspace.Repositories))
	for i, repo := range workspace.Repositories {
		repoNames[i] = repo.Name
	}
	env = append(env, fmt.Sprintf("WSM_WORKSPACE_REPOS=%s", strings.Join(repoNames, ",")))

	// Execute workspace root setup.sh
	rootSetupScript := filepath.Join(workspace.Path, ".wsm", "setup.sh")
	if err := wm.executeSetupScript(ctx, rootSetupScript, workspace.Path, env); err != nil {
		output.LogWarn(
			fmt.Sprintf("Failed to execute root setup script: %s", rootSetupScript),
			"Root setup script failed",
			"script", rootSetupScript,
			"error", err,
		)
	}

	// Collect and execute setup.d scripts in order across all locations
	setupScripts, err := wm.collectSetupDScripts(workspace)
	if err != nil {
		return errors.Wrap(err, "failed to collect setup.d scripts")
	}

	for _, script := range setupScripts {
		if err := wm.executeSetupScript(ctx, script.Path, script.WorkingDir, env); err != nil {
			output.LogWarn(
				fmt.Sprintf("Failed to execute setup script: %s", script.Path),
				"Setup script failed",
				"script", script.Path,
				"workingDir", script.WorkingDir,
				"error", err,
			)
		}
	}

	return nil
}

// SetupScript represents a setup script with its execution context
type SetupScript struct {
	Path       string
	WorkingDir string
	Name       string
}

// collectSetupDScripts collects all setup.d scripts from workspace root and repository directories
func (wm *WorkspaceManager) collectSetupDScripts(workspace *Workspace) ([]SetupScript, error) {
	var scripts []SetupScript

	// Collect scripts from workspace root .wsm/setup.d/
	rootSetupDir := filepath.Join(workspace.Path, ".wsm", "setup.d")
	rootScripts, err := wm.getSetupDScripts(rootSetupDir, workspace.Path)
	if err != nil {
		output.LogWarn(
			fmt.Sprintf("Failed to read root setup.d directory: %s", rootSetupDir),
			"Failed to read root setup.d directory",
			"dir", rootSetupDir,
			"error", err,
		)
	} else {
		scripts = append(scripts, rootScripts...)
	}

	// Collect scripts from each repository's .wsm/setup.d/
	for _, repo := range workspace.Repositories {
		repoSetupDir := filepath.Join(workspace.Path, repo.Name, ".wsm", "setup.d")
		repoScripts, err := wm.getSetupDScripts(repoSetupDir, filepath.Join(workspace.Path, repo.Name))
		if err != nil {
			output.LogWarn(
				fmt.Sprintf("Failed to read repository setup.d directory: %s", repoSetupDir),
				"Failed to read repository setup.d directory",
				"repo", repo.Name,
				"dir", repoSetupDir,
				"error", err,
			)
		} else {
			scripts = append(scripts, repoScripts...)
		}
	}

	// Sort scripts by name for consistent execution order
	sort.Slice(scripts, func(i, j int) bool {
		return scripts[i].Name < scripts[j].Name
	})

	return scripts, nil
}

// getSetupDScripts gets executable scripts from a setup.d directory
func (wm *WorkspaceManager) getSetupDScripts(setupDir, workingDir string) ([]SetupScript, error) {
	var scripts []SetupScript

	entries, err := os.ReadDir(setupDir)
	if os.IsNotExist(err) {
		return scripts, nil // Directory doesn't exist, not an error
	}
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read setup.d directory: %s", setupDir)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		scriptPath := filepath.Join(setupDir, entry.Name())

		// Check if file is executable
		info, err := entry.Info()
		if err != nil {
			continue
		}

		if info.Mode()&0111 == 0 { // Not executable
			continue
		}

		scripts = append(scripts, SetupScript{
			Path:       scriptPath,
			WorkingDir: workingDir,
			Name:       entry.Name(),
		})
	}

	return scripts, nil
}

// executeSetupScript executes a single setup script
func (wm *WorkspaceManager) executeSetupScript(ctx context.Context, scriptPath, workingDir string, env []string) error {
	// Check if script exists
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		return nil // Script doesn't exist, not an error
	}

	output.PrintInfo("Executing setup script: %s", filepath.Base(scriptPath))

	cmd := exec.CommandContext(ctx, "bash", scriptPath)
	cmd.Dir = workingDir
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return errors.Wrapf(err, "script execution failed: %s", scriptPath)
	}

	output.PrintInfo("Setup script completed: %s", filepath.Base(scriptPath))
	return nil
}

// PreviewSetupScripts shows which setup scripts would be executed in dry-run mode
func (wm *WorkspaceManager) PreviewSetupScripts(workspace *Workspace, stepNum int) error {
	fmt.Printf("  %d. Create workspace metadata file:\n", stepNum)
	fmt.Printf("     - %s/.wsm/wsm.json (JSON with workspace information)\n", workspace.Path)
	fmt.Printf("  %d. Execute setup scripts:\n", stepNum+1)

	// Check for workspace root setup.sh
	rootSetupScript := filepath.Join(workspace.Path, ".wsm", "setup.sh")
	fmt.Printf("     - %s (working dir: %s)\n", rootSetupScript, workspace.Path)

	// Preview setup.d scripts
	setupScripts, err := wm.collectSetupDScriptsPreview(workspace)
	if err != nil {
		return errors.Wrap(err, "failed to collect setup.d scripts for preview")
	}

	if len(setupScripts) > 0 {
		fmt.Printf("     - setup.d scripts (in execution order):\n")
		for _, script := range setupScripts {
			fmt.Printf("       • %s (working dir: %s)\n", script.Path, script.WorkingDir)
		}
	}

	fmt.Printf("     Environment variables:\n")
	fmt.Printf("       WSM_WORKSPACE_NAME=%s\n", workspace.Name)
	fmt.Printf("       WSM_WORKSPACE_PATH=%s\n", workspace.Path)
	fmt.Printf("       WSM_WORKSPACE_BRANCH=%s\n", workspace.Branch)
	if workspace.BaseBranch != "" {
		fmt.Printf("       WSM_WORKSPACE_BASE_BRANCH=%s\n", workspace.BaseBranch)
	}

	repoNames := make([]string, len(workspace.Repositories))
	for i, repo := range workspace.Repositories {
		repoNames[i] = repo.Name
	}
	fmt.Printf("       WSM_WORKSPACE_REPOS=%s\n", strings.Join(repoNames, ","))

	return nil
}

// collectSetupDScriptsPreview is like collectSetupDScripts but works with hypothetical workspace structure
func (wm *WorkspaceManager) collectSetupDScriptsPreview(workspace *Workspace) ([]SetupScript, error) {
	var scripts []SetupScript

	// Add hypothetical scripts from workspace root .wsm/setup.d/
	rootSetupDir := filepath.Join(workspace.Path, ".wsm", "setup.d")
	scripts = append(scripts, SetupScript{
		Path:       filepath.Join(rootSetupDir, "*.sh"),
		WorkingDir: workspace.Path,
		Name:       "00-workspace-setup",
	})

	// Add hypothetical scripts from each repository's .wsm/setup.d/
	for _, repo := range workspace.Repositories {
		repoSetupDir := filepath.Join(workspace.Path, repo.Name, ".wsm", "setup.d")
		scripts = append(scripts, SetupScript{
			Path:       filepath.Join(repoSetupDir, "*.sh"),
			WorkingDir: filepath.Join(workspace.Path, repo.Name),
			Name:       fmt.Sprintf("10-%s-setup", repo.Name),
		})
	}

	// Sort scripts by name for consistent execution order
	sort.Slice(scripts, func(i, j int) bool {
		return scripts[i].Name < scripts[j].Name
	})

	return scripts, nil
}

// WorkspaceMetadata represents the JSON structure for wsm.json
type WorkspaceMetadata struct {
	Name         string               `json:"name"`
	Path         string               `json:"path"`
	Branch       string               `json:"branch"`
	BaseBranch   string               `json:"baseBranch,omitempty"`
	GoWorkspace  bool                 `json:"goWorkspace"`
	AgentMD      string               `json:"agentMD,omitempty"`
	CreatedAt    time.Time            `json:"createdAt"`
	Repositories []RepositoryMetadata `json:"repositories"`
	Environment  map[string]string    `json:"environment"`
}

// RepositoryMetadata represents repository information in the metadata
type RepositoryMetadata struct {
	Name         string   `json:"name"`
	Path         string   `json:"path"`
	Categories   []string `json:"categories"`
	WorktreePath string   `json:"worktreePath"`
}

// createWorkspaceMetadata creates a wsm.json file with workspace metadata
func (wm *WorkspaceManager) createWorkspaceMetadata(workspace *Workspace) error {
	// Create .wsm directory if it doesn't exist
	wsmDir := filepath.Join(workspace.Path, ".wsm")
	if err := os.MkdirAll(wsmDir, 0755); err != nil {
		return errors.Wrapf(err, "failed to create .wsm directory: %s", wsmDir)
	}

	// Prepare repository metadata
	repoMetadata := make([]RepositoryMetadata, len(workspace.Repositories))
	for i, repo := range workspace.Repositories {
		repoMetadata[i] = RepositoryMetadata{
			Name:         repo.Name,
			Path:         repo.Path,
			Categories:   repo.Categories,
			WorktreePath: filepath.Join(workspace.Path, repo.Name),
		}
	}

	// Prepare environment variables
	repoNames := make([]string, len(workspace.Repositories))
	for i, repo := range workspace.Repositories {
		repoNames[i] = repo.Name
	}

	environment := map[string]string{
		"WSM_WORKSPACE_NAME":   workspace.Name,
		"WSM_WORKSPACE_PATH":   workspace.Path,
		"WSM_WORKSPACE_BRANCH": workspace.Branch,
		"WSM_WORKSPACE_REPOS":  strings.Join(repoNames, ","),
	}

	if workspace.BaseBranch != "" {
		environment["WSM_WORKSPACE_BASE_BRANCH"] = workspace.BaseBranch
	}

	// Create metadata structure
	metadata := WorkspaceMetadata{
		Name:         workspace.Name,
		Path:         workspace.Path,
		Branch:       workspace.Branch,
		BaseBranch:   workspace.BaseBranch,
		GoWorkspace:  workspace.GoWorkspace,
		AgentMD:      workspace.AgentMD,
		CreatedAt:    time.Now(),
		Repositories: repoMetadata,
		Environment:  environment,
	}

	// Write JSON file
	metadataPath := filepath.Join(wsmDir, "wsm.json")
	jsonData, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return errors.Wrap(err, "failed to marshal workspace metadata to JSON")
	}

	if err := os.WriteFile(metadataPath, jsonData, 0644); err != nil {
		return errors.Wrapf(err, "failed to write workspace metadata file: %s", metadataPath)
	}

	output.LogInfo(
		fmt.Sprintf("Created workspace metadata file: %s", metadataPath),
		"Created workspace metadata file",
		"metadataPath", metadataPath,
	)

	return nil
}

// executeSetupScriptsForRepo executes setup scripts for a newly added repository
func (wm *WorkspaceManager) executeSetupScriptsForRepo(ctx context.Context, workspace *Workspace, repo Repository) error {
	// Prepare environment variables
	env := os.Environ()
	env = append(env, fmt.Sprintf("WSM_WORKSPACE_NAME=%s", workspace.Name))
	env = append(env, fmt.Sprintf("WSM_WORKSPACE_PATH=%s", workspace.Path))
	env = append(env, fmt.Sprintf("WSM_WORKSPACE_BRANCH=%s", workspace.Branch))
	if workspace.BaseBranch != "" {
		env = append(env, fmt.Sprintf("WSM_WORKSPACE_BASE_BRANCH=%s", workspace.BaseBranch))
	}

	// Add repository names as comma-separated list
	repoNames := make([]string, len(workspace.Repositories))
	for i, r := range workspace.Repositories {
		repoNames[i] = r.Name
	}
	env = append(env, fmt.Sprintf("WSM_WORKSPACE_REPOS=%s", strings.Join(repoNames, ",")))
	env = append(env, fmt.Sprintf("WSM_ADDED_REPO=%s", repo.Name))

	// Execute setup scripts from the newly added repository's .wsm/setup.d/
	repoSetupDir := filepath.Join(workspace.Path, repo.Name, ".wsm", "setup.d")
	repoScripts, err := wm.getSetupDScripts(repoSetupDir, filepath.Join(workspace.Path, repo.Name))
	if err != nil {
		output.LogWarn(
			fmt.Sprintf("Failed to read repository setup.d directory: %s", repoSetupDir),
			"Failed to read repository setup.d directory",
			"repo", repo.Name,
			"dir", repoSetupDir,
			"error", err,
		)
		return nil // Not a critical error
	}

	// Sort scripts by name for consistent execution order
	sort.Slice(repoScripts, func(i, j int) bool {
		return repoScripts[i].Name < repoScripts[j].Name
	})

	if len(repoScripts) > 0 {
		output.PrintInfo("Executing setup scripts for newly added repository: %s", repo.Name)

		for _, script := range repoScripts {
			if err := wm.executeSetupScript(ctx, script.Path, script.WorkingDir, env); err != nil {
				output.LogWarn(
					fmt.Sprintf("Failed to execute setup script: %s", script.Path),
					"Setup script failed",
					"script", script.Path,
					"workingDir", script.WorkingDir,
					"error", err,
				)
			}
		}
	}

	return nil
}
