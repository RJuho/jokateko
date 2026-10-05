+++
title = 'Configurable UI translations for modal strings'
status = 'done'
priority = 'low'
tags = ['backend', 'frontend', 'i18n']
summary = 'UI translations are a key/value map: every Web UI label, including all modal strings, can be configured from config.toml [translations], and unknown keys are rejected with CFG-010.'
created_at = '2026-10-04T17:50:44Z'
changed_at = '2026-10-05T07:30:18Z'
+++

## Context

This was split out of `261004-web-ui-review-fixes-after-preact-11-upgr`. `model.TranslationsConfig` is a fixed Go struct, merged field by field in `internal/config/load.go`. Any new `t()` key added in the UI therefore can't be configured from `config.toml` unless the backend changes too.

## Acceptance Criteria
- [x] Decide whether to keep the fixed struct or switch to `map[string]string`, and record the decision as a strategy if it is architectural.
- [x] Translations config supports the new keys, and `default.toml`, the merge logic and gentypes (`web/src/types/generated.ts`) are updated.
- [x] Hard-coded English strings in TaskDetailModal, TaskEditModal, CreateTaskModal, ColumnDetailModal and AboutModal use `t()` keys, with defaults in `defaultTranslations`.
- [x] i18n unit tests cover the new keys and config overrides.
- [x] `bun run check`, `bun test`, `bun run test:e2e` and `go test ./...` pass.

## Notes

### [2026-10-05 06:51 UTC]

Implemented. Strategy: `ui-translations-are-a-keyvalue-map`.

- **Backend:** translations are now `map[string]string`. I removed `model.TranslationsConfig`, and `mergeTranslations` skips blank values. A new rule, **CFG-010** in `validate.go`, rejects keys that aren't in the embedded `default.toml`. `default.toml` now lists all 248 UI keys, up from 51, so the calendar, weekday and month keys are configurable too. gentypes now emits `translations?: Record<string, string>`.
- **Frontend:** the five modals use `t()`, plus the new `tf()` and `tParts()` `{name}` interpolation helpers. A translation can reorder words, and `tParts` interpolates JSX such as the `<code>` nodes in ColumnDetailModal. The rendered English and the markup are unchanged. `no_terms_found` and `no_strategies_found` were moved from inline fallbacks into `defaultTranslations`. The example placeholders (tag list, sample task IDs) are intentionally left untranslated.
- **Tests:** Go tests cover the map merge, CFG-010 in both Validate and Load, and the defaults validating. The i18n tests cover modal defaults, overrides, `tf` and `tParts`, and a **parity test** that keeps `default.toml` and `defaultTranslations` identical.
- **Verified:** `go test ./...`, `bun run check`, `bun run typecheck`, `bun test` (79 pass) and Playwright E2E (52 pass) are all green. `jokateko parse` passes on this repo, and a typo key fails with CFG-010.

## Completion Summary
- **Completed At:** 2026-10-05T07:30:18Z

### What Was Done
- Backend: replaced the fixed `model.TranslationsConfig` struct with `map[string]string` in config, merge (blank values keep defaults), snapshot and gentypes output (`translations?: Record<string, string>`). New rule CFG-010 rejects keys that aren't in the embedded `default.toml`, which lists every UI key with its English default.
- Frontend: the TaskDetail, TaskEdit, CreateTask, ColumnDetail and About modals use `t()` keys. Added the `tf()` and `tParts()` `{name}` interpolation helpers, so a translation can reorder words and embed JSX nodes. `no_terms_found` and `no_strategies_found` moved from inline fallbacks into the defaults.
- Tests: Go merge, Validate and Load tests for CFG-010. i18n unit tests for the modal defaults, overrides and interpolation, plus a parity test that keeps `default.toml` and `defaultTranslations` identical.

### Why / Rationale
A fixed struct needed Go changes for every new UI label, so most labels (calendar, filters and others) were never configurable. With a map plus a key list in `default.toml`, a new label only touches the frontend dictionary and the TOML, while validation still catches typos. The user chose to reject unknown keys. Placeholders avoid gluing prefix and suffix keys together, which breaks word order in languages like Finnish. The decision is recorded in the strategy `ui-translations-are-a-keyvalue-map`.
