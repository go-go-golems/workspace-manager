---
Title: WSM Troubleshooting Guide
Slug: wsm-troubleshooting
Short: Diagnose common workspace, git, and runner issues quickly.
Topics:
- workspace-manager
- troubleshooting
- operations
Commands:
- status
- rebase
- delete
- runner
Flags:
- --workspace
- --jobs
- --output-mode
- --force-worktrees
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

When a WSM command fails or produces unexpected output, the cause almost always
falls into one of four categories: registry problems, workspace state issues,
git conflicts, or command context errors. This guide helps you identify which
category you are in and recover quickly.

## Quick triage

Before diving into specific errors, run through this checklist:

1. **Where am I?** Run `pwd` and confirm you are inside a workspace directory
   (or pass `--workspace <name>` explicitly).
2. **Is the registry current?** Run `wsm list repos` -- does it include the
   repositories you expect?
3. **Does the workspace exist?** Run `wsm list workspaces` -- is your workspace
   listed?
4. **What does status say?** Run `wsm status --workspace <name> --output-mode both`
   for the fullest picture.

## Registry issues

### "repositories not found" during create

The repositories you specified with `--repos` are not in WSM's registry.

**Diagnose**:
```bash
wsm list repos --output-mode data
```

Look for the missing repository names. If they are absent:

**Fix**:
```bash
wsm discover ~/code  # point at the parent directory containing your repos
wsm list repos       # confirm they appear now
wsm create my-feature --repos the-missing-repo
```

### Registry seems stale after moving repositories

WSM's registry stores absolute paths. If you move a repository to a different
directory, the old path in the registry becomes invalid.

**Fix**: Re-run discovery. It overwrites the registry with fresh data:
```bash
wsm discover ~/new-location
```

## Workspace issues

### "not in a workspace directory"

WSM cannot detect which workspace you mean. This happens when:
- You are outside any known workspace directory.
- Your working directory is not a subdirectory of a workspace path.

**Fix**: Pass the workspace explicitly:
```bash
wsm status my-feature
# or
wsm status --workspace my-feature
```

### Workspace in list but directory is gone

Someone (or you) deleted the workspace directory manually, but the JSON config
file at `~/.config/workspace-manager/workspaces/<name>.json` still exists.

**Fix**:
```bash
wsm delete <name>  # removes the stale JSON config
```

If `wsm delete` fails because it cannot find the directory, delete the JSON file
manually:
```bash
rm ~/.config/workspace-manager/workspaces/<name>.json
```

### Create succeeds but workspace looks wrong

Check the workspace JSON:
```bash
wsm info <name>
wsm info <name> --field path
```

If the path or repos look incorrect, delete and recreate:
```bash
wsm delete <name>
wsm create <name> --repos correct,repos
```

## Rebase issues

Rebase is the most complex multi-repo operation. Problems usually affect one
repository at a time.

### Rebase fails for a single repo

The other repositories may have rebased successfully. Check per-repo state:

```bash
wsm rebase status
```

This shows a table with columns: REPOSITORY, STATE, CONFLICTS, ERROR. Possible
states:
- `none` -- no rebase in progress
- `in_progress` -- rebase is paused (usually due to conflicts)

### Recovering from rebase conflicts

Step-by-step recovery:

1. **Identify conflicts**:
   ```bash
   wsm rebase status --repo <repo-name>
   ```

2. **Resolve conflicts manually**:
   ```bash
   cd <workspace-path>/<repo-name>
   # edit the conflicted files
   git add <resolved-files>
   ```

3. **Continue the rebase**:
   ```bash
   wsm rebase continue --repo <repo-name>
   ```

4. **Or abort if recovery is not worth it**:
   ```bash
   wsm rebase abort --repo <repo-name>
   ```

### Rebase hangs or seems stuck

If a rebase operation appears to hang, it may be waiting for an editor (for
interactive rebase) or processing a large repository.

- If you used `--interactive`, WSM opens an editor. Complete the editor session.
- Try running with `--manual` to see the exact git commands, then run them yourself:
  ```bash
  wsm rebase --manual --target main
  ```

### Want to preview before rebasing

Use `--dry-run` to see what would happen:
```bash
wsm rebase --dry-run
```

## Commit issues

### "commit message is required"

You ran `wsm commit` without `-m` and without `--interactive`.

**Fix**: Either provide a message or use interactive mode:
```bash
wsm commit -m "your message"
# or
wsm commit --interactive
```

### "No changes found in workspace"

All repositories are clean. Run `wsm status` to confirm, or `wsm diff` to check
for changes.

### Want to commit and push in one step

```bash
wsm commit -m "your message" --push
```

Add `--add-all` to also stage all unstaged changes:
```bash
wsm commit -m "your message" --add-all --push
```

## Delete issues

### Delete fails due to worktree safety checks

WSM refuses to remove worktrees that have uncommitted changes. This is
intentional -- it prevents data loss.

**Options**:
1. Commit or stash changes first, then delete.
2. If you are sure the changes can be discarded:
   ```bash
   wsm delete <name> --force-worktrees
   ```

### Want to keep the files but remove the workspace registration

```bash
wsm delete <name>
```

Without `--remove-files`, WSM removes the workspace config and worktree
registrations but leaves the directory intact.

## JS runner issues

### "Cannot find module 'wsm'"

The script was not executed through `wsm runner`. The `wsm` module is only
available inside the runner's goja environment.

**Fix**: Use `wsm runner`, not `node`:
```bash
wsm runner my-script.js
```

### Script returns nothing

The script needs to end with an expression (not a statement). Return an object
literal:

```javascript
// Wrong -- ends with a function call (statement)
console.log(result);

// Right -- ends with an expression
({
  count: repos.length,
  version: wsm.version,
});
```

### Script output is messy in automation

Use `--output-mode data` and `--print-result=false` for clean machine-readable
output:
```bash
wsm runner script.js --output-mode data --print-result=false
```

### `TypeError` from JS API call

The JS wrappers validate required inputs before running workflows. Common
examples:

- `createWorkspace` requires `name` and `repos`
- `workspaces.add` requires `workspaceName` and `repoName`
- `workspaces.merge` requires `workspaceName`
- `git.commit` requires `message` or `template`
- `git.branch.create/switch` require `branchName`

**Fix**: pass all required fields in the method input object.

### Script fails on one repository in batch operations

Some APIs return per-repository row results instead of throwing for every row
failure. Check row payloads for repository-level `success`/`error` fields:

- `git.branch.create/switch` return `results[]`
- `git.rebase.status/continue/abort` return `rows[]`

If the call itself fails (for example workspace not found), a top-level error
is thrown.

## Diagnostic commands reference

Here are the most useful commands for debugging, all using `--output-mode data`
for structured output:

```bash
# What repos does WSM know about?
wsm list repos --output-mode data

# What workspaces exist?
wsm list workspaces --output-mode data

# Detailed status for a workspace
wsm status --workspace <name> --output-mode data

# Rebase state for all repos
wsm rebase status --output-mode data

# Workspace metadata
wsm info <name> --output-mode data

# Diff across workspace
wsm diff --output-mode data

# Run the smoke test
wsm runner demo/js/wsm-api-smoke.js --output-mode both
```

## Getting help

- Run `wsm --help` for the top-level command list.
- Run `wsm <command> --help` for flags and usage of any specific command.
- Check `wsm help wsm-persistence-and-state` if you suspect state corruption.
- Check `wsm help wsm-architecture-overview` if contributing or debugging
  internals.

## See Also

- `wsm help wsm-getting-started`
- `wsm help wsm-command-reference`
- `wsm help wsm-persistence-and-state`
