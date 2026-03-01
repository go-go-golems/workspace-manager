---
Title: WSM Architecture Overview
Slug: wsm-architecture-overview
Short: High-level architecture map for commands, workflows, services, and JS integration.
Topics:
- workspace-manager
- architecture
- refactor
Commands:
- all
Flags:
- --output-mode
IsTopLevel: true
IsTemplate: false
ShowPerDefault: true
SectionType: GeneralTopic
---

This page covers how the WSM codebase is organized and where behavior should live. It is the starting point for contributors.

This matters because the project intentionally keeps command adapters thin and puts reusable behavior in `pkg/`.

## Layering model

- `cmd/wsm/...`: CLI command wiring, flag decoding, output formatting.
- `pkg/wsm/workflows/...`: operation-specific orchestration and request/result contracts.
- `pkg/wsm/...`: core domain services and git/workspace primitives.
- `pkg/wsmjs/...`: JavaScript facade (`service`), module adapter (`module`), and script runner (`runner`).

## Branch abstraction

Branch behavior is centralized in `pkg/wsm/branch` with typed enums and resolution plans:

- `ResolutionMode`
- `ResolutionStrategy`
- `RemoteRefKind`

Avoid re-implementing branch policy in command handlers.

## CLI output model

Commands use shared runtime settings and support:

- human output
- structured data output
- combined output

This keeps automation and interactive usage on the same command surface.

## JS integration model

- `wsm runner` bootstraps goja runtime and registers native module `wsm`.
- `pkg/wsmjs/module` exports JS-facing API.
- `pkg/wsmjs/service` maps JS calls to existing workflow/services.

## Troubleshooting

| Problem | Cause | Solution |
|---|---|---|
| New command has lots of business logic | Logic added in `cmd/` instead of `pkg/` | Move orchestration into workflow/service package |
| Duplicate branch decisions in multiple files | Branch policy bypassed | Route through `pkg/wsm/branch` service |
| JS API diverges from CLI behavior | Separate code paths | Reuse workflow/service layer in both adapters |

## See Also

- `wsm help wsm-command-reference`
- `wsm help wsm-persistence-and-state`
- `wsm help wsm-js-api-and-runner`
