package workflows

import "testing"

func TestResolveCommitTemplate(t *testing.T) {
	if got := ResolveCommitTemplate("feature"); got != "feat: add new feature" {
		t.Fatalf("expected feature template expansion, got %q", got)
	}

	if got := ResolveCommitTemplate("custom: message"); got != "custom: message" {
		t.Fatalf("expected passthrough template, got %q", got)
	}
}
