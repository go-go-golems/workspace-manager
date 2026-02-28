package branch

func validateRequest(req BranchResolutionRequest) error {
	if req.Mode == ResolutionModeUnspecified {
		return ErrInvalidResolutionMode
	}
	if req.TargetBranch == "" {
		return ErrEmptyTargetBranch
	}
	return nil
}

func normalizeRemote(reqRemote, defaultRemote RemoteName) RemoteName {
	if reqRemote != "" {
		return reqRemote
	}
	if defaultRemote != "" {
		return defaultRemote
	}
	return DefaultRemoteName
}

func resolveFromState(req BranchResolutionRequest, defaultRemote RemoteName, localExists bool, remoteTrackingExists bool) (*BranchResolutionPlan, error) {
	if err := validateRequest(req); err != nil {
		return nil, err
	}

	remote := normalizeRemote(req.Remote, defaultRemote)
	plan := &BranchResolutionPlan{
		Mode:                 req.Mode,
		Remote:               remote,
		TargetBranch:         req.TargetBranch,
		LocalExists:          localExists,
		RemoteTrackingExists: remoteTrackingExists,
		RemoteRefKind:        RemoteRefKindNone,
	}

	switch {
	case localExists:
		plan.Strategy = ResolutionStrategyUseLocal
		return plan, nil
	case remoteTrackingExists:
		plan.Strategy = ResolutionStrategyTrackRemote
		plan.RemoteRefKind = RemoteRefKindRemoteTrackingBranch
		plan.RemoteRef = RemoteTrackingRef(remote, req.TargetBranch)
		return plan, nil
	case req.BaseBranch != "":
		plan.Strategy = ResolutionStrategyCreateFromBase
		plan.StartPoint = string(req.BaseBranch)
		return plan, nil
	default:
		plan.Strategy = ResolutionStrategyCreateFromHead
		return plan, nil
	}
}
