package cmds

import (
	"context"
	"fmt"
	"github.com/go-go-golems/workspace-manager/pkg/wsm"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/pkg/errors"
)

// Initialize the main model
func newMainModel() (*mainModel, error) {
	// Load discoverer
	registryPath, err := getRegistryPath()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get registry path")
	}

	discoverer := wsm.NewRepositoryDiscoverer(registryPath)
	if err := discoverer.LoadRegistry(); err != nil {
		return nil, errors.Wrap(err, "failed to load registry")
	}

	// Load workspace manager
	workspaceManager, err := wsm.NewWorkspaceManager()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create workspace manager")
	}

	// Initialize text inputs
	workspaceName := textinput.New()
	workspaceName.Placeholder = "Enter workspace name..."
	workspaceName.CharLimit = 50

	branchName := textinput.New()
	branchName.Placeholder = "Enter branch name (optional)..."
	branchName.CharLimit = 50

	agentPath := textinput.New()
	agentPath.Placeholder = "Path to AGENT.md (optional)..."
	agentPath.CharLimit = 200

	model := &mainModel{
		state:            stateMain,
		discoverer:       discoverer,
		workspaceManager: workspaceManager,
		selectedRepos:    make(map[string]bool),
		workspaceName:    workspaceName,
		branchName:       branchName,
		agentPath:        agentPath,
		keys:             defaultKeyMap(),
	}

	// Initialize lists with default sizes
	model.initLists()
	model.repoList.SetSize(80, 20)
	model.workspaceList.SetSize(80, 20)

	// Load initial data
	model.refreshRepositories()
	model.refreshWorkspaces()

	return model, nil
}

// Initialize the lists
func (m *mainModel) initLists() {
	// Repository list
	repoDelegate := list.NewDefaultDelegate()
	repoDelegate.SetHeight(2)
	m.repoList = list.New([]list.Item{}, repoDelegate, 0, 0)
	m.repoList.Title = "Available Repositories"
	m.repoList.SetShowStatusBar(true)
	m.repoList.SetFilteringEnabled(true)

	// Workspace list
	workspaceDelegate := list.NewDefaultDelegate()
	workspaceDelegate.SetHeight(2)
	m.workspaceList = list.New([]list.Item{}, workspaceDelegate, 0, 0)
	m.workspaceList.Title = "Created Workspaces"
	m.workspaceList.SetShowStatusBar(true)
	m.workspaceList.SetFilteringEnabled(true)
}

// Refresh repositories from registry
func (m *mainModel) refreshRepositories() {
	m.repositories = m.discoverer.GetRepositories()
	items := make([]list.Item, len(m.repositories))
	for i, repo := range m.repositories {
		items[i] = repoItem{
			repo:     repo,
			selected: m.selectedRepos[repo.Name],
		}
	}
	if m.repoList.Items() != nil {
		m.repoList.SetItems(items)
	}
}

// Refresh workspaces
func (m *mainModel) refreshWorkspaces() {
	workspaces, err := wsm.LoadWorkspaces()
	if err != nil {
		m.message = fmt.Sprintf("Error loading workspaces: %v", err)
		return
	}
	m.workspaces = workspaces
	items := make([]list.Item, len(workspaces))
	for i, workspace := range workspaces {
		items[i] = workspaceItem{workspace: workspace}
	}
	if m.workspaceList.Items() != nil {
		m.workspaceList.SetItems(items)
	}
}

// Bubble Tea interface implementation
func (m mainModel) Init() tea.Cmd {
	return tea.EnterAltScreen
}

