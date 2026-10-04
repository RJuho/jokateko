+++
title = 'Configurable UI translations for modal strings'
status = 'backlog'
priority = 'low'
tags = ['frontend', 'backend', 'i18n']
summary = 'Move the hard-coded English strings in the Web UI modals (task detail, edit, create, column detail, About) to t() keys, and make every UI translation key configurable from config.toml.'
created_at = '2026-10-04T17:50:44Z'
changed_at = '2026-10-04T17:50:44Z'
+++

## Context

This was split out of `261004-web-ui-review-fixes-after-preact-11-upgr`. `model.TranslationsConfig` is a fixed Go struct, merged field by field in `internal/config/load.go`. Any new `t()` key added in the UI therefore can't be configured from `config.toml` unless the backend changes too.

## Acceptance Criteria
- [ ] Decide whether to keep the fixed struct or switch to `map[string]string`, and record the decision as a strategy if it is architectural.
- [ ] Translations config supports the new keys, and `default.toml`, the merge logic and gentypes (`web/src/types/generated.ts`) are updated.
- [ ] Hard-coded English strings in TaskDetailModal, TaskEditModal, CreateTaskModal, ColumnDetailModal and AboutModal use `t()` keys, with defaults in `defaultTranslations`.
- [ ] i18n unit tests cover the new keys and config overrides.
- [ ] `bun run check`, `bun test`, `bun run test:e2e` and `go test ./...` pass.
