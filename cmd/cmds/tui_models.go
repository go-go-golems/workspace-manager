package cmds

import (
	"fmt"
	"github.com/go-go-golems/workspace-manager/pkg/wsm"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

// Application states
type appState int

const (
	stateMain appState = iota
	stateRepositories
	stateWorkspaces
	stateCreateWorkspace
	stateWorkspaceForm
	stateDeleteConfirm
)

// Main TUI model
type mainModel struct {
	state            appState
	discoverer       *wsm.RepositoryDiscoverer
	workspaceManager *wsm.WorkspaceManager
	repositories     []wsm.Repository
	workspaces       []wsm.Workspace

	// UI components
	repoList      list.Model
	workspaceList list.Model
	selectedRepos map[string]bool

	// Workspace creation form
	workspaceName textinput.Model
	branchName    textinput.Model
	agentPath     textinput.Model
	formStep      int

	// Filtering
	tagFilter   string
	searchQuery string

	// UI state
	width    int
	height   int
	showHelp bool
	message  string

	// Delete confirmation
	deleteWorkspace *wsm.Workspace
	deleteFiles     bool
	forceWorktrees  bool

	// Key bindings
	keys keyMap
}

// Key map for the TUI
type keyMap struct {
	Up      key.Binding
	Down    key.Binding
	Left    key.Binding
	Right   key.Binding
	Enter   key.Binding
	Space   key.Binding
	Tab     key.Binding
	Escape  key.Binding
	Quit    key.Binding
	Help    key.Binding
	Create  key.Binding
	Delete  key.Binding
	Refresh key.Binding
	Filter  key.Binding
}

// Default key bindings
func defaultKeyMap() keyMap {
	return keyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "move up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "move down"),
		),
		Left: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("←/h", "back"),
		),
		Right: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("→/l", "forward"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select/confirm"),
		),
		Space: key.NewBinding(
			key.WithKeys(" "),
			key.WithHelp("space", "toggle selection"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next section"),
		),
		Escape: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back/cancel"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "toggle help"),
		),
		Create: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "create workspace"),
		),
		Delete: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "delete workspace"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
		Filter: key.NewBinding(
			key.WithKeys("f"),
			key.WithHelp("f", "filter"),
		),
	}
}

// Repository list item
type repoItem struct {
	repo     wsm.Repository
	selected bool
}

func (r repoItem) Title() string { return r.repo.Name }
func (r repoItem) Description() string {
	desc := fmt.Sprintf("%s [%s]", r.repo.Path, strings.Join(r.repo.Categories, ", "))
	if r.selected {
		desc = "✓ " + desc
	}
	return desc
}
func (r repoItem) FilterValue() string {
	return r.repo.Name + " " + strings.Join(r.repo.Categories, " ")
}

// Workspace list item
type workspaceItem struct {
	workspace wsm.Workspace
}

func (w workspaceItem) Title() string { return w.workspace.Name }
func (w workspaceItem) Description() string {
	repoNames := make([]string, len(w.workspace.Repositories))
	for i, repo := range w.workspace.Repositories {
		repoNames[i] = repo.Name
	}
	return fmt.Sprintf("%s - %s repos", w.workspace.Path, strings.Join(repoNames, ", "))
}
func (w workspaceItem) FilterValue() string { return w.workspace.Name }

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1).
			Margin(1, 0)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7D56F4")).
			Margin(1, 0, 0, 0)

	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7D56F4"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262")).
			Margin(1, 0)

	messageStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#04B575")).
			Margin(1, 0)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF5F56")).
			Margin(1, 0)

	formStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7D56F4")).
			Padding(1).
			Margin(1, 0)
)