func (m mainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.initLists()
		m.repoList.SetSize(msg.Width-4, msg.Height-8)
		m.workspaceList.SetSize(msg.Width-4, msg.Height-8)
		m.refreshRepositories()
		m.refreshWorkspaces()

	case tea.KeyMsg:
		// Global key handling
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.keys.Help):
			m.showHelp = !m.showHelp
			return m, nil
		}

		// State-specific key handling
		switch m.state {
		case stateMain:
			return m.updateMain(msg)
		case stateRepositories:
			return m.updateRepositories(msg)
		case stateWorkspaces:
			return m.updateWorkspaces(msg)
		case stateCreateWorkspace:
			return m.updateCreateWorkspace(msg)
		case stateWorkspaceForm:
			return m.updateWorkspaceForm(msg)
		case stateDeleteConfirm:
			return m.updateDeleteConfirm(msg)
		}
	}

	// Update active components
	switch m.state {
	case stateRepositories:
		m.repoList, cmd = m.repoList.Update(msg)
		cmds = append(cmds, cmd)
	case stateWorkspaces:
		m.workspaceList, cmd = m.workspaceList.Update(msg)
		cmds = append(cmds, cmd)
	case stateWorkspaceForm:
		switch m.formStep {
		case 0:
			m.workspaceName, cmd = m.workspaceName.Update(msg)
		case 1:
			m.branchName, cmd = m.branchName.Update(msg)
		case 2:
			m.agentPath, cmd = m.agentPath.Update(msg)
		}
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m mainModel) View() string {
	if m.width == 0 {
		return "Initializing..."
	}

	var content string

	switch m.state {
	case stateMain:
		content = m.viewMain()
	case stateRepositories:
		content = m.viewRepositories()
	case stateWorkspaces:
		content = m.viewWorkspaces()
	case stateCreateWorkspace:
		content = m.viewCreateWorkspace()
	case stateWorkspaceForm:
		content = m.viewWorkspaceForm()
	case stateDeleteConfirm:
		content = m.viewDeleteConfirm()
	}

	// Add message if any
	if m.message != "" {
		content += "\n" + messageStyle.Render(m.message)
	}

	// Add help if enabled
	if m.showHelp {
		content += "\n" + m.viewHelp()
	}

	return content
}

func (m mainModel) updateMain(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Enter):
		// Default to repositories view
		m.state = stateRepositories
		return m, nil
	case msg.String() == "1":
		m.state = stateRepositories
		return m, nil
	case msg.String() == "2":
		m.state = stateWorkspaces
		return m, nil
	case msg.String() == "3":
		m.state = stateCreateWorkspace
		return m, nil
	case key.Matches(msg, m.keys.Refresh):
		m.refreshRepositories()
		m.refreshWorkspaces()
		m.message = "Data refreshed"
		return m, nil
	}
	return m, nil
}

func (m mainModel) updateRepositories(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Escape):
		m.state = stateMain
		return m, nil
	case key.Matches(msg, m.keys.Space):
		// Toggle repository selection
		if selected := m.repoList.SelectedItem(); selected != nil {
			if item, ok := selected.(repoItem); ok {
				m.selectedRepos[item.repo.Name] = !m.selectedRepos[item.repo.Name]
				m.refreshRepositories()
			}
		}
		return m, nil
	case key.Matches(msg, m.keys.Create):
		if len(m.selectedRepos) > 0 {
			m.state = stateWorkspaceForm
			m.formStep = 0
			m.workspaceName.Focus()
		} else {
			m.message = "Select repositories first (use space to toggle)"
		}
		return m, nil
	case key.Matches(msg, m.keys.Refresh):
		m.refreshRepositories()
		m.message = "Repositories refreshed"
		return m, nil
	}
	return m, nil
}

func (m mainModel) updateWorkspaces(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Escape):
		m.state = stateMain
		return m, nil
	case key.Matches(msg, m.keys.Delete):
		// Delete selected workspace
		if selected := m.workspaceList.SelectedItem(); selected != nil {
			if item, ok := selected.(workspaceItem); ok {
				m.deleteWorkspace = &item.workspace
				m.deleteFiles = false
				m.forceWorktrees = false
				m.state = stateDeleteConfirm
			}
		}
		return m, nil
	case key.Matches(msg, m.keys.Refresh):
		m.refreshWorkspaces()
		m.message = "Workspaces refreshed"
		return m, nil
	}
	return m, nil
}

func (m mainModel) updateCreateWorkspace(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Escape):
		m.state = stateMain
		return m, nil
	case key.Matches(msg, m.keys.Enter):
		m.state = stateWorkspaceForm
		m.formStep = 0
		m.workspaceName.Focus()
		return m, nil
	}
	return m, nil
}

