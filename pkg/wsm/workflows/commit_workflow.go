package workflows

import (
	"context"
	"os"

	"github.com/go-go-golems/workspace-manager/pkg/wsm"
	"github.com/pkg/errors"
)

// CommitRequest captures commit options shared by command and workflow.
type CommitRequest struct {
	Message  string
	Template string
	AddAll   bool
	Push     bool
	DryRun   bool
}

// CommitPreparation captures resolved context before interactive selection/rendering.
type CommitPreparation struct {
	Workspace *wsm.Workspace
	Changes   map[string][]wsm.FileChange
	Message   string
}

// CommitWorkflow orchestrates workspace-wide commit execution.
type CommitWorkflow struct {
	workspaceContext *wsm.WorkspaceContextService
}

// NewCommitWorkflow creates a commit workflow service.
func NewCommitWorkflow() *CommitWorkflow {
	return &CommitWorkflow{workspaceContext: wsm.NewWorkspaceContextService()}
}

// Prepare resolves current workspace, loads changes, and resolves template-driven messages.
func (cw *CommitWorkflow) Prepare(ctx context.Context, req CommitRequest) (*CommitPreparation, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get current directory")
	}

	workspace, err := cw.workspaceContext.DetectCurrentWorkspace(cwd)
	if err != nil {
		return nil, errors.Wrap(err, "failed to detect current workspace")
	}

	gitOps := wsm.NewGitOperations(workspace)
	allChanges, err := gitOps.GetWorkspaceChanges(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get workspace changes")
	}

	message := req.Message
	if message == "" && req.Template != "" {
		message = ResolveCommitTemplate(req.Template)
	}

	return &CommitPreparation{
		Workspace: workspace,
		Changes:   allChanges,
		Message:   message,
	}, nil
}

// Execute commits the selected changes.
func (cw *CommitWorkflow) Execute(ctx context.Context, prep *CommitPreparation, selectedChanges map[string][]wsm.FileChange, req CommitRequest) error {
	if prep == nil {
		return errors.New("commit preparation is required")
	}
	if len(selectedChanges) == 0 {
		return errors.New("no files selected for commit")
	}
	if prep.Message == "" {
		return errors.New("commit message is required")
	}

	gitOps := wsm.NewGitOperations(prep.Workspace)
	operation := &wsm.CommitOperation{
		Message: prep.Message,
		Files:   selectedChanges,
		DryRun:  req.DryRun,
		AddAll:  req.AddAll,
		Push:    req.Push,
	}

	if err := gitOps.CommitChanges(ctx, operation); err != nil {
		return errors.Wrap(err, "commit failed")
	}

	return nil
}

// ResolveCommitTemplate resolves named templates, falling back to raw input.
func ResolveCommitTemplate(template string) string {
	templates := map[string]string{
		"feature":  "feat: add new feature",
		"fix":      "fix: resolve issue",
		"docs":     "docs: update documentation",
		"style":    "style: formatting changes",
		"refactor": "refactor: code restructuring",
		"test":     "test: add or update tests",
		"chore":    "chore: maintenance tasks",
	}

	if msg, exists := templates[template]; exists {
		return msg
	}

	return template
}
