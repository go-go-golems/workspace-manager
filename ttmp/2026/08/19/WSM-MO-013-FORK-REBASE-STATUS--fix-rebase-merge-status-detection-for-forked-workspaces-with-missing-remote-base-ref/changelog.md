# Changelog

## 2026-08-19

- Initial workspace created


## 2026-08-19

Created ticket and wrote intern-oriented design doc: root-caused forked-workspace status bug to pkg/wsm/git_utils.go hardcoding origin/<base> and swallowing/returning errors on missing remote-tracking ref; proposed ResolveBaseRef (prefer remote, fall back to local, else unknown) with 6-phase plan

### Related Files

- /home/manuel/workspaces/2026-08-19/fix-git-rebase-bug/workspace-manager/pkg/wsm/git_utils.go — Buggy merge/rebase checks


## 2026-08-19

Added investigation diary (Steps 1-3): log triage, reproduction against ragkit-coinvault-mysql/geppetto (exit 128, remote-tracking ref absent, local base present), system mapping and fix design

### Related Files

- /home/manuel/workspaces/2026-08-19/fix-git-rebase-bug/workspace-manager/pkg/wsm/status.go — Call site swallowing errors


## 2026-08-19

Wrote design-doc 02: Status Reporting Enhancements covering Q1 (failure reason), Q2 (which branch compared), Q3 (per-repo default base via symbolic-ref, verified goldeneaglecoin.com=develop), Q4 (per-worktree override + wsm set-base). Unified by BaseComparison struct; 5 decision records (E1-E5); 6-phase plan (E1-E6)

### Related Files

- /home/manuel/workspaces/2026-08-19/fix-git-rebase-bug/workspace-manager/pkg/wsm/types.go — BaseComparison model


## 2026-08-19

Added 6 tasks (E1-E6) and related 8 key files to design-doc 02 (types, git_utils, branch/types, cli_client, discovery, workspace, status cmd, goldeneaglecoin.com)

### Related Files

- /home/manuel/workspaces/2026-08-19/fix-git-rebase-bug/workspace-manager/pkg/wsm/workspace.go — RepositoryMetadata + LoadWorkspace merge (E3)


## 2026-08-19

Updated design-doc 02 per user feedback: wsm set-base now defaults to the in-workspace .wsm/wsm.json only; --global writes the config-dir workspace JSON only (no mirroring). Revised decision E3 (two flag-selected stores, in-workspace beats config-dir at load), added config-dir Repository.BaseBranch/BaseRemote fields, expanded precedence to 6 layers (in-workspace per-repo > config-dir per-repo > workspace base > discovered default > env > main), updated §5.7 command, §7.3 flow, §7.4 diagram, Phase E4, and risks.

### Related Files

- /home/manuel/workspaces/2026-08-19/fix-git-rebase-bug/workspace-manager/cmd/wsm/cmds/workspace/set_base.go — set-base default=workspace, --global=config-dir


## 2026-08-19

Added design-doc 03 (fork divergence confirmation) + tasks F1-F3; deferring implementation until E1-E6 base-resolution foundation lands

### Related Files

- /home/manuel/workspaces/2026-08-19/fix-git-rebase-bug/workspace-manager/pkg/wsm/workflows/fork_workflow.go — deferred fork divergence fix