func (m mainModel) updateWorkspaceForm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Escape):
		m.state = stateCreateWorkspace
		m.workspaceName.Blur()
		m.branchName.Blur()
		m.agentPath.Blur()
		return m, nil
	case key.Matches(msg, m.keys.Tab):
		// Move to next form field
		m.nextFormField()
		return m, nil
	case key.Matches(msg, m.keys.Enter):
		if m.formStep < 2 {
			m.nextFormField()
		} else {
			// Submit form
			return m.submitWorkspaceForm()
		}
		return m, nil
	}
	return m, nil
}

func (m *mainModel) nextFormField() {
	switch m.formStep {
	case 0:
		m.workspaceName.Blur()
		m.branchName.Focus()
		m.formStep = 1
	case 1:
		m.branchName.Blur()
		m.agentPath.Focus()
		m.formStep = 2
	case 2:
		m.agentPath.Blur()
		m.workspaceName.Focus()
		m.formStep = 0
	}
}

func (m mainModel) submitWorkspaceForm() (tea.Model, tea.Cmd) {
	// Get selected repository names
	var selectedRepoNames []string
	for name, selected := range m.selectedRepos {
		if selected {
			selectedRepoNames = append(selectedRepoNames, name)
		}
	}

	if len(selectedRepoNames) == 0 {
		m.message = "No repositories selected"
		return m, nil
	}

	// Create workspace
	ctx := context.Background()
	workspace, err := m.workspaceManager.CreateWorkspace(
		ctx,
		m.workspaceName.Value(),
		selectedRepoNames,
		m.branchName.Value(),
		m.agentPath.Value(),
		false, // not dry run
	)

	if err != nil {
		m.message = fmt.Sprintf("Error creating workspace: %v", err)
		return m, nil
	}

	// Reset form and go back to main
	m.workspaceName.SetValue("")
	m.branchName.SetValue("")
	m.agentPath.SetValue("")
	m.formStep = 0
	m.selectedRepos = make(map[string]bool)
	m.state = stateMain
	m.refreshWorkspaces()
	m.refreshRepositories()
	m.message = fmt.Sprintf("✅ Workspace '%s' created successfully!", workspace.Name)

	return m, nil
}

// View functions
func (m mainModel) viewMain() string {
	title := titleStyle.Render("🔧 Workspace Manager")

	content := fmt.Sprintf(`%s

Welcome to Workspace Manager! Choose an option:

1. 📁 Browse Repositories (%d found)
2. 🏗️  Manage Workspaces (%d created)
3. ➕ Create New Workspace

Press number key or enter to continue
Press ? for help, q to quit
`, title, len(m.repositories), len(m.workspaces))

	return content
}

func (m mainModel) viewRepositories() string {
	selectedCount := 0
	for _, selected := range m.selectedRepos {
		if selected {
			selectedCount++
		}
	}

	header := headerStyle.Render(fmt.Sprintf("📁 Repositories (%d selected)", selectedCount))
	help := helpStyle.Render("Space: toggle selection • c: create workspace • r: refresh • esc: back")

	return fmt.Sprintf("%s\n\n%s\n\n%s", header, m.repoList.View(), help)
}

func (m mainModel) viewWorkspaces() string {
	header := headerStyle.Render("🏗️  Workspaces")
	help := helpStyle.Render("r: refresh • esc: back")

	return fmt.Sprintf("%s\n\n%s\n\n%s", header, m.workspaceList.View(), help)
}

func (m mainModel) viewCreateWorkspace() string {
	header := headerStyle.Render("➕ Create New Workspace")

	content := `
First, go to the Repositories view and select the repositories you want to include.
Then return here to configure the workspace.

Press enter to continue to workspace form
Press esc to go back
`

	return fmt.Sprintf("%s\n%s", header, content)
}

func (m mainModel) viewWorkspaceForm() string {
	header := headerStyle.Render("📝 Workspace Configuration")

	selectedRepos := []string{}
	for name, selected := range m.selectedRepos {
		if selected {
			selectedRepos = append(selectedRepos, name)
		}
	}

	form := fmt.Sprintf(`
Selected Repositories: %s

%s
%s

%s
%s

%s
%s

Press tab to move between fields
Press enter to create workspace
Press esc to cancel
`,
		strings.Join(selectedRepos, ", "),
		"Workspace Name:",
		m.workspaceName.View(),
		"Branch Name (optional):",
		m.branchName.View(),
		"AGENT.md Path (optional):",
		m.agentPath.View(),
	)

	return fmt.Sprintf("%s\n%s", header, formStyle.Render(form))
}

