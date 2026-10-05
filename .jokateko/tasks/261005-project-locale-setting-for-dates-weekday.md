+++
title = 'Project locale setting for dates, weekdays and month names'
status = 'done'
priority = 'medium'
tags = ['backend', 'frontend', 'i18n']
summary = 'The optional [project] locale drives all date formatting and the calendar weekday and month names via Intl, falling back to the browser locale. Translations and locale now reach the live UI via /api/board, and every screen-reader label is translatable.'
created_at = '2026-10-05T07:00:00Z'
changed_at = '2026-10-05T07:30:24Z'
+++

## Context

This follows `261004-configurable-ui-translations-for-modal-s`. The weekday and month names were 40 translation keys (`day_*`, `month_*`), while the dates beside them in the UI were formatted in the browser locale, so the two could disagree. A project-wide locale keeps everyone on a team seeing the same date formats on the timeline. Timezone handling is deliberately out of scope and will be done later.

## Acceptance Criteria
- [x] `[project] locale` config (optional BCP 47 tag, e.g. `fi-FI`), validated with a new CFG rule, merged, exported in the snapshot and in gentypes/Valibot.
- [x] All Web UI `Intl` / `toLocale*` date formatting uses the project locale, falling back to the browser locale when it is unset or invalid.
- [x] Calendar weekday and month names come from `Intl.DateTimeFormat`. Browser capitalisation is kept.
- [x] The `day_*` and `month_*` translation keys are removed from `defaultTranslations` and `default.toml`.
- [x] `<html lang>` follows the project locale when it is set.
- [x] Tests cover locale validation, the Intl names and the fallback. Playwright runs with a fixed `en-US` locale.
- [x] `bun run check`, `bun test`, `bun run test:e2e` and `go test ./...` pass.

## Notes

### [2026-10-05 07:09 UTC]

Implemented. Strategy `ui-translations-are-a-keyvalue-map` was updated and renamed to "UI translations and project locale".

- **Config:** added `[project] locale` (empty = browser locale), validated by **CFG-011**, which is a regex shape check because no new dependencies are allowed. It's exported in `SnapshotConfig.project.locale`. `default.toml` documents it, and `jokateko init` writes `locale = ""`.
- **UI:** a new `uiLocale()` in `i18n.ts` returns the canonical tag, or `undefined` (browser) when the locale is unset or invalid. The `date.ts` helpers take a `locale` argument. The calendar uses the new `weekdayName()` and `monthName()`, and the task timestamps, target-date badge and build-time indicator all use `uiLocale()`. `TaskDetailModal` now reuses `formatBrowserDateTime` instead of its private copy. `initDocumentLang()` keeps `<html lang>` in sync.
- **Removed 38 keys:** the weekday names (short and long) and month names (long and short).
- **Live-mode gap fixed (it existed before this work):** the live UI builds its config from `GET /api/board`, which never carried `translations`. So `[translations]` overrides from `config.toml` only worked in static exports, never on the live server. `BoardState` now carries `locale` and `translations`, and `fetchLiveBoard` applies them. The MCP `get_board_state` output is unchanged because it has its own struct.
- **Tests:** CFG-011 and loading in Go, the snapshot export, and the `/api/board` fields. Unit tests cover the Intl names and titles in en-US and fi-FI, the `uiLocale` fallbacks, and an explicit-locale `formatBrowserDateTime`. The new `tests/e2e/locale.spec.ts` covers fi-FI calendar names, `<html lang>`, timestamps and a live translation override. Playwright is pinned to `en-US`.
- **Verified:** `go test ./...`, `bun run check`, `typecheck`, `bun test` (86) and Playwright (55) all pass. `parse` passes on this repo and rejects `fi_FI` with CFG-011.
- **Not done:** timezone handling (deliberately out of scope), and the `(N tasks)` text in a calendar aria-label is still hard-coded.

### [2026-10-05 07:24 UTC]

Follow-up on request: **all screen-reader labels are now translatable**, not only the modals.

- **Key changes:** added 47 `arial_*` keys covering every remaining `aria-label`, `title` tooltip and SVG `<title>`. That's in TaskCard, Column, KanbanBoard, MilestoneCards, CalendarView, StrategiesView, GlossaryView, FilterBar, Badge, Header, ModeIndicator and ValidationBanner. Labels that include values use `{placeholders}`, for example `arial_calendar_day: '{day} {date} ({count} tasks)'` and `arial_scroll_to_column: 'Scroll to column {name} ({count} tasks)'`. `task_id_copied` was renamed to a shared `id_copied`, now also used for the task-card, strategy and glossary "copied!" text. The total is 257 keys, kept in sync with `default.toml`.
- **Behaviour:** the English output is unchanged and so is the markup.
- **Guard test:** `screen-reader labels › has no hard-coded English labels left in components` in `i18n.test.ts` fails if a component gets a literal or template-literal `aria-label`, `title` or `alt`, or a literal SVG `<title>`.
- **Verified:** `go test ./...`, `bun run check`, `typecheck`, `bun test` and Playwright (55) all pass. `parse` is OK.
- **Flaky test found:** while verifying, `internal/proxy` `TestProxy_LegacyInitializeSurvivesSwitches` hung once for the full 10-minute timeout. It also hangs on clean HEAD 5470573 (in 1 of 3 batches of 40 runs), so it is pre-existing and unrelated to this work. Logged as `261005-flaky-hang-in-testproxy-legacyinitialize`.

## Completion Summary
- **Completed At:** 2026-10-05T07:30:24Z

### What Was Done
- Config: `[project] locale` (BCP 47, validated by CFG-011 with a regex) is exported in the snapshot and in `/api/board`. `/api/board` now also carries `translations`, fixing an existing gap where config translations never reached the live UI.
- UI: `uiLocale()` returns the canonical tag, or `undefined` to use the browser locale. The `date.ts` helpers (`formatBrowserDateTime`, `weekdayName`, `monthName`, `formatMonthYearTitle`, `formatWeekTitle`) take a locale, and the calendar, badges, timestamps and build indicator use it. `initDocumentLang()` syncs `<html lang>`. Removed 38 `day_*` and `month_*` keys.
- Screen-reader labels: 47 new `arial_*` keys cover every remaining `aria-label`, `title` and SVG `<title>`. `task_id_copied` was renamed to a shared `id_copied`. A guard unit test fails on hard-coded labels.
- Tests: Go tests for CFG-011, the snapshot and `/api/board` fields. Unit tests for the Intl names and titles in en-US and fi-FI and the `uiLocale` fallbacks. A new E2E spec `locale.spec.ts` covers fi-FI calendar names, `lang`, timestamps and a live translation. Playwright is pinned to en-US.

### Why / Rationale
Intl gives correct weekday names, month names and date formats for every language, so there are no hand-maintained lists. A single project locale also keeps everyone on a team on the same date formats, while timezones are left for later. The live UI builds its config from `/api/board`, so UI settings must be delivered there as well as in the snapshot. That rule is recorded in the strategy `ui-translations-are-a-keyvalue-map`.
