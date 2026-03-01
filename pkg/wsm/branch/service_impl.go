package branch

import (
	"context"

	"github.com/go-go-golems/workspace-manager/pkg/wsm/gitclient"
	"github.com/pkg/errors"
)

type service struct {
	client        gitclient.GitClient
	defaultRemote RemoteName
}

// NewService creates a new branch service using the provided git backend.
func NewService(client gitclient.GitClient, defaultRemote RemoteName) Service {
	if defaultRemote == "" {
		defaultRemote = DefaultRemoteName
	}
	return &service{client: client, defaultRemote: defaultRemote}
}

func (s *service) open(ctx context.Context, repoPath string) (gitclient.RepositoryHandle, error) {
	h, err := s.client.Open(ctx, repoPath)
	if err != nil {
		return nil, errors.Wrap(err, "open repository")
	}
	return h, nil
}

func (s *service) Snapshot(ctx context.Context, repoPath string) (*BranchSnapshot, error) {
	h, err := s.open(ctx, repoPath)
	if err != nil {
		return nil, err
	}

	locals, err := s.client.ListLocalBranches(ctx, h)
	if err != nil {
		return nil, errors.Wrap(err, "list local branches")
	}
	remoteTracking, err := s.client.ListRemoteTrackingBranches(ctx, h, string(s.defaultRemote))
	if err != nil {
		return nil, errors.Wrap(err, "list remote tracking branches")
	}
	current, err := s.client.CurrentBranch(ctx, h)
	if err != nil {
		return nil, errors.Wrap(err, "current branch")
	}

	localNames := make([]BranchName, 0, len(locals))
	for _, b := range locals {
		localNames = append(localNames, BranchName(b))
	}
	remoteNames := make([]BranchName, 0, len(remoteTracking))
	for _, b := range remoteTracking {
		remoteNames = append(remoteNames, BranchName(b))
	}

	return &BranchSnapshot{
		CurrentBranch: BranchName(current),
		LocalBranches: localNames,
		RemoteTrackingBranches: map[RemoteName][]BranchName{
			s.defaultRemote: remoteNames,
		},
	}, nil
}

func (s *service) Resolve(ctx context.Context, repoPath string, req BranchResolutionRequest) (*BranchResolutionPlan, error) {
	h, err := s.open(ctx, repoPath)
	if err != nil {
		return nil, err
	}

	localExists, err := s.client.LocalBranchExists(ctx, h, string(req.TargetBranch))
	if err != nil {
		return nil, errors.Wrap(err, "check local branch existence")
	}
	remote := normalizeRemote(req.Remote, s.defaultRemote)
	remoteExists, err := s.client.RemoteTrackingBranchExists(ctx, h, string(remote), string(req.TargetBranch))
	if err != nil {
		return nil, errors.Wrap(err, "check remote tracking branch existence")
	}

	plan, err := resolveFromState(req, s.defaultRemote, localExists, remoteExists)
	if err != nil {
		return nil, err
	}
	return plan, nil
}

func (s *service) LocalExists(ctx context.Context, repoPath string, branch BranchName) (bool, error) {
	h, err := s.open(ctx, repoPath)
	if err != nil {
		return false, err
	}
	ok, err := s.client.LocalBranchExists(ctx, h, string(branch))
	if err != nil {
		return false, errors.Wrap(err, "check local branch existence")
	}
	return ok, nil
}

func (s *service) RemoteTrackingExists(ctx context.Context, repoPath string, remote RemoteName, branch BranchName) (bool, error) {
	h, err := s.open(ctx, repoPath)
	if err != nil {
		return false, err
	}
	r := normalizeRemote(remote, s.defaultRemote)
	ok, err := s.client.RemoteTrackingBranchExists(ctx, h, string(r), string(branch))
	if err != nil {
		return false, errors.Wrap(err, "check remote tracking branch existence")
	}
	return ok, nil
}

func (s *service) ListLocal(ctx context.Context, repoPath string) ([]BranchName, error) {
	h, err := s.open(ctx, repoPath)
	if err != nil {
		return nil, err
	}
	locals, err := s.client.ListLocalBranches(ctx, h)
	if err != nil {
		return nil, errors.Wrap(err, "list local branches")
	}
	out := make([]BranchName, 0, len(locals))
	for _, b := range locals {
		out = append(out, BranchName(b))
	}
	return out, nil
}

func (s *service) ListRemoteTracking(ctx context.Context, repoPath string, remote RemoteName) ([]BranchName, error) {
	h, err := s.open(ctx, repoPath)
	if err != nil {
		return nil, err
	}
	r := normalizeRemote(remote, s.defaultRemote)
	branches, err := s.client.ListRemoteTrackingBranches(ctx, h, string(r))
	if err != nil {
		return nil, errors.Wrap(err, "list remote tracking branches")
	}
	out := make([]BranchName, 0, len(branches))
	for _, b := range branches {
		out = append(out, BranchName(b))
	}
	return out, nil
}
