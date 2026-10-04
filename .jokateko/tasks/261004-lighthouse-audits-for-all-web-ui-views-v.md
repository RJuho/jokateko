+++
title = 'Lighthouse audits for all Web UI views via Playwright Chromium'
status = 'done'
priority = 'medium'
tags = ['e2e', 'lighthouse', 'testing', 'web-ui']
summary = 'Lighthouse audits all 11 Web UI views (including deep links) via Playwright Chromium over CDP. All views pass the default minimums after lazy Mermaid loading, SEO and accessibility fixes, Lucide icons and diagram pan/zoom.'
created_at = '2026-10-04T17:57:53Z'
changed_at = '2026-10-04T19:37:04Z'
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

## Phase 2: Fix findings (plan, awaiting approval)

### Performance: lazy-load Mermaid (no major refactor)
Mermaid and its dependencies (elkjs 1.4 MB, @mermaid-js/parser, cytoscape, katex, …) are about **5.1 MB of the 5.46 MB JS bundle**. The app itself plus Preact and Valibot is about 0.3 MB. Mermaid is only reached via the async `renderMermaidDiagrams()` in `web/src/utils/mermaid.ts`, so deferring it does not touch any components. The Lighthouse "Legacy JavaScript" hint (`@babel/plugin-transform-classes`, `Math.hypot`, ~8 KiB) comes from **cytoscape's prebuilt dist**, a Mermaid dependency. We cannot fix that in our build, but deferring Mermaid removes it from startup.

Measured (desktop preset, 4 views):

| Variant | HTML (gzip) | Performance |
|---|---|---|
| Today: all inline | 5.6 MB (1.5 MB) | 78–83 |
| Mermaid as inert inline `<script type="text/plain">`, executed on demand | 5.6 MB (1.5 MB) | 86–89 |
| Mermaid as separate same-origin file, loaded on demand | 0.46 MB (86 KB) | **97–100** |

Plan:
- `web/scripts/bundle.ts`: second Bun.build entry `src/vendor/mermaid-entry.ts` (IIFE that sets `window.__jokatekoMermaid`), written to `dist/mermaid.js` (+ `.gz`). The main bundle no longer imports `mermaid`. Keep `bundled-packages.json`/licenses covering both.
- `web/src/utils/mermaid.ts`: `loadMermaid()` returns a cached promise.
  - If an inert `#jokateko-mermaid` block exists (static export), it injects an inline `<script>` from that text.
  - Otherwise it appends `<script src="./assets/mermaid.js">`.
  - `initialize`/`render` await it.
