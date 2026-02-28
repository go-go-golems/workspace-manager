package common

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/go-go-golems/glazed/pkg/types"
	"github.com/pkg/errors"
)

const RuntimeSectionSlug = "runtime"

const (
	OutputModeHuman = "human"
	OutputModeData  = "data"
	OutputModeBoth  = "both"
)

// RuntimeSettings defines shared command runtime behavior.
type RuntimeSettings struct {
	OutputMode string `glazed:"output-mode"`
}

// NewRuntimeSection creates a minimal shared runtime section for command behavior.
func NewRuntimeSection() (schema.Section, error) {
	return schema.NewSection(
		RuntimeSectionSlug,
		"Runtime behavior",
		schema.WithFields(
			fields.New(
				"output-mode",
				fields.TypeChoice,
				fields.WithDefault(OutputModeHuman),
				fields.WithChoices(OutputModeHuman, OutputModeData, OutputModeBoth),
				fields.WithHelp("Output mode: human (default), data, or both (human then data)"),
			),
		),
	)
}

// NewStandardSections returns standard sections to compose into command descriptions.
func NewStandardSections() ([]schema.Section, error) {
	glazedSection, err := settings.NewGlazedSchema()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create glazed section")
	}

	runtimeSection, err := NewRuntimeSection()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create runtime section")
	}

	return []schema.Section{glazedSection, runtimeSection}, nil
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

// ResolveOutputMode decodes runtime settings and returns a validated output mode.
func ResolveOutputMode(vals *values.Values) string {
	rs := &RuntimeSettings{OutputMode: OutputModeHuman}
	if err := vals.DecodeSectionInto(RuntimeSectionSlug, rs); err != nil {
		return OutputModeHuman
	}

	switch strings.ToLower(rs.OutputMode) {
	case OutputModeData:
		return OutputModeData
	case OutputModeBoth:
		return OutputModeBoth
	default:
		return OutputModeHuman
	}
}

// EmitRows emits structured rows using glazed output settings.
func EmitRows(ctx context.Context, vals *values.Values, rows []types.Row) error {
	if len(rows) == 0 {
		return nil
	}

	glazedValues, ok := vals.Get(settings.GlazedSlug)
	if !ok {
		return errors.New("glazed section not found")
	}

	gp, err := settings.SetupTableProcessor(glazedValues)
	if err != nil {
		return errors.Wrap(err, "failed to setup table processor")
	}
	if _, err := settings.SetupProcessorOutput(gp, glazedValues, os.Stdout); err != nil {
		return errors.Wrap(err, "failed to setup processor output")
	}

	for _, row := range rows {
		if err := gp.AddRow(ctx, row); err != nil {
			return errors.Wrap(err, "failed to add row")
		}
	}

	if err := gp.Close(ctx); err != nil {
		return errors.Wrap(err, "failed to close processor")
	}

	return nil
}

// ShouldOutputHuman reports whether human-oriented output should be produced.
func ShouldOutputHuman(mode string) bool {
	return mode == OutputModeHuman || mode == OutputModeBoth
}

// ShouldOutputData reports whether structured output should be produced.
func ShouldOutputData(mode string) bool {
	return mode == OutputModeData || mode == OutputModeBoth
}

// ErrUnsupportedOutputMode returns a normalized error for bad output-mode values.
func ErrUnsupportedOutputMode(mode string) error {
	return fmt.Errorf("unsupported output-mode %q (expected one of: human, data, both)", mode)
}
