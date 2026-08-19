# Tasks

## TODO

- [ ] Phase 1: Add ResolveBaseRef helper + unit tests (remote/local/none) <!-- t:39ov -->
- [ ] Phase 2: Rewrite CheckBranchMerged/CheckBranchNeedsRebase to return (value, resolved, err) and use ResolveBaseRef <!-- t:omoc -->
- [ ] Phase 3: Thread resolved/base_ref into RepositoryStatus (keep bool JSON compat) <!-- t:z6l5 -->
- [ ] Phase 4: Surface BASE_REF/unknown column in status table <!-- t:p5vo -->
- [ ] Phase 5: Regression test TestStatus_ForkedWorkspace_LocalOnlyBase <!-- t:ngv6 -->
- [ ] Phase 6: Validate against real workspace + no exit 128 in logs <!-- t:554x -->
