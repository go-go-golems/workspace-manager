package common

import (
	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/spf13/cobra"
)

// BuildCobraCommand builds a Cobra command with consistent parser config.
func BuildCobraCommand(command cmds.Command) (*cobra.Command, error) {
	return cli.BuildCobraCommandFromCommand(command,
		cli.WithParserConfig(cli.CobraParserConfig{
			ShortHelpSections: []string{schema.DefaultSlug, RuntimeSectionSlug},
			MiddlewaresFunc:   cli.CobraCommandDefaultMiddlewares,
		}),
	)
}
