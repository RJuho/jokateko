+++
title = 'MCP tool catalog'
tier = 3
tags = ['api', 'backend']
summary = "Every Jokateko MCP tool, resource and prompt with the behaviour the JSON schemas don't tell you: guards, side effects, error conditions and what force really does."
+++

# MCP tool catalog

Parameter names, types and required fields come from the live schemas (`tools/list`), so they are not repeated here. This page lists the **behaviour** behind each tool, as implemented. All write tools go through `internal/service` (see *Go packages and data flow*) and fail when `[mcp] allow_mutations = false`.

## Tasks

| Tool | Behaviour |
|---|---|
| `list_tasks` | Compact rows (no body). Filters: `status`, `milestone`, `tag`, `priority`. Sorted like the board columns |
| `get_task` | Full task including body, criteria counts, timestamps |
| `create_task` | ID `YYMMDD-slug`. `status` defaults to `[board] default_create_state`, and an unknown status → error. An unknown `priority` silently becomes `medium`. Tags are checked against `[tags] allowed` only when `enforce_allowed = true`. Attaching to an **archived** milestone (closed or all tasks done) needs `reopen_milestone: true`. **Not checked:** that `milestone` exists, and that `dependencies` exist or are acyclic. Prefer `add_task_dependency`, and run `jokateko parse` |
| `update_task_content` | Partial update with the same tag and archived-milestone guards and the same gaps as create. Changing `body` outside `[board] editable_states` is rejected unless the only difference is checkbox state |
| `update_task_status` | Any configured column **except `done`**: agents must use `complete_task`. (The Web UI and REST may move a card to Done directly; this guard is MCP-only.) |
| `list_task_items` / `update_task_item` | 1-based index over every checklist item in the body, including items inside notes |
| `add_task_note` | Appends `### [YYYY-MM-DD HH:MM UTC]` under `## Notes` (kept before `## Completion Summary`). Allowed in every status. Checklist items in the note become new criteria that block completion |
| `set_task_target` | Sets or clears `target_at` (`YYYY-MM-DD`, `YYYY-MM-DDTHH:MM`, RFC 3339; empty string clears) |
| `add_task_dependency` / `remove_task_dependency` | Both tasks must exist. Self-dependencies and cycles (DFS over the whole graph) are rejected. Adding is idempotent. **This is the safe way to set dependencies** |
| `complete_task` | Rejects when any criterion is unchecked, or when a dependency is not `done` (unless `ignore_dependencies`). Sets `status = "done"`, replaces `summary`, and writes `## Completion Summary` with *Completed At*, *What Was Done* and *Why / Rationale*. The result lists downstream tasks that are now unblocked |
| `delete_task` | Rejected when other tasks depend on it. `force: true` deletes the file **without** editing the dependents, leaving dangling IDs that `jokateko parse` reports as `TSK-007`. Remove those dependencies first |

## Milestones

| Tool | Behaviour |
|---|---|
| `list_milestones` | Progress per milestone. Archived ones (status `closed`, or every task `done`) are hidden unless `include_archived: true`. A milestone with zero tasks is never auto-archived |
| `get_milestone` | Full milestone plus assigned task IDs |
| `create_milestone` / `update_milestone` | ID `YYMMDD-slug` (creation date). `status` is `open` or `closed`. `target_date` is `YYYY-MM-DD`. Tags are checked as for tasks |
| `delete_milestone` | Rejected when tasks are assigned. `force: true` deletes the file **without** clearing the tasks' `milestone` field (→ `TSK-006` in `parse`). Reassign the tasks first |

## Strategies and glossary

| Tool | Behaviour |
|---|---|
| `list_strategies` | Title, tier, tags and summary only (progressive disclosure). Filter by `tier` and `tag` |
| `get_strategy` | Full Markdown body |
| `create_strategy` / `update_strategy` | ID is the slug of the title. `tier` must be a configured tier (default 1 if omitted) |
| `delete_strategy` | No reference checks |
| `lookup_glossary` | Without `term`: whole glossary. With `term`: exact ID or case-insensitive title substring, falling back to FTS search |
| `create_glossary_term` / `update_glossary_term` | ID is the slug of the title. Duplicate titles are reported by `parse` (`GLS-003`) |
| `delete_glossary_term` | No reference checks |

## Discovery and search

| Tool | Behaviour |
|---|---|
| `get_board_state` | Project name and columns with counts, plus `handled_by` and `instructions` per column |
| `list_tags` | Every tag in use with per-type counts, and whether the vocabulary is enforced. Call it before inventing a new tag |
| `search_tasks` · `search_milestones` · `search_strategies` · `search_glossary` · `search_all` | SQLite FTS5 over title, summary and body (so notes and completion summaries too). Optional `tag` filter, `limit` default 20. Results carry a highlighted snippet |

## Resources

| URI | Content |
|---|---|
| `jokateko://board` | Board snapshot (columns + tasks) |
| `jokateko://strategies/tier1` | All tier-1 strategies, concatenated |
| `jokateko://strategies/tiers` | The configured tiers with titles and scopes |
| `jokateko://glossary` | The whole glossary |

## Prompt

`next_task`: picks the highest-priority task in `ready` whose dependencies are all `done`, and attaches the tier-1 strategy summaries.

## Errors

Service errors carry a kind (invalid input, not found, conflict). Tools return them as MCP tool errors with an actionable message, for example the list of blocking tasks, unchecked criteria, or valid columns. Agents should fix the input instead of retrying the same call.
