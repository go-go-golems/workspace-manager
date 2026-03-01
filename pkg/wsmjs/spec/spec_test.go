package spec

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestGeneratedDeclarationMatchesTemplate(t *testing.T) {
	tmpl, err := os.ReadFile("wsm.d.ts.tmpl")
	if err != nil {
		t.Fatalf("read template: %v", err)
	}

	generated, err := os.ReadFile("wsm.d.ts")
	if err != nil {
		t.Fatalf("read generated declaration: %v", err)
	}

	if !bytes.Equal(tmpl, generated) {
		t.Fatalf("generated declaration is out of sync; run `go generate ./pkg/wsmjs/spec`")
	}
}

func TestTemplateContainsCompletionSurface(t *testing.T) {
	data, err := os.ReadFile("wsm.d.ts.tmpl")
	if err != nil {
		t.Fatalf("read template: %v", err)
	}
	content := string(data)

	required := []string{
		`createManager(options?: ManagerOptions): Manager;`,
		`loadWorkspace(name: string): WorkspaceHandle;`,
		`listWorkspaces(): any[];`,
		`workspaces: WorkspacesNamespace;`,
		`git: GitNamespace;`,
		`branch: BranchNamespace;`,
		`rebase: RebaseNamespace;`,
		`"continue"(input?: RebaseActionInput): any;`,
		`addRepository(input: Omit<AddRepositoryInput, "workspaceName">): any;`,
		`mergeWorkspace(input: MergeWorkspaceInput): any;`,
	}

	for _, fragment := range required {
		if !strings.Contains(content, fragment) {
			t.Fatalf("template missing required fragment: %s", fragment)
		}
	}
}