func (m mainModel) viewHelp() string {
	help := `
Key Bindings:
  ↑/k, ↓/j    Navigate up/down
  ←/h, →/l    Navigate left/right
  enter       Select/confirm
  space       Toggle selection (in repository view)
  tab         Next form field
  esc         Back/cancel
  c           Create workspace (in repository view)
  d           Delete workspace (in workspace view)
  w           Toggle force worktrees (in delete confirmation)
  r           Refresh data
  ?           Toggle this help
  q           Quit

Navigation:
  Main Menu   → Choose between repositories, workspaces, or create
  Repositories→ Browse and select repositories for workspace
  Workspaces  → View created workspaces
  Create      → Configure and create new workspace
`
	return helpStyle.Render(help)
}

func (m mainModel) updateDeleteConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Escape):
		m.state = stateWorkspaces
		m.deleteWorkspace = nil
		return m, nil
	case msg.String() == "y" || msg.String() == "Y":
		// Confirm delete
		if m.deleteWorkspace != nil {
			if err := m.workspaceManager.DeleteWorkspace(context.Background(), m.deleteWorkspace.Name, m.deleteFiles, m.forceWorktrees); err != nil {
				m.message = fmt.Sprintf("Failed to delete workspace: %v", err)
			} else {
				if m.deleteFiles {
					m.message = fmt.Sprintf("Workspace '%s' and files deleted successfully", m.deleteWorkspace.Name)
				} else {
					m.message = fmt.Sprintf("Workspace configuration '%s' deleted successfully", m.deleteWorkspace.Name)
				}
				m.refreshWorkspaces()
			}
		}
		m.state = stateWorkspaces
		m.deleteWorkspace = nil
		return m, nil
	case msg.String() == "n" || msg.String() == "N":
		// Cancel delete
		m.state = stateWorkspaces
		m.deleteWorkspace = nil
		return m, nil
	case msg.String() == "f" || msg.String() == "F":
		// Toggle file deletion
		m.deleteFiles = !m.deleteFiles
		return m, nil
	case msg.String() == "w" || msg.String() == "W":
		// Toggle force worktrees
		m.forceWorktrees = !m.forceWorktrees
		return m, nil
	}
	return m, nil
}

func (m mainModel) viewDeleteConfirm() string {
	if m.deleteWorkspace == nil {
		return "Error: No workspace selected for deletion"
	}

	content := titleStyle.Render("Delete Workspace Confirmation")
	content += "\n\n"

	content += fmt.Sprintf("Workspace: %s\n", selectedStyle.Render(m.deleteWorkspace.Name))
	content += fmt.Sprintf("Path: %s\n", m.deleteWorkspace.Path)
	content += fmt.Sprintf("Repositories: %d\n", len(m.deleteWorkspace.Repositories))

	content += "\n"

	content += "This will:\n"
	if m.forceWorktrees {
		content += "  1. Remove git worktrees (git worktree remove --force)\n"
	} else {
		content += "  1. Remove git worktrees (git worktree remove)\n"
		content += "     ⚠️  Will fail if there are uncommitted changes\n"
	}

	if m.deleteFiles {
		content += "  2. DELETE the workspace directory and ALL its contents!\n"
	} else {
		content += "  2. Remove workspace configuration only\n"
		content += "  3. Workspace files will remain at: " + m.deleteWorkspace.Path + "\n"
	}

	content += "\n\n"
	content += "Options:\n"
	content += fmt.Sprintf("  [f] Toggle file deletion (currently: %s)\n",
		map[bool]string{true: "ON - will delete files", false: "OFF - keep files"}[m.deleteFiles])
	content += fmt.Sprintf("  [w] Toggle force worktrees (currently: %s)\n",
		map[bool]string{true: "ON - will force removal", false: "OFF - safe removal"}[m.forceWorktrees])
	content += "  [y] Confirm deletion\n"
	content += "  [n] Cancel\n"
	content += "  [esc] Cancel\n"

	return formStyle.Render(content)
}
