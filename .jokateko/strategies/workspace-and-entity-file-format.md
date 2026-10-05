+++
title = 'Workspace and entity file format'
tier = 2
tags = ['backend', 'docs']
summary = "Layout of a project's .jokateko/ workspace, ID and file-naming rules, and the TOML-frontmatter Markdown format of tasks, milestones, strategies and glossary terms, including the body sections Jokateko itself manages."
+++

# Workspace and entity file format

## Workspace layout

`jokateko init` creates this layout in the project root, with a commented `config.toml` and one starter file per entity type:

```text
my-project/
└── .jokateko/
    ├── config.toml        # overlays the compiled-in defaults (internal/config/default.toml)
    ├── tasks/             # YYMMDD-slug.md
    ├── milestones/        # YYMMDD-slug.md
    ├── strategies/        # slug.md
    └── glossary/          # slug.md
```

- Commit everything under `.jokateko/`. Add the export target (`dist-kanban/` by default, `[paths] export`) to `.gitignore`.
- Entity dirs can be moved with `[paths]`. Configured paths must stay inside the workspace (`CFG-005`).
- There are no lockfiles or runtime state on disk. A running daemon is found through `GET /api/health`.

## IDs and file names

The file name without `.md` is the entity ID.

| Entity | Pattern | Example |
|---|---|---|
| Task | `YYMMDD-slug` | `261005-fix-login.md` |
| Milestone | `YYMMDD-slug` | `261005-public-release-v010.md` |
| Strategy | `slug` | `architecture.md` |
| Glossary term | `slug` | `tasks-as-code.md` |

- `YYMMDD` is the **creation date** (UTC, from the service clock). It is not a target date. It keeps files in chronological order.
- The slug is generated from the title: lowercase `a-z0-9`, with spaces, `-` and `_` collapsed to one `-`, at most 40 characters.
- Valid IDs match `^[a-z0-9][a-z0-9-]{0,99}$`, so an ID is always a safe single file name. An explicit duplicate ID is rejected. A duplicate generated ID gets a `-2`, `-3`, … suffix.
- Non-dated task and milestone names only produce a `TSK-001` / `MLS-001` warning.

## Frontmatter

Every file starts with **TOML** frontmatter between `+++` lines (YAML `---` is not supported). A UTF-8 BOM and CRLF line endings are accepted.

### Task

```toml
+++
title = 'Fix login redirect'
status = 'in_progress'          # a [[board.columns]] id
priority = 'high'               # a [[priorities]] id; optional
milestone = '261005-public-release-v010'   # optional
tags = ['backend']
summary = 'One or two sentences.'
dependencies = ['261004-session-store']    # optional, blocking task IDs
created_at = '2026-10-05T07:23:39Z'        # managed
changed_at = '2026-10-05T10:31:31Z'        # managed
target_at = '2026-10-10'                   # optional due date
+++
```

- `created_at` is set once. Older files without it get a fallback derived from the ID date or the file's mod time. `changed_at` is restamped on every write through the service.
- `target_at` accepts `YYYY-MM-DD`, `YYYY-MM-DDTHH:MM` or RFC 3339 UTC.
- The status ID `done` is special: milestone progress, auto-archive and `complete_task` treat exactly `done` as completed.

### Milestone

```toml
+++
title = 'Public Release v0.1.0'
status = 'open'                 # open | closed
target_date = '2026-11-01'      # optional, YYYY-MM-DD
tags = ['release']
summary = '...'
+++
```

Progress is computed, not stored. A milestone with tasks is **auto-archived** when all of its tasks are `done`, or when it is `closed`. Its timeframe comes from the earliest and latest `target_at` of its tasks, falling back to `target_date`.

### Strategy

```toml
+++
title = 'Go packages and data flow'
tier = 2                        # 1 core · 2 domain · 3 implementation ([[strategies.tiers]])
tags = ['backend']
summary = '...'
+++
```

### Glossary term

```toml
+++
title = 'Tasks-as-Code'
tags = ['docs']
summary = 'Short definition.'
+++
```

Glossary titles must be unique across the project (`GLS-003`).

## Body conventions (tasks)

- Free Markdown. GitHub-style checklists (`- [ ]`, `- [x]`) anywhere in the body are the task's **acceptance criteria**. They are counted for progress and must all be checked before `complete_task` succeeds.
- `## Notes`: maintained by `add_task_note`. Each note is a `### [YYYY-MM-DD HH:MM UTC]` block. Checklist items inside notes also count as criteria.
- `## Completion Summary`: written by `complete_task` with a timestamp, `### What Was Done` and `### Why / Rationale`. It always stays the last section. Notes added later are inserted before it.
- Outside `[board] editable_states` (default `["backlog"]`), the body is locked. Only checkbox toggles, notes and completion are allowed.
- Mermaid code blocks (` ```mermaid `) render as diagrams in the Web UI and in static exports.

Hand-editing these files is possible, since the watcher ingests them. But agents must use the MCP tools (see *Agent guidance*), and humans should prefer the UI so the guards above apply.
