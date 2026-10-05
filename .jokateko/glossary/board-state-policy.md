+++
title = 'Board state policy'
tags = ['backend', 'frontend']
summary = "The workflow rules configured under `[board]`: which columns allow editing a task's spec (`editable_states`), where tasks can be created (`creatable_states`, `default_create_state`), and who handles each column (`handled_by`, `instructions`)."
+++

| Key | Default | Effect |
|---|---|---|
| `editable_states` | `["backlog"]` | Outside these columns, a task body is locked. Only checkbox toggles, notes and completion are allowed (REST and MCP both return a conflict) |
| `creatable_states` | `["backlog"]` | Columns that show the "+" create button in the UI |
| `default_create_state` | `"backlog"` | Status for new tasks when none is given |
| `[[board.columns]] handled_by` | none | Who owns the column, for example `human`, `agent:coder`, `agent:reviewer` |
| `[[board.columns]] instructions` | none | Entry and exit criteria for the column |
| `[[board.columns]] sort_by` / `sort_direction` | workflow default | Per-column card order |

`handled_by` and `instructions` are shown in the UI's column dialog, returned by `get_board_state`, and appended to the MCP server instructions (see *Agent guidance*).

The status `done` is special: only `complete_task` moves a task there over MCP. In the UI, a human may drag a card to Done directly.
