# Changelog

## 2026-02-28

- Initialized ticket workspace `WSM-MO-007-JS-API-DESIGN` with design and diary documents.
- Completed evidence-gathering sweep across `workspace-manager`, `go-go-goja`, and `geppetto` with line-anchored references.
- Authored comprehensive JS API brainstorm covering multiple API styles, recommendation, API sketches, pseudocode, implementation phases, test strategy, and risks.
- Recorded detailed investigation diary with prompt context, command history, failures, and design rationale.
- Related key architecture files to design and diary docs via `docmgr doc relate`.
- Added missing topic vocabulary slugs (`api-design`, `geppetto`, `goja`, `javascript`) and re-ran doctor to clean status.
- Uploaded bundled design + diary PDF to reMarkable at `/ai/2026/02/28/WSM-MO-007-JS-API-DESIGN`.
- Cleaned frontmatter RelatedFiles metadata to remove duplicate/invalid path expansions and kept a concise evidence set.
- Uploaded refreshed bundle `WSM-MO-007 JS API Brainstorm v2` after metadata cleanup and verified both files in remote listing.
- Uploaded final refreshed bundle `WSM-MO-007 JS API Brainstorm v3` as the final handoff bundle for review.

## 2026-03-01

- Converted WSM-MO-007 from research-only to implementation execution with phased task list.
- Implemented `pkg/wsmjs/service` workflow facade for discover/create/status/list operations.
- Implemented native `require("wsm")` module with `createManager`, functional shortcuts, grouped manager namespaces, and typed const exports.
- Added reusable JS runtime runner package (`pkg/wsmjs/runner`) and tests for module + runner surfaces.
- Added `wsm runner` CLI verb (`cmd/wsm/cmds/js`) and registered it in root command.
- Added runnable demo script at `demo/js/wsm-api-smoke.js`.
- Verified with `go test ./pkg/wsmjs/... ./cmd/wsm/...`, `go test ./...`, and `go run ./cmd/wsm runner demo/js/wsm-api-smoke.js`.
- Recorded implementation commit `88b8f52` for JS API core + runner verb delivery.
- Uploaded refreshed reMarkable bundle `WSM-MO-007 JS API Brainstorm v4` and verified remote listing.
