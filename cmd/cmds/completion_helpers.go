package cmds

import (
	"github.com/carapace-sh/carapace"
	"github.com/go-go-golems/workspace-manager/pkg/wsm"
)

// WorkspaceNameCompletion returns a carapace.Action that completes workspace names.
func WorkspaceNameCompletion() carapace.Action {
	return carapace.ActionCallback(func(ctx carapace.Context) carapace.Action {
		workspaces, err := wsm.LoadWorkspaces()
		if err != nil {
			return carapace.ActionMessage("failed to load workspaces")
		}
		var names []string
		for _, ws := range workspaces {
			names = append(names, ws.Name)
		}
		return carapace.ActionValues(names...)
	})
}

// RepositoryNameCompletion returns a carapace.Action that completes repository names
// from the registry for add commands.
func RepositoryNameCompletion() carapace.Action {
	return carapace.ActionCallback(func(ctx carapace.Context) carapace.Action {
		registryPath, err := getRegistryPath()
		if err != nil {
			return carapace.ActionMessage("failed to get registry path")
		}
		discoverer := wsm.NewRepositoryDiscoverer(registryPath)
		if err := discoverer.LoadRegistry(); err != nil {
			return carapace.ActionMessage("failed to load registry")
		}
		var names []string
		for _, repo := range discoverer.GetRepositories() {
			names = append(names, repo.Name)
		}
		return carapace.ActionValues(names...)
	})
}

// WorkspaceRepositoryCompletion returns a carapace.Action that completes repository names
// that are currently part of the specified workspace (for remove commands).
func WorkspaceRepositoryCompletion() carapace.Action {
	return carapace.ActionCallback(func(ctx carapace.Context) carapace.Action {
		if len(ctx.Args) < 1 {
			return carapace.ActionMessage("workspace name required")
		}
		workspaceName := ctx.Args[0]

		workspaces, err := wsm.LoadWorkspaces()
		if err != nil {
			return carapace.ActionMessage("failed to load workspaces")
		}

		for _, ws := range workspaces {
			if ws.Name == workspaceName {
				var names []string
				for _, repo := range ws.Repositories {
					names = append(names, repo.Name)
				}
				return carapace.ActionValues(names...)
			}
		}
		return carapace.ActionMessage("workspace not found")
	})
}

// TagCompletion returns a carapace.Action that completes repository tags.
func TagCompletion() carapace.Action {
	return carapace.ActionCallback(func(ctx carapace.Context) carapace.Action {
		registryPath, err := getRegistryPath()
		if err != nil {
			return carapace.ActionMessage("failed to get registry path")
		}
		discoverer := wsm.NewRepositoryDiscoverer(registryPath)
		if err := discoverer.LoadRegistry(); err != nil {
			return carapace.ActionMessage("failed to load registry")
		}
		tagsSet := make(map[string]struct{})
		for _, repo := range discoverer.GetRepositories() {
			for _, tag := range repo.Categories {
				tagsSet[tag] = struct{}{}
			}
		}
		var tags []string
		for tag := range tagsSet {
			tags = append(tags, tag)
		}
		return carapace.ActionValues(tags...)
	})
}
