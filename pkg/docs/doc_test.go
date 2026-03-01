package docs

import (
	"testing"

	"github.com/go-go-golems/glazed/pkg/help"
)

func TestAddDocToHelpSystem_LoadsWSMPages(t *testing.T) {
	hs := help.NewHelpSystem()
	if err := AddDocToHelpSystem(hs); err != nil {
		t.Fatalf("AddDocToHelpSystem failed: %v", err)
	}

	expected := []string{
		"wsm-getting-started",
		"wsm-command-reference",
		"wsm-js-api-and-runner",
		"wsm-architecture-overview",
		"wsm-persistence-and-state",
		"wsm-troubleshooting",
	}

	for _, slug := range expected {
		if _, err := hs.GetSectionWithSlug(slug); err != nil {
			t.Fatalf("expected slug %s to be loaded: %v", slug, err)
		}
	}
}