- Go `serve`: embed `dist/mermaid.js(.gz)` and serve it at `/assets/mermaid.js` (gzip, long cache). The default CSP is `script-src 'self'`, so no CSP change is needed.
- Go `build` (`internal/exporter`): append the inert `<script type="text/plain" id="jokateko-mermaid">` block before `</body>`. The export stays a single offline file (≈86–89 perf, acceptable for an export). Use a function replacer: Mermaid code contains `$&`/`` $` `` sequences.
- Optional: check board CLS 0.109 (board scored 97 rather than 100 without Mermaid).

### SEO (82 → 100)
- Add `<meta name="description">` to `web/index.html`.
- Serve a valid `/robots.txt` from the Go server; today it falls through to the SPA HTML.

### Accessibility: no colour changes, look kept
- **label** (task modal): server-rendered criterion checkboxes (`internal/parser/markdown.go`) have no accessible name. Add `aria-label` (e.g. the item text via client-side enhancement, falling back to `Criterion N` from Go).
- **landmark-one-main**: Board and Calendar already use `<main>`. Change the Strategies and Glossary view root to `<main>`, with the same classes.
- **heading-order** (strategy card `h3` directly under the page `h1`): use `h2` with the same classes, so the visual is unchanged.
- **Nested interactive**: the strategy card's tag badges are `<button>`s inside the card `<button>`, which is invalid HTML. Make them siblings, keeping the same layout.
- **target-size** (24×24 minimum):
  - Strategy tag badges (20 px): wrap in a transparent button with vertical padding. The badge looks the same, and the hit area becomes 24 px.
  - Calendar week-number button (24×20, partly obscured): give it `min-h-6`.
  - Calendar task chips (17.8 px tall): give them `min-h-6`. This is a **small visual change** (+6 px per chip); decide before doing it.
- **color-contrast**: left as is per decision (do not change colours). It remains the only failing a11y audit. Board already scores 96 with only that failure, so the other views are expected to land around 94–96 after the fixes above.

- [x] Mermaid split: separate `mermaid.js` asset (serve) + inert inline block (static export) + `loadMermaid()`
- [x] SEO: meta description + valid robots.txt
- [x] A11y: checkbox labels, `<main>` landmarks, strategy heading level + un-nested tag buttons, target sizes
- [x] Lighthouse suite green on default minimums; e2e (incl. mermaid.spec / static-export.spec) still passing

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

### [2026-10-04 18:34 UTC]

**Evaluated: load Mermaid from the jsDelivr CDN with a pinned SRI hash, instead of a local lazy file.**

Findings:
- `mermaid.esm.min.mjs` (the file the Mermaid docs link to) is a 30 KB loader. It statically imports 9 chunks and dynamically imports 37 more, from 208 chunk files. SRI (`integrity=`) only covers the top-level file, so the chunks would load unverified.
- `+esm` is generated by jsDelivr, not the bytes Mermaid published. Avoid pinning a hash to generated output.
- The only file a single hash fully covers is `dist/mermaid.min.js` (5.49 MB, 0 dynamic imports). It is the same size as our own lazy split.
- CDN trade-offs:
  - Breaks offline use, including the "100% self-contained" `jokateko build` export.
  - Sends the user's IP to a third party.
  - Widens CSP `script-src`.
  - Adds a pinned-version and hash file plus a CI check to maintain.
  - Gains: the binary is about 1.5 MB (gzip) smaller. There is no Lighthouse difference, because both options are lazy.
- The local lazy split already has the "pinned version + hash" property: `bun.lock` pins `mermaid@12.1.0` with `sha512-wlVCp+8e…` integrity, verified on `bun install --frozen-lockfile`. Upgrades are an explicit `bun update mermaid` that shows up in the diff.
- "Just fetch latest" (unpinned) rules out SRI, and is a supply-chain risk. Rejected.

Recommendation: keep the Phase 2 local lazy split. If a CDN is wanted later, use a pinned exact version, `dist/mermaid.min.js` only, SRI plus `crossorigin`, and a CI check that compares the CDN bytes with the npm tarball file pinned in `bun.lock`.

### [2026-10-04 19:01 UTC]

**Phase 2 implemented** (decision: daemon serves its own Mermaid file; `jokateko build --mermaidjs=bundled|cdn|none`, default `cdn`).

**Mermaid runtime.** All modes use the published, self-contained `mermaid/dist/mermaid.min.js` from the `bun.lock`-pinned package (12.1.0). `bundle.ts` computes its sha384 SRI hash at build time, so no manually maintained hash is needed. I verified that jsDelivr serves byte-identical content with the same hash and CORS `*`.
- `web/src/utils/mermaidLoader.ts`: lazy `loadMermaid()`. The mode comes from `<meta name="jokateko-mermaid">`; the default is the daemon asset. On failure (offline, SRI mismatch) the raw code blocks are kept.
- Daemon: embedded `dist/mermaid.min.js.gz` is served at `GET /assets/mermaid-<version>.min.js` (gzip, immutable cache, versioned URL). The script is loaded with `integrity` and `crossorigin`. CSP `script-src` also gets the SRI hash, in addition to `'self'`.
- Export modes:
  - `cdn`: jsDelivr URL for the pinned version, with SRI.
  - `bundled`: inert `<script type="text/plain">` block, executed on first use.
  - `none`: no runtime.
- Bug found and fixed: in `bundled` mode a deep link (`#task/x`) asked for Mermaid while the browser was still parsing the 5.5 MB block, so `textContent` was partial ("Unexpected end of input"). The loader now waits for `DOMContentLoaded`.
- App HTML: 5.6 MB to 458 KB (86 KB gzip).

