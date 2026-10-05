+++
title = 'UI translations and project locale'
tier = 2
tags = ['backend', 'frontend', 'i18n']
summary = 'UI labels are a flat string map: default.toml [translations] lists every known key, and defaultTranslations in i18n.ts holds the same keys, with unknown keys rejected by CFG-010. Dates, weekday and month names come from Intl, using the optional [project] locale or else the browser locale.'
+++

## Decision

Translations are `map[string]string` end to end. There is no Go struct with one field per label.

- **Go:** `config.Config.Translations` and `model.SnapshotConfig.Translations` are `map[string]string`. gentypes emits `translations?: Record<string, string>`, and the Valibot schema is `v.record(v.string(), v.string())`.
- **Known keys:** the `[translations]` table in `internal/config/default.toml` is the backend's single list of valid keys. `Validate` rejects any other key with **CFG-010**, so a typo fails `jokateko parse` and server start.
- **Merge:** `mergeTranslations` copies non-blank values from the user config over the defaults. Blank values keep the default.
- **Frontend:** `defaultTranslations` in `web/src/utils/i18n.ts` has exactly the same keys and English values. `t(key)` reads the config first, then the default.

## Locale: dates, weekday and month names

- `[project] locale` is an optional BCP 47 tag such as `fi-FI`. It is validated by **CFG-011** with a shape check. That check is a regex, because only stdlib may be used and `x/text` would be a new dependency.
- `uiLocale()` in `i18n.ts` returns the canonical tag via `Intl.getCanonicalLocales`, or `undefined` when the locale is unset or invalid. `undefined` means the browser locale.
- **Every** `Intl.DateTimeFormat` or `toLocale*` call in the Web UI passes `uiLocale()`. The helpers in `web/src/utils/date.ts` take an optional `locale` argument and stay free of store imports.
- Weekday and month names come from `weekdayName()` and `monthName()` in `date.ts`, built on Intl. They are **not** translation keys. Browser capitalisation is kept as-is, so Finnish shows "ma" and "syyskuu".
- `initDocumentLang()` keeps `<html lang>` equal to the project locale.
- Playwright pins the browser locale to `en-US`. Unit tests pass explicit locales.

Timezones are not handled yet. Times are shown in each viewer's own timezone.

## Delivery to the UI

- **Static export:** these settings travel in the embedded `SnapshotConfig` (`project.locale`, `translations`).
- **Live server:** the UI builds its config from `GET /api/board`, so `model.BoardState` carries `locale` and `translations`, and `fetchLiveBoard` copies them into `config`. Any new UI-facing config field must be added to **both** paths. The MCP `get_board_state` tool uses its own output struct and does not expose these fields.

## Rules

1. To add a UI string, add the key to `defaultTranslations` **and** to `default.toml` with the same value. No Go code changes are needed. The parity test `default.toml parity` in `web/src/utils/i18n.test.ts` fails if the two differ.
2. Name keys `arial_*` for `aria-label`, `title` and screen-reader-only text. Use an area prefix for visible text, such as `task_`, `column_` or `about_`. Shared words like `cancel`, `close` and `copy` are unprefixed.
3. Never build sentences from separate prefix and suffix keys. Use `{name}` placeholders with `tf(key, vars)` for strings, or `tParts(key, vars)` when a placeholder is a JSX node such as `<code>`.
4. Don't pass a fallback to `t(key, fallback)` for a real UI key. Put the English text in `defaultTranslations` so the key can be configured.
5. Never add translation keys for things Intl can produce, such as weekday names, month names or date formats. Use the `date.ts` helpers with `uiLocale()`.
6. Leave example data in placeholders untranslated, such as sample tag lists or sample task IDs.

## Why

With the fixed struct, every new label needed Go changes, so many labels were never configurable. With a map, a new label only touches the frontend dictionary and `default.toml`, and validation against `default.toml` still catches typos. Intl gives correct names and date formats for every language, so there are no hand-maintained month or day lists. A single project locale also means the whole team sees the same date formats on the timeline.
