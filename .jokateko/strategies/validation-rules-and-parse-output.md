+++
title = 'Validation rules and parse output'
tier = 3
tags = ['backend', 'testing']
summary = 'What `jokateko parse` (alias `lint`) checks, in which order, every rule ID with its severity, and the exit-code and diagnostic format that humans, CI and agents rely on.'
+++

# Validation rules and parse output

`jokateko parse [-dir <path>]` (alias `lint`) is a read-only dry run over the whole workspace. It starts no server and changes no files. CI runs it on this repository, and agents should run it after bulk changes.

## Exit codes

| Code | Meaning |
|---|---|
| `0` | No **errors**. Warnings may still be printed |
| `1` | At least one error, or `-dir` missing / not a directory |

## Order of checks (`internal/validator`)

1. **Config.** Uses `<dir>/config.toml` if present, else `<dir>/.jokateko/config.toml`, else the compiled-in defaults. Load errors → `CFG-001`. Then `config.Validate` (the same function `serve` uses at startup).
2. Pre-scan all task and milestone IDs, for reference checks.
3. Tasks, building the dependency graph.
4. Cycle detection over the whole graph (`DAG-001`).
5. Milestones. 6. Strategies. 7. Glossary terms, including the cross-file title uniqueness check.

All diagnostics are collected before anything is printed. One run reports everything.

## Rules

### Configuration (`config.toml`)

| ID | Severity | Check |
|---|---|---|
| `CFG-000` | error | `version` is a supported schema version (`"0"`) |
| `CFG-001` | error | File loads: valid TOML, known structure |
| `CFG-002` | error | At least two `[[board.columns]]` |
| `CFG-003` | error | Column IDs non-empty and unique |
| `CFG-004` | error | `server.port` in 1024–65535 |
| `CFG-005` | error | `[paths]` stay inside the workspace |
| `CFG-006` / `CFG-007` | error | At least one `[[priorities]]`; priority IDs non-empty and unique |
| `CFG-008` / `CFG-009` | error | At least one `[[strategies.tiers]]`; tier IDs non-empty and unique |
| `CFG-010` | error | Every `[translations]` key is a known UI key (see *UI translations and project locale*) |
| `CFG-011` | error | `project.locale` looks like a BCP 47 tag |
| `TAG-001` | error | With `enforce_allowed`, `tags.allowed` is non-empty and every tag is kebab-case `^[a-z0-9]+(-[a-z0-9]+)*$` |
| `TAG-002` | error | No duplicates in `tags.allowed` |

### Entities

| ID | Severity | Check |
|---|---|---|
| `TSK-001` / `MLS-001` | warning | File name is `YYMMDD-slug.md` |
| `TSK-002` / `MLS-002` / `STR-001` / `GLS-001` | error | File starts with `+++` TOML frontmatter that closes and decodes |
| `TSK-003` / `MLS-003` / `STR-002` / `GLS-002` | error | Required fields: `title` + `summary` (strategies also need `tier`) |
| `TSK-004` | error | `status` is a configured column ID (the message lists the valid ones) |
| `TSK-005` | warning | `priority`, if set, is a configured `[[priorities]]` ID |
| `TSK-006` | error | `milestone`, if set, exists |
| `TSK-007` | error | Every `dependencies` entry exists |
| `TSK-008` | error | No self-dependency |
| `MLS-004` | warning | `target_date` is `YYYY-MM-DD` |
| `STR-003` | error | `tier` is a configured tier |
| `GLS-003` | error | Glossary titles are unique (case-insensitive) |
| `TSK-009` / `MLS-005` / `STR-004` / `GLS-004` | error | With `enforce_allowed`, every tag is in `tags.allowed` |
| `DAG-001` | error | Task dependencies form a DAG. Three-colour DFS; the message prints the full cycle path |

## Output format

```text
$ jokateko parse
[FAIL] Found 2 errors and 1 warning during validation:

ERROR [TSK-004] .jokateko/tasks/260901-auth-endpoints.md:4
  Invalid status: "review"
  Allowed column statuses: [backlog, ready, in_progress, in_review, done]
  Fix: Update 'status = "review"' to one of the defined board columns.

ERROR [DAG-001] .jokateko/tasks/260902-database-schema.md
  Circular task dependency detected:
  260902-database-schema -> 260903-migration-runner -> 260902-database-schema
  ...
```

On success:

```text
[OK] Validation successful:
  ✓ Configuration: .jokateko/config.toml (5 columns defined)
  ✓ Tasks: 54 files parsed (0 errors, 0 cycle conflicts)
  ✓ Milestones: 2 files parsed
  ✓ Strategies: 5 guidelines validated
  ✓ Glossary: 2 terms validated (0 errors)

Project is healthy. Exiting with code 0.
```

## Rules for contributors

1. Every new check gets a **new, never reused** rule ID in the right family, with a `Fix` hint when the fix is mechanical.
2. The goal is that write paths and `parse` agree: nothing written through the service should fail `parse`. Today they don't fully agree. MCP `create_task`/`update_task_content` don't verify `milestone` or `dependencies`; `force` deletes leave dangling references; and the service checks `priority` against the four built-in IDs, not `[[priorities]]`. Until those are fixed, `parse` is the safety net. Treat each mismatch as a bug, not as intended behaviour.
3. Add a validator test for each rule ID, including its severity.
