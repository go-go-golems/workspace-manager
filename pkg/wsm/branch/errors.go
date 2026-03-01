package branch

import "errors"

var (
	ErrInvalidResolutionMode = errors.New("invalid resolution mode")
	ErrEmptyTargetBranch     = errors.New("target branch is required")
)
