+++
title = 'Lighthouse audits for all Web UI views via Playwright Chromium'
status = 'in_review'
priority = 'medium'
tags = ['e2e', 'lighthouse', 'testing', 'web-ui']
summary = 'Integrate Lighthouse directly with Playwright-launched Chromium over CDP (--remote-debugging-port) and audit every Web UI view and deep-linked resource against configurable category thresholds.'
created_at = '2026-10-04T17:57:53Z'
changed_at = '2026-10-04T18:08:04Z'
+++

## Approach
Direct integration pattern (https://unlighthouse.dev/learn-lighthouse/playwright): Playwright launches Chromium with `--remote-debugging-port=<port>`, then `lighthouse(url, { port })` attaches over CDP and audits. Single worker (shared debugging port). `lighthouse` is a dev-only dependency (human-approved); never shipped in the binary/bundle.

Runs as a separate suite (`make lighthouse-test`, `playwright.lighthouse.config.ts`, `tests/lighthouse/`) so `make e2e-test` stays fast.

## View matrix
| Name | Hash |
|---|---|
| board | `#board` |
| board-task | `#task/<taskId>` |
| board-milestone | `#milestone/<milestoneId>` |
| calendar-month | `#calendar/<YYYY-MM>` |
| calendar-week | `#calendar/<YYYY-Www>` |
| calendar-task | `#calendar/<YYYY-MM>/task/<taskId>` |
| calendar-milestone | `#calendar/<YYYY-MM>/milestone/<milestoneId>` |
| strategies | `#strategies` |
| strategy-detail | `#strategy/<id>` |
| glossary | `#glossary` |
| glossary-term | `#glossary/<id>` |

Each view is warmed up with Playwright first and its expected `data-testid` asserted visible, proving the deep link resolved before scoring.

## Thresholds
Env-configurable minimums: `LH_MIN_PERFORMANCE` (90), `LH_MIN_ACCESSIBILITY` (90), `LH_MIN_BEST_PRACTICES` (90), `LH_MIN_SEO` (80). Soft-asserted per category; failing audit ids logged.

## Acceptance Criteria
- [x] `lighthouse` added as a web devDependency (approved)
- [x] Helper `tests/lighthouse/helpers/audit.ts` (launch with CDP port, run audit, save report, assert thresholds)
- [x] Spec covering Board, Calendar (month, week), Strategies, Glossary, plus deep links (task, milestone, strategy, glossary term, calendar task/milestone)
- [x] Separate Playwright config and `make lighthouse-test` target
- [x] Thresholds configurable via env; HTML + JSON reports written to `lighthouse-report/`
- [x] Report dir gitignored and cleaned by `make clean-cache`
- [x] Docs updated (docs/README testing section)

## Notes

### [2026-10-04 18:08 UTC]

Implemented. `make lighthouse-test` audits 11 views in ~2 min (single worker) and writes 22 reports to `lighthouse-report/`.

**Baseline scores (desktop preset)** — the suite currently FAILS against the default minimums (perf 90 / a11y 90 / bp 90 / seo 80); these are genuine findings, not test bugs:

| View | Perf | A11y | BP | SEO |
|---|---|---|---|---|
| board | 78 | 96 | 100 | 82 |
| board-task | 78 | 91 | 100 | 82 |
| board-milestone | 79 | 96 | 100 | 82 |
| calendar-month | 82 | 91 | 100 | 82 |
| calendar-week | 82 | 91 | 100 | 82 |
| calendar-task | 82 | **87** | 100 | 82 |
| calendar-milestone | 81 | 91 | 100 | 82 |
| strategies | 80 | **88** | 100 | 82 |
| strategy-detail | 83 | **88** | 100 | 82 |
| glossary | 83 | 94 | 100 | 82 |
| glossary-term | 82 | 94 | 100 | 82 |

- Performance: first-contentful-paint (~58), LCP, TBT, speed-index — consistent with the ~5.6 MB single-file inlined bundle (mermaid etc.).
- Accessibility: color-contrast, label (calendar task modal), target-size, heading-order and landmark-one-main (strategies).
- Follow-ups: either fix these or decide on baseline minimums (env vars `LH_MIN_*`).

**Bug fixed along the way:** `GlossaryView` cleared opened terms in its `[searchQuery]` effect on mount, so `#glossary/<id>` did not expand the term on a fresh page load (only on same-document hash change). Now keeps the URL term open (same as StrategiesView).

**Runner gotcha:** under the Bun-backed Playwright runner, `*.spec.ts` entry files are parsed as plain JS — no type annotations, `type` aliases or type-only imports (fails with an opaque "AggregateError: N errors building"). Helpers may use TS but derive types via `ReturnType<typeof …>`. Documented in docs/web-ui-architecture.md §7.1.

Verified: env overrides pass (LH_MIN_PERFORMANCE=75 LH_MIN_ACCESSIBILITY=85), `make e2e-test` 46 passed (unaffected), `bun run check` + `typecheck` clean, `make install` done.
