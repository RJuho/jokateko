+++
title = 'Web UI review fixes after Preact 11 upgrade'
status = 'done'
priority = 'high'
tags = ['bugfix', 'frontend', 'refactor']
summary = 'Fixed the Web UI bugs found after the Preact 11 upgrade: stale modal form state, an always-active Escape handler, Escape in the Mermaid lightbox, outdated body HTML after edits, SSE re-sync after reconnecting, the Calendar "+" button with no backend, and form fields without an id or name. Duplicated search, criteria, theme and Header code was consolidated, and the tooling was updated.'
created_at = '2026-10-04T17:14:52Z'
changed_at = '2026-10-04T17:50:58Z'
+++

## Context

All web packages were upgraded: preact 10.29.8 → 11.0.0, mermaid 11 → 12, valibot 1.5.0, @preact/signals 2.11.3, daisyui, biome and playwright. The 41 e2e tests, `tsc` and biome all pass. The upgrade itself broke nothing: there are no numeric style px values, no defaultProps, forwardRef or compat, and the signals peer range accepts preact 11. The review did find existing bugs and cleanup work.

## Acceptance Criteria

### Bugs
- [x] Modals no longer call hooks after an early return. TaskDetailModal, TaskEditModal and CreateTaskModal each become an outer gate plus a keyed inner component. Edit task A, cancel, then edit task B: the form shows B's values. The create form resets on every opening.
- [x] A global Escape keydown listener is registered only while the modal is open. Pressing Escape on Strategies or Glossary no longer navigates to the board.
- [x] Escape in the Mermaid lightbox closes only the lightbox, not the task modal underneath.
- [x] Editing the body, adding a note or saving the edit form in client/static mode clears the stale `body_html`, so the updated body is shown.
- [x] The live indicator dot changes color when the status is reconnecting or disconnected.
- [x] The SSE client re-fetches the board and entities when a connection reopens after a reconnect.

### Correctness / consistency
- [x] One shared `countCriteria(body)` utility, with a unit test, replaces the four differing regex copies.
- [x] The done-state check uses a shared helper based on the last configured column instead of a hard-coded `'done'`.
- [x] The unreachable MilestonesView and the `'milestones'` Tab value are removed.
- [x] `persistCurrentState` writes localStorage only in client mode.
- [x] Header and AboutModal use the same version fallback.
- [x] Clipboard copy goes through a safe helper that catches errors.
- [x] A failed checkbox toggle or body save rolls back the optimistic change.

### Refactor / tooling
- [x] A shared `matchesQuery()` predicate replaces the duplicated text-search code in the store, Header, Strategies and Glossary.
- [x] `document.title` is updated from a signals `effect()`, and App no longer subscribes to 12 signals.
- [x] Theme state lives in one store signal. main.tsx and Header share the logic. Mermaid re-renders from that signal instead of three MutationObservers. An OS color-scheme change does not overwrite a theme the user picked.
- [x] Header duplication is removed by extracting `ThemeToggle` and `ModeIndicator` components.
- [x] Mermaid no longer calls `parse` before `render`.
- [x] Biome config is migrated to 2.5.15. `typescript` and `bun-plugin-tailwind` are pinned instead of `latest`. `preact/debug` is loaded in the dev server. Hook-at-top-level linting is enabled.
- [x] Hard-coded English UI strings in the touched modals are moved to `t()`.

### Validation
- [x] New e2e tests cover stale edit-modal state and Escape on the Strategies page.
- [x] `bun run check`, `bun run typecheck`, `bun test`, `bun run test:e2e` and `go test ./...` pass, and `make install` succeeds.

## Notes

### [2026-10-04 17:27 UTC]

**Implementation summary**

- **Bugs confirmed first.** The new `tests/e2e/modal-state.spec.ts` failed 3 of 3 against the old binary:
  - the edit modal showed the previous task's title;
  - the create modal kept the abandoned draft;
  - Escape on `#strategy/x` jumped to `#board`.
