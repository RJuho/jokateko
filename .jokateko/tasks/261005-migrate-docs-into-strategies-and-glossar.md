+++
title = 'Migrate docs/ into Strategies and Glossary'
status = 'backlog'
priority = 'high'
milestone = '261005-public-release-v010'
tags = ['docs', 'release']
summary = 'Move the ~2,000 lines of docs/ specifications into Jokateko strategies and glossary terms, verifying each claim against the current code, then delete docs/.'
created_at = '2026-10-05T10:58:09Z'
changed_at = '2026-10-05T10:58:09Z'
+++

## Context
Jokateko should be the single home for project knowledge. `docs/` holds 7 files (~2,000 lines). Some content is stale, and `docs/README.md` links use `file:///workspaces/...`, which break outside the devcontainer. The knowledge base itself is thin: 5 strategies (one, `zero-downtime-daemon-migration`, is a one-line stub) and 2 one-line glossary terms.

While moving content, **check each claim against the current code** and drop or fix anything stale. Do not copy what the MCP tool schemas or `internal/config/default.toml` already state; link or summarize instead. Keep strategies scannable. Progressive disclosure means the tier-1 docs stay short.

## Mapping
| Source | Target |
|---|---|
| `docs/README.md` (principles + architecture diagram) | merge into tier-1 `architecture` |
| `docs/module-structure.md` | new tier-2 "Go packages and data flow" (package boundaries, ingest / UI mutation / MCP pipelines, concurrency model) |
| `docs/file-structure.md` (source layout) | merge into tier-2 `project-structure` |
| `docs/file-structure.md` (workspace layout, naming, entity formats, gitignore advice) | new tier-2 "Workspace and entity file format" |
| `docs/web-ui-architecture.md` | new tier-2 "Web UI architecture" (snapshot injection, live vs static, SSE, Valibot, bundling, Mermaid, Playwright/Lighthouse) |
| `docs/mcp-specification.md` (modes, proxy, failover, handshake replay) | new tier-2 "MCP server and stdio proxy", which absorbs the `zero-downtime-daemon-migration` stub |
| `docs/mcp-specification.md` (tool catalog, resources, prompts) | new tier-3 "MCP tool catalog" (only behaviour the schemas do not already describe) |
| `docs/api-specification.md` | new tier-3 "REST API and SSE" |
| `docs/validation-and-linting.md` | new tier-3 "Validation rules and parse output" (rule IDs, exit codes) |

## Acceptance Criteria
- [ ] `architecture` (tier 1) updated with the principles and the high-level Mermaid diagram from `docs/README.md`
- [ ] New tier-2 strategies created: Go packages and data flow, Workspace and entity file format, Web UI architecture, MCP server and stdio proxy
- [ ] `project-structure` updated with the source layout; its `docs/` references removed
- [ ] New tier-3 strategies created: MCP tool catalog, REST API and SSE, Validation rules and parse output
- [ ] `zero-downtime-daemon-migration` stub merged into the MCP proxy strategy, then deleted
- [ ] Glossary: `tasks-as-code` and `zero-cgo` expanded; new terms added: Spec-First, Strategy tier (progressive disclosure), Live mode vs Static export, Snapshot injection, Unidirectional sync, Atomic write, MCP proxy failover, Task slug, Board state policy, Agent guidance
- [ ] Every migrated statement checked against the code; stale content fixed or dropped (list notable corrections in Notes)
- [ ] `docs/` deleted; references updated in `AGENTS.md` and `README.md`
- [ ] `jokateko parse` passes; strategies render correctly (Mermaid included) in the Web UI
- [ ] All changes to `.jokateko/` made through MCP tools only
