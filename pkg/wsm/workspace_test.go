package wsm

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateGoWorkspaceUsesPatchLevelGoDirective(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skipf("go not found: %v", err)
	}

	wm := &WorkspaceManager{}
	goVersion, err := wm.getGoVersion(context.Background())
	if err != nil {
		t.Fatalf("get Go version: %v", err)
	}
	if len(strings.Split(goVersion, ".")) < 3 {
		t.Skipf("current Go version has no patch component: %s", goVersion)
	}

	workspaceDir := t.TempDir()
	repoDir := filepath.Join(workspaceDir, "module-a")
	if err := os.Mkdir(repoDir, 0755); err != nil {
		t.Fatalf("create repo dir: %v", err)
	}

	goMod := "module example.com/module-a\n\ngo " + goVersion + "\n"
	if err := os.WriteFile(filepath.Join(repoDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoDir, "main.go"), []byte("package main\n"), 0644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}

	workspace := &Workspace{
		Path: workspaceDir,
		Repositories: []Repository{
			{Name: "module-a"},
		},
	}
	if err := wm.CreateGoWorkspace(workspace); err != nil {
		t.Fatalf("create go.work: %v", err)
	}

	goWork, err := os.ReadFile(filepath.Join(workspaceDir, "go.work"))
	if err != nil {
		t.Fatalf("read go.work: %v", err)
	}
	if !strings.Contains(string(goWork), "go "+goVersion) {
		t.Fatalf("go.work does not preserve patch-level Go version %s:\n%s", goVersion, goWork)
	}

	cmd := exec.Command("go", "list", "./...")
	cmd.Dir = repoDir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go list should succeed with generated go.work: %v\n%s", err, output)
	}
}