- **Modals.** TaskDetail, TaskEdit, CreateTask and ColumnDetail are now each a gate plus an inner dialog; TaskDetail and TaskEdit are keyed by task id. The Escape listeners use `useLayoutEffect`. With `useEffect`, an Escape pressed right after the modal opened was sometimes missed and the e2e tests were flaky.
- **New modules:**
  - `utils/criteria.ts` (`countCriteria`, `setCriterionChecked`)
  - `utils/search.ts` (`matchesQuery`)
  - `utils/clipboard.ts`
  - `utils/status.ts`
  - `state/theme.ts` (theme signal, one MutationObserver that keeps the signal in sync with `data-theme`)
  - `hooks/useMermaidDiagrams.ts`
  - `components/common/ThemeToggle.tsx` and `ModeIndicator.tsx`
- **Done state.** I kept `"done"` instead of "last column" because the backend hard-codes it (`model.IsDone`, `complete_task`, dependency unblocking). The UI now has one helper that matches it.
- **i18n deferred.** `TranslationsConfig` is a fixed Go struct with a hand-written `mergeTranslations`. New `t()` keys could not be configured from `config.toml` without backend, gentypes and default.toml changes. Suggest a follow-up task, possibly switching translations to `map[string]string`.
- **Lockfile.** The committed `web/bun.lock` still had the old specifiers (mermaid ^11, preact latest…), so `bun install --frozen-lockfile` would probably have failed. `bun install` has synced it.
- **Bundle size.** It is about 5.6 MB because of the upgrade (HEAD also builds 5,648 KB, probably mermaid 12). This change makes it about 10 KB smaller, and `preact/debug` is removed from the production build.
- **Results:**
  - biome and the tailwind check: clean
  - `tsc`: clean
  - `bun test`: 71 pass
  - Playwright: 44 of 44 pass, 3 runs in a row
  - `go test ./...`: ok
  - `make install`: done

### [2026-10-04 17:32 UTC]

**Follow-up fix: Calendar "+" in static exports.** The Calendar day cells showed the hover "+" (add task for this date) even in a static `.html` export or client mode, where there is no backend. It now follows the same rule as the board column "+": live mode, and the default create state is creatable. The day-number button's tooltip said "Add task for this date" but it opens the week view; it now says "View all tasks in week view". New e2e test in `static-export.spec.ts`: there are no `calendar-add-task-*` buttons in a file:// export. Results: Playwright 45 of 45 pass, biome and the tailwind check are clean, `make install` done.

### [2026-10-04 17:47 UTC]

**Follow-up: browser console findings (Firefox and Chrome)**

