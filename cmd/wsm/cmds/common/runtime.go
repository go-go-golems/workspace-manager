package common

import (
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/pkg/errors"
)

// NewStandardSections returns standard sections to compose into command descriptions.
func NewStandardSections() ([]schema.Section, error) {
	glazedSection, err := settings.NewGlazedSchema()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create glazed section")
	}

	return []schema.Section{glazedSection}, nil
}

// BuildDescription creates a command description with standard sections pre-wired.
func BuildDescription(name string, options ...cmds.CommandDescriptionOption) (*cmds.CommandDescription, error) {
	sections, err := NewStandardSections()
	if err != nil {
		return nil, err
	}

	all := make([]cmds.CommandDescriptionOption, 0, len(options)+1)
	all = append(all, cmds.WithSections(sections...))
	all = append(all, options...)
	return cmds.NewCommandDescription(name, all...), nil
}
