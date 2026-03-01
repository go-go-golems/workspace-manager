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

This page covers frequent failure modes and fast recovery actions. Use it when commands fail unexpectedly or output looks inconsistent.

This matters because most operational failures are recoverable if you identify whether the issue is registry, workspace, git state, or command context.

## Quick triage checklist

1. Confirm your working directory and workspace selection.
2. Verify discovered repositories are current.
3. Check workspace JSON presence.
4. Re-run with explicit `--workspace` and `--output-mode data`.

## Common issues

| Problem | Cause | Solution |
|---|---|---|
| `not in a workspace directory` | Running outside workspace and no explicit workspace selected | Pass `--workspace <name>` |
| `repositories not found` on create | Registry does not contain repo names | Re-run `wsm discover` and verify with `wsm list repos` |
| Rebase fails for one repo only | Branch missing/conflicts in that repository | Use `wsm rebase status --repo <repo>` then continue/abort as needed |
| Delete fails due to worktree safety checks | Uncommitted changes in one or more worktrees | Resolve changes or use `--force-worktrees` intentionally |
| `wsm runner` script cannot access API | Script not executed via runner or syntax/runtime error | Run with `wsm runner <script>` and inspect error output |

## Recommended debug commands

```bash
wsm list repos --output-mode data
wsm list workspaces --output-mode data
wsm status --workspace <name> --output-mode data
wsm rebase status --repo <repo> --output-mode data
wsm runner demo/js/wsm-api-smoke.js --output-mode both
```

## Escalation hints for contributors

- If output parsing fails, inspect command runtime settings (`output-mode`).
- If state looks wrong, inspect persisted config files under `~/.config/workspace-manager/`.
- If JS API behavior diverges, compare `pkg/wsmjs/service` calls with equivalent workflow execution.

## See Also

- `wsm help wsm-getting-started`
- `wsm help wsm-command-reference`
- `wsm help wsm-persistence-and-state`