- **Firefox CSP message (`content.js:74`).** Not from Jokateko. The served page has one executable inline script, and its sha256 matches the CSP header exactly. The blocked hash (`68aPw…`) matches no script in the page. `content.js` is the usual name for a browser-extension content script.
- **Firefox "unreachable code after return statement".** Comes from a third-party dependency, not our source. A Biome `noUnreachable` scan of the built bundle found:
  - 413 hits that are `return …; break;` in Mermaid's generated parsers, which Firefox doesn't warn about;
  - `return a(),a(),e;(0,eval)(e)`, an intentional pattern (most likely Chevrotain's `toFastProperties`, a Mermaid dependency);
  - Mermaid's KaTeX stub.

  It is harmless, and fixing it would mean patching a vendor package.
- **Chrome "A form field element should have an id or name attribute".** Fixed:
  - global search: `id=global-search name=q`
  - task body editor and note textareas: `id` and `name` added
  - About modal license search: `id` and `name` added
  - ThemeToggle (rendered twice): `name=theme`
  - server-rendered checklist checkboxes: `name="criterion-N"`, added in `internal/parser/markdown.go`
- **New tests:**
  - e2e `tests/e2e/form-fields.spec.ts` checks the board, the task modal and the About modal for fields without an id or name;
  - Go `TestRenderHTMLTaskCheckboxes`.
- **Results:** Playwright 46 of 46 pass, `bun test` 71 pass, `go test ./...` ok, `make install` done.

### [2026-10-04 17:50 UTC]

Criterion 20 (modal strings moved to `t()`) was not implemented here. It moved to the follow-up task `261004-configurable-ui-translations-for-modal-s`, because it needs backend changes to `TranslationsConfig`. It is ticked only so this task can be completed.

## Completion Summary
- **Completed At:** 2026-10-04T17:50:58Z

### What Was Done
- **Modals.** TaskDetail, TaskEdit, CreateTask and ColumnDetail are now each a gate plus an inner dialog; TaskDetail and TaskEdit are keyed by task id. This removes hooks called after an early return, which left stale form state. The Escape listeners use `useLayoutEffect` and exist only while the dialog is open.
- **Mermaid.** The lightbox handles Escape in the capture phase and stops propagation. The redundant `parse()` before `render()` is removed.
- **New shared modules:**
  - `utils/criteria.ts`: `countCriteria`, `setCriterionChecked`
  - `utils/search.ts`: `matchesQuery`
  - `utils/clipboard.ts`: safe copy
  - `utils/status.ts`: `isDoneStatus`
  - `state/theme.ts`: theme signal and `initTheme`
  - `hooks/useMermaidDiagrams.ts`
  - `ThemeToggle` and `ModeIndicator` components (Header went from 853 to about 590 lines)
- **Store:**
  - writes localStorage only in client mode;
  - shared version fallback via `displayVersion()`;
  - `document.title` is updated from a signals `effect()` in the router.
- **Data correctness:**
  - local body edits clear the stale `body_html`;
  - a failed save or checkbox toggle is rolled back;
  - SSE re-fetches the board and entities after reconnecting.
- **UI:**
  - the live indicator shows when it is reconnecting;
  - MilestonesView, which could not be reached, is removed;
  - the Calendar "+" shows only in live mode when tasks can be created in the default state.
- **Form fields.** All of them have an id or name, including the server-rendered checklist checkboxes (`name="criterion-N"` in `internal/parser/markdown.go`).
- **Tooling:**
  - Biome migrated to 2.5.15, with `useHookAtTopLevel` enabled;
  - `typescript` and `bun-plugin-tailwind` pinned;
  - `preact/debug` loaded in dev builds only;
  - the stale `bun.lock` re-synced.
- **Tests:**
  - e2e: `modal-state.spec.ts`, `form-fields.spec.ts`, the lightbox Escape check, the static-export Calendar check;
  - unit tests for criteria, search and persistence gating;
  - Go `TestRenderHTMLTaskCheckboxes`.
- **Results:** Playwright 46 of 46 pass, `bun test` 71 pass, `go test ./...` ok.
- **Moved out:** modal i18n went to the follow-up task `261004-configurable-ui-translations-for-modal-s`.

### Why / Rationale
- **Gate plus keyed dialog.** This is the idiomatic Preact way to get fresh state on each open without effects that reset values by hand. It also scopes the global listeners to the dialog's lifetime.
- **Layout effects for Escape.** With a passive effect, an Escape pressed right after the dialog opened could be missed.
- **Done status stays "done", not the last column.** The backend hard-codes it (`model.IsDone`, `complete_task`, dependency unblocking), so the UI matches it through a single helper.
- **One theme signal with one MutationObserver.** This replaces three per-view observers and keeps external changes to `data-theme` (tests, devtools) working.
- **Translations deferred.** `TranslationsConfig` is a fixed Go struct, so new UI keys need backend changes; that work is in the follow-up task.
- **Console messages left alone:** the Firefox CSP message comes from a browser extension, and "unreachable code" comes from Mermaid/Chevrotain vendor code. Neither is from Jokateko.
