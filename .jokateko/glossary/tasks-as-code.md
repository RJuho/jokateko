+++
title = 'Tasks-as-Code'
tags = ['docs']
summary = 'Keeping tasks, milestones, specifications and project knowledge as version-controlled Markdown files in the code repository, so they are the single source of truth.'
+++

Tasks-as-Code treats work items and specifications like code. They live in `.jokateko/` next to the source, change through commits and pull requests, and are linted in CI (`jokateko parse`).

**Benefits**
- **Auditable:** every change to a task is a Git commit, so `git log`/`git blame` work on the plan too.
- **Branch-aware:** a feature branch carries its own task updates, and switching branches switches the board (the watcher re-indexes).
- **Offline and tool-free:** plain Markdown with TOML frontmatter, readable without Jokateko.
- **Merge-friendly:** one file per entity, so parallel work rarely conflicts.
- **Agent-friendly:** AI agents read and update the same files through MCP.

See the tier-1 strategy *Tasks-as-Code Architecture* and the term *Spec-First*.