**SEO:** meta description, plus `/robots.txt` served as text.

**Accessibility (no colour changes):**
- Criterion checkboxes get `aria-label` with the item text (Go renderer, escaped).
- `<main>` landmark in Strategies and Glossary.
- Strategy card title `h3` changed to `h2`, same classes.
- Strategy card no longer nests tag buttons inside the toggle button. It uses the stretched-button pattern: the toggle button's `::after` covers title, tags and summary; the tags row is `relative z-10 self-start`. The whole card area still toggles, and the look is unchanged.
- Not changed: calendar week-number and task-chip target spacing, which would be a visual change, and colour contrast.

**Final Lighthouse (default minimums, all 11 pass):**

| Views | Perf | A11y | BP | SEO |
|---|---|---|---|---|
| board, board-task, board-milestone | 97 | 96 | 100 | 100 |
| calendar month / week / milestone | 100 | 91 | 100 | 100 |
| calendar-task | 100 | 92 | 100 | 100 |
| strategies, strategy-detail, glossary, glossary-term | 100 | 96 | 100 | 100 |

**Tests added:**
- Go: server asset, gzip, CSP hash, SRI hash matching the bytes, robots; `InjectMermaid`, `ParseMermaidMode`, `Export` in all modes; CLI `--mermaidjs` valid and invalid; checkbox `aria-label`.
- e2e `static-export-mermaid.spec.ts`:
  - `bundled` renders offline with zero requests.
  - `cdn` requests the pinned URL, with the response served from the local npm file, and SRI passes.
  - A tampered CDN file is blocked and the raw block is kept.
  - `none` keeps the raw block.
- e2e `strategy-card.spec.ts`: tag filters without toggling, a summary click toggles, Enter toggles, no nested buttons.

**Results:** `go test ./...` OK; e2e 51/51; Lighthouse 11/11; biome, tailwind and typecheck clean; `make install` done.

**Follow-up (not changed here):** regenerating licenses would add elkjs and chevrotain (shipped, missing today) but drop `fastdom`. `fastdom` is prebundled inside mermaid's dist and not in `bun.lock`. The license files were left as committed; this needs a decision on how to attribute packages prebundled in Mermaid's dist.

### [2026-10-04 19:22 UTC]

**Round 3: licenses, Lucide icons, diagram pan/zoom**

**Licenses (decision: show only our main packages; Mermaid is always listed, even with `--mermaidjs=none`).**
- `cmd/genlicenses` now lists the `dependencies` in `web/package.json`, plus the direct (non-`// indirect`) `go.mod` requires that are linked into the binary. Result: 10 entries (5 Go, 5 npm: @preact/signals, lucide-preact, mermaid, preact, valibot).
- Removed the Bun metafile harvesting and `dist/bundled-packages.json` (it replaced the approach from task 260906). This also resolves the earlier fastdom/elkjs attribution question.
- Unit test for `mainNpmPackages`. The About e2e test now asserts main packages are present and transitive ones (dompurify, golang.org/x/sys) are absent.

**Icons:** added `lucide-preact` 1.52.0 (ISC, approved). Replaced all 28 inline SVGs and the 7 `✕` glyphs across 15 components, plus the Mermaid toolbar and lightbox.
- Size classes and stroke weights are kept.
- Informational icons keep `<title>` (Blocked, Checklist progress, Target date, theme).
- Lucide has no brand icons, so the GitHub link uses `FolderGit2`.
- The broken expand icon is now `Maximize2`/`Minimize2` (task modal and diagram).
- App HTML went from 458 KB to 374 KB (80 KB gzip).

