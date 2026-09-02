+++
title = 'Goldmark Markdown Rendering & Prose Typography Support'
status = 'done'
priority = 'high'
milestone = '260902-mvp'
tags = ['frontend', 'backend', 'feature']
summary = 'Enabled server-side Goldmark markdown rendering into HTML and integrated @tailwindcss/typography (prose) across Task Modal, Strategies, and Glossary views.'
dependencies = ['260902-dynamic-mcp-proxy-hot-failover-auto-reco']
+++

# Goldmark Markdown Rendering & Prose Typography Support

Enable full markdown-to-HTML rendering using Go's `goldmark` package on the backend and integrate `@tailwindcss/typography` (`prose` classes) in the Web UI across the Task Detail Modal, Architectural Strategies view, and Project Glossary view.

## Acceptance Criteria
- [x] Extend domain models (`Task`, `Strategy`, `GlossaryTerm`, `Milestone`) with `BodyHTML` field
- [x] Integrate `parser.RenderHTML` using `goldmark` with GFM, TaskList extensions, and interactive checkbox formatting
- [x] Populate `BodyHTML` during parsing, store retrieval, and mutation handlers
- [x] Regenerate TypeScript models and update Valibot schemas to include `body_html`
- [x] Integrate `@tailwindcss/typography` (`prose`) styling in `TaskDetailModal.tsx`, `StrategiesView.tsx`, and `GlossaryView.tsx`
- [x] Ensure task checklist checkboxes remain interactive via event delegation while disabling in static mode
- [x] Add comprehensive E2E Playwright tests verifying bold text formatting and prose class styling
- [x] Verify all unit, integration, and E2E tests pass with `go test ./...` and `bunx playwright test`

## Completion Summary
- **Completed At:** 2026-09-02T17:49:30Z

### What Was Done
Added `BodyHTML` to Go domain models and implemented markdown-to-HTML compilation via `goldmark`. Updated SQLite store and REST handlers to compute and deliver rendered HTML. Regenerated TypeScript types and updated Valibot schemas. Replaced manual line-splitting in `TaskDetailModal.tsx`, `StrategiesView.tsx`, and `GlossaryView.tsx` with `@tailwindcss/typography` (`prose`) containers. Styled checkboxes with daisyUI classes and added full Playwright E2E test coverage.

### Why / Rationale
Fixes unrendered markdown syntax (such as bold headers like `- **Completed At:**`, code fences, lists, and quotes) so tasks, architectural strategies, and glossary terms render as designed for human developers while preserving raw markdown in files as the single source of truth.
