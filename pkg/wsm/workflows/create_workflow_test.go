package workflows

import "testing"

func TestBuildWorkspaceBranch(t *testing.T) {
	t.Run("explicit branch", func(t *testing.T) {
		branch, auto := BuildWorkspaceBranch("feature-1", "feat/custom", "task")
		if branch != "feat/custom" {
			t.Fatalf("expected explicit branch to be preserved, got %q", branch)
		}
		if auto {
			t.Fatalf("expected auto=false for explicit branch")
		}
	})

	t.Run("generated branch", func(t *testing.T) {
		branch, auto := BuildWorkspaceBranch("feature-1", "", "task")
		if branch != "task/feature-1" {
			t.Fatalf("expected generated branch task/feature-1, got %q", branch)
		}
		if !auto {
			t.Fatalf("expected auto=true for generated branch")
		}
	})

	t.Run("default prefix", func(t *testing.T) {
		branch, _ := BuildWorkspaceBranch("feature-1", "", "")
		if branch != "task/feature-1" {
			t.Fatalf("expected default prefix task, got %q", branch)
		}
	})
}
