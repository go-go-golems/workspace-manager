package docs

import (
	"embed"

	"github.com/go-go-golems/glazed/pkg/help"
)

//go:embed *.md
var docFS embed.FS

// AddDocToHelpSystem loads embedded markdown help pages.
func AddDocToHelpSystem(helpSystem *help.HelpSystem) error {
	return helpSystem.LoadSectionsFromFS(docFS, ".")
}
