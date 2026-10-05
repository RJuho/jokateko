+++
title = 'Migrate docs/ into Strategies and Glossary'
status = 'done'
priority = 'high'
milestone = '261005-public-release-v010'
tags = ['docs', 'release']
summary = 'docs/ is migrated into Jokateko and deleted. There are 11 strategies (tiered 1–3) and 12 glossary terms, all checked against the current code, with known code gaps flagged in place.'
created_at = '2026-10-05T10:58:09Z'
changed_at = '2026-10-05T11:24:41Z'
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
- [x] `architecture` (tier 1) updated with the principles and the high-level Mermaid diagram from `docs/README.md`
- [x] New tier-2 strategies created: Go packages and data flow, Workspace and entity file format, Web UI architecture, MCP server and stdio proxy
- [x] `project-structure` updated with the source layout; its `docs/` references removed
- [x] New tier-3 strategies created: MCP tool catalog, REST API and SSE, Validation rules and parse output
- [x] `zero-downtime-daemon-migration` stub merged into the MCP proxy strategy, then deleted
- [x] Glossary: `tasks-as-code` and `zero-cgo` expanded; new terms added: Spec-First, Strategy tier (progressive disclosure), Live mode vs Static export, Snapshot injection, Unidirectional sync, Atomic write, MCP proxy failover, Task slug, Board state policy, Agent guidance
- [x] Every migrated statement checked against the code; stale content fixed or dropped (list notable corrections in Notes)
- [x] `docs/` deleted; references updated in `AGENTS.md` and `README.md`
- [x] `jokateko parse` passes; strategies render correctly (Mermaid included) in the Web UI
- [x] All changes to `.jokateko/` made through MCP tools only

## Notes

### [2026-10-05 11:14 UTC]

**Result.** `docs/` (7 files) is gone. Strategies 5 → 11: `architecture` and `project-structure` updated; new `go-packages-and-data-flow`, `workspace-and-entity-file-format`, `web-ui-architecture`, `mcp-server-and-stdio-proxy` (tier 2), and `mcp-tool-catalog`, `rest-api-and-sse`, `validation-rules-and-parse-output` (tier 3). The stub `zero-downtime-daemon-migration` was deleted. Glossary 2 → 12 terms. `jokateko parse` is OK. Every diagram renders in Chromium, with no Mermaid syntax errors and no console errors.

**Stale doc content corrected while migrating**
- Frontmatter is TOML between `+++`, not YAML `---` (rule names `TSK-002` etc. are `toml-frontmatter`).
- The milestone ID prefix `YYMMDD` is the creation date, not the target date. Go baseline is 1.27, not 1.23.
- `config.toml` is **not** watched; it is read at startup (the docs said the watcher tracks it).
- SSE also sends `strategy.*` and `glossary.*` events. `board.refreshed` is never emitted (the UI still listens for it).
- The UI has a calendar view and a third boot mode, `client` (localStorage). The doc's component tree, `MilestonesView`, Tailwind "standalone CLI", and `index.tsx` paths were outdated.
- CSP hashes are computed at runtime from the embedded `index.html`, not from `hashes.json`.
- The rule list was missing `CFG-006`…`CFG-011`. A root `config.toml` takes precedence over `.jokateko/config.toml`.
- `robots.txt` allows all crawlers.
- `force` deletes do **not** clean up dependents or task milestones (the docs claimed they did).
- MCP tool catalog: `delete_*`, `set_task_target` and the `jokateko://strategies/tiers` resource were added. CLI commands `about` and `licenses` were missing from the docs and COMMANDS.md (for the README task).

**Code gaps found (documented in the strategies; candidate follow-up tasks, not fixed here)**
1. MCP `create_task`/`update_task_content` don't check that `milestone` or `dependencies` exist, and don't run cycle detection. Only `add_task_dependency` does.
2. `force` deletes leave dangling references, which then fail `parse` (`TSK-006`/`TSK-007`).
3. The service validates `priority` against the 4 built-in IDs and silently maps unknown values to `medium`, ignoring `[[priorities]]`. `parse` uses the configured list.
4. REST doesn't enforce `[tags]` or the archived-milestone guard. Those checks exist only in the MCP layer.
5. `[mcp] enabled` and `[mcp] timeout_seconds` are parsed but unused.
6. `web/src/types/generated.ts` (gentypes output) is imported nowhere; the Valibot schemas are hand-maintained.
7. Dead SSE listener `board.refreshed` in `web/src/state/sse.ts`.
8. Security note for SECURITY.md: the Host (DNS-rebinding) check only applies to loopback connections, so with `host = "0.0.0.0"` (this repo's config) the API is reachable from the LAN with no auth.

## Completion Summary
- **Completed At:** 2026-10-05T11:24:41Z

### What Was Done
- Updated the tier-1 `architecture` strategy with the invariants and the high-level Mermaid diagram. Updated `project-structure` so it no longer refers to docs/ and says that documentation lives in `.jokateko/`.
- New tier-2 strategies: `go-packages-and-data-flow`, `workspace-and-entity-file-format`, `web-ui-architecture`, `mcp-server-and-stdio-proxy`. The last one absorbs the deleted `zero-downtime-daemon-migration` stub.
- New tier-3 strategies: `mcp-tool-catalog`, `rest-api-and-sse`, `validation-rules-and-parse-output`.
- Glossary: expanded `tasks-as-code` and `zero-cgo`. Added `spec-first`, `strategy-tier`, `live-mode-and-static-export`, `snapshot-injection`, `unidirectional-sync`, `atomic-write`, `mcp-proxy-failover`, `task-slug`, `board-state-policy`, `agent-guidance`.
- Deleted `docs/` (7 files). AGENTS.md and README.md now point to the strategies.
- Verification: `jokateko parse` OK. All strategies render in Chromium with every Mermaid diagram drawn and no console errors.

### Why / Rationale
The docs had drifted from the code: YAML frontmatter, a watched config.toml, cleanup on force delete, outdated SSE events and UI structure. So every statement was verified against the source instead of copied. Where the code does something surprising, the strategy describes the actual behaviour and marks it as a gap, so readers are not misled. Content is split by tier for progressive disclosure: tier 1 stays short, and tool and API detail sits in tier 3. The MCP schemas already carry parameter definitions, so the catalog documents only behaviour. The code gaps are tracked as a separate follow-up task.