**Diagram pan/zoom bug.** Root cause: zoom used `transform: scale()` on a centred element. The layout and scroll area never grew, so the zoomed left and top parts could not be scrolled to, and there was no drag handling at all.
- New `web/src/utils/diagramViewport.ts`, shared by the card and the lightbox:
  - Zoom resizes the SVG's layout box, and `justify-center-safe` keeps every edge reachable.
  - Mouse and pen drag pans with a grab cursor; touch and the plain wheel scroll natively.
  - Ctrl/⌘ + wheel (and trackpad pinch) zooms around the cursor.
  - The viewport is focusable: arrows scroll; `+`, `-` and `0` zoom.
  - 100% means fit: the card fits the width without upscaling; the lightbox fits the whole diagram.
  - Re-fits on resize and on theme re-render.
- The lightbox now has its own zoom toolbar.
- New e2e test checks: zoomed width over 2.9× the fit width, the scroll area grows, left and top edges are reachable, drag pans, keyboard zoom, Ctrl+wheel zoom (the plain wheel does not zoom), and the lightbox fits.

**Results:** Go tests OK; e2e 52/52; Lighthouse 11/11 (same scores); biome, tailwind and tsc clean; `make install` done. Checked visually with screenshots of the board, the task modal toolbar and the lightbox.

## Completion Summary
- **Completed At:** 2026-10-04T19:37:04Z

### What Was Done
**Lighthouse suite**
- `make lighthouse-test`: `tests/lighthouse/` plus `playwright.lighthouse.config.ts`, using the dev-only `lighthouse` dependency.
- Covers 11 views and deep links. Each test does a fresh-load warm-up and asserts `data-testid` and `document.title`.
- Category minimums are env-configurable (`LH_MIN_*`); HTML and JSON reports go to `lighthouse-report/`.

**Mermaid runtime**
- Mermaid is removed from the app bundle. The pinned `mermaid/dist/mermaid.min.js` is embedded with a build-time SRI hash (`mermaid.json`).
- The daemon serves it lazily at `/assets/mermaid-<version>.min.js`, with SRI and the hash added to CSP.
- `jokateko build --mermaidjs=cdn|bundled|none`, default `cdn` (pinned jsDelivr URL with SRI).
- Fixed a bundled-mode race on deep links.

**SEO:** meta description and `/robots.txt`.

**Accessibility (no colour changes)**
- Checkbox `aria-label`s.
- `<main>` landmarks.
- Strategy card `h2`, with a stretched-button card that no longer nests buttons.
- Fixed the glossary deep link not expanding on fresh load.

**Licenses:** only main packages are listed (`package.json` dependencies plus direct `go.mod` requires). Mermaid is always listed.

**Icons:** `lucide-preact` replaces all inline SVG and `✕` icons.

**Diagram pan/zoom (`diagramViewport.ts`)**
- Zoom resizes the layout, so every edge stays reachable.
- Drag pans; Ctrl/⌘ + wheel or pinch zooms; `+`, `-` and `0` keys; fit is 100%.
- The lightbox has its own controls.

**Results:** app HTML 5.6 MB → 374 KB. Final scores: Perf 97–100, A11y 91–96, BP 100, SEO 100. New Go and e2e tests; e2e 52/52.

### Why / Rationale
- **Lighthouse:** direct CDP integration reuses the Playwright Chromium we already have, and gives deterministic, repeatable quality gates for every view.
- **Mermaid:** it is about 95% of the JavaScript, so loading it on demand fixes startup performance without a major refactor. One hash derived from `bun.lock` keeps the CDN, inline and embedded copies byte-identical and verifiable, in line with the security priority. The export mode lets users choose a small file versus fully offline.
- **Accessibility:** fixes are markup-only, because the human decided the look and colours are final.
- **Licenses:** showing main packages only follows the human's decision.
- **Icons and pan/zoom:** Lucide was approved by the human. Pan/zoom replaces the transform-based zoom, which could not scroll to the zoomed content.
