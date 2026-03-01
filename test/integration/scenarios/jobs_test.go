package scenarios

import (
	"testing"

	h "github.com/go-go-golems/workspace-manager/test/integration/helpers"
)

func TestJobsConcurrency(t *testing.T) {
	s := h.NewSandbox(t)
	defer s.Cleanup()

	remote := s.InitBareRepo(t, "remote")
	_ = s.InitRepo(t, "repo1", remote)
	_ = s.InitRepo(t, "repo2", remote)
	_ = s.InitRepo(t, "repo3", remote)

	// Discover and create workspace
	t.Logf("[debug] running discover in %s", s.ReposDir)
	res := s.RunWSM(t, nil, s.ReposDir, "discover", s.ReposDir)
	t.Logf("[debug] discover exit=%d\nstdout:\n%s\nstderr:\n%s", res.ExitCode, res.Stdout, res.Stderr)

	wsName := "ws-jobs"
	res = s.RunWSM(t, nil, "", "create", wsName, "--repos", "repo1,repo2,repo3", "--branch", "feature/jobs")
	if res.ExitCode != 0 {
		t.Fatalf("create failed: %s\n%s", res.Stdout, res.Stderr)
	}
	wsPath := s.LoadWorkspacePath(t, wsName)

	res = s.RunWSM(t, nil, wsPath, "status", "--workspace", wsName, "--jobs", "4")
	if res.ExitCode != 0 {
		t.Fatalf("status with jobs failed: %s\n%s", res.Stdout, res.Stderr)
	}
	res = s.RunWSM(t, nil, wsPath, "diff", "--jobs", "4")
	if res.ExitCode != 0 {
		t.Fatalf("diff with jobs failed: %s\n%s", res.Stdout, res.Stderr)
	}
}
