# Tasks

## TODO

- [x] Phase 1: Add ResolveBaseRef helper + unit tests (remote/local/none) <!-- t:39ov -->
- [x] Phase 2: Rewrite CheckBranchMerged/CheckBranchNeedsRebase to return (value, resolved, err) and use ResolveBaseRef <!-- t:omoc -->
- [x] Phase 3: Thread resolved/base_ref into RepositoryStatus (keep bool JSON compat) <!-- t:z6l5 -->
- [ ] Phase 4: Surface BASE_REF/unknown column in status table <!-- t:p5vo -->
- [ ] Phase 5: Regression test TestStatus_ForkedWorkspace_LocalOnlyBase <!-- t:ngv6 -->
- [ ] Phase 6: Validate against real workspace + no exit 128 in logs <!-- t:554x -->
- [x] E1: BaseComparison model + honest checks (Q1/Q2 core) <!-- t:6sqa -->
- [x] E2: DefaultBranch discovery via symbolic-ref + persistence (Q3) <!-- t:y398 -->
- [x] E3: Per-repo override storage + LoadWorkspace merge + precedence (Q4 core) <!-- t:wmpg -->
- [x] E4: wsm set-base command (Q4 surface) <!-- t:adt3 -->
- [ ] E5: Status table BASE column + honest MERGED/REBASE + JSON fields (Q1/Q2 surface) <!-- t:8h31 -->
- [ ] E6: Validate end-to-end on ragkit-coinvault-mysql + set-base flow <!-- t:ayaj -->
- [ ] F1: ForkWorkflow typed ErrBranchDivergence + BaseBranch override <!-- t:2esw -->
- [ ] F2: fork CLI --base-branch flag + huh prompt + allowPrompt gating <!-- t:dnix -->
- [ ] F3: Validate fork divergence flow end-to-end <!-- t:yqzf -->
