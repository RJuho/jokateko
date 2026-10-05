+++
title = 'Web UI architecture'
tier = 2
tags = ['frontend']
summary = 'One Preact bundle, built by Bun into a single self-contained index.html and embedded in the binary, boots in live, static or client mode; state is signals, all external data is validated with Valibot, Mermaid is lazy-loaded and SRI-pinned, and tests target data-testid.'
+++

# Web UI architecture

## Stack

Preact 11 + `@preact/signals`, Valibot, Tailwind CSS v4 + daisyUI 5 + `@tailwindcss/typography`, `lucide-preact` icons, Mermaid. Built with the Bun bundler. Lint and format with Biome (`bun run check`). Exact versions are in `web/package.json` and `web/bun.lock`, and new UI dependencies need human approval.

## One bundle, three modes

`web/scripts/bundle.ts` produces **one** `index.html` with the CSS and JS inlined. The same file serves every mode. `state/bootstrap.ts` chooses the mode at startup:

```mermaid
flowchart TD
    Start["index.html loads"] --> Q1{"#jokateko-data holds a snapshot?"}
    Q1 -- yes --> Static["static: read-only snapshot<br/>(jokateko build)"]
    Q1 -- no --> Q2{"served over http(s)?"}
    Q2 -- yes --> Live["live: GET /api/board + entities,<br/>SSE /api/events, REST mutations<br/>(jokateko serve)"]
    Q2 -- no --> Client["client: browser-only,<br/>state persisted in localStorage"]
```

- **Snapshot injection.** The bundle contains `<script id="jokateko-data" type="application/json">/* JOKATEKO_PAYLOAD_PLACEHOLDER */</script>`. `serve` sends it untouched. `internal/exporter` replaces the placeholder with the snapshot JSON (tasks, milestones, strategies, glossary, UI-facing config). Go never concatenates HTML, CSS or JS.
- **Static** hides every edit control and shows the `mode-indicator-static` badge. Search, filters, routing and modals still work offline. **Client** shows `mode-indicator-client`, and **live** shows `mode-indicator-live`.
- **Live** reconnects SSE with backoff and applies `task|milestone|strategy|glossary.created|updated|deleted` events to the signal store. The client also listens for `board.refreshed`, but the server never sends it.
- A new UI-facing config field must reach **both** paths: `model.SnapshotConfig` (static) and `model.BoardState` from `/api/board` (live). See *UI translations and project locale*.

## Code layout (`web/src`)

| Path | Contents |
|---|---|
| `App.tsx`, `main.tsx`, `router.ts` | Boot, hash router |
| `components/board` | `KanbanBoard`, `Column`, `TaskCard`, `MilestoneCards` |
| `components/calendar` | Month and ISO-week calendar of `target_at` dates and milestone timeframes |
| `components/strategies`, `components/glossary` | Tiered strategy list, glossary |
| `components/modal` | Task detail/edit/create, column details, About + licenses |
| `components/common` | Header, FilterBar (search, tags, milestone, priority, sort), badges, footer, theme toggle, validation banner |
| `state/` | Signal store and its loaders (`fetchLiveBoard`, `fetchLiveEntities`), bootstrap, SSE client, localStorage persistence, theme |
| `schemas/models.ts` | Valibot schemas; the UI's types are inferred from them |
| `utils/` | i18n, dates (Intl), sorting, search, criteria parsing, Mermaid loader |
| `types/generated.ts` | Output of `cmd/gentypes` mirroring Go JSON. **Currently not imported anywhere**: the schemas are hand-maintained, so keep them in sync with `internal/model` |

Routes are hash-based deep links: `#board`, `#task/<id>`, `#milestone/<id>`, `#calendar[/YYYY-MM | /YYYY-Www][/task/<id> | /milestone/<id>]`, `#strategies`, `#strategy/<id>`, `#glossary`, `#glossary/<id>`.

## Rules

1. **Validate everything that comes from outside** (snapshot, REST, SSE, localStorage) with `v.safeParse`. On failure, show the non-fatal `validation-error-banner` and keep rendering whatever parsed. Never crash the board.
2. **Shared state lives in signals** in `state/store.ts`, and board and entity loading happens there. Today, task mutations (drag-and-drop in `KanbanBoard`; create, edit, notes and checkboxes in the modals) call `fetch` from the component and rely on SSE to refresh the store. Follow that pattern or move it into the store; do not add a third way.
3. **Every testable element has a `data-testid`** (for example `task-card-<id>`, `column-<id>`, `task-detail-modal`, `search-input`, `mode-indicator-live`). Selectors in tests use only these.
4. **No inline handlers or remote assets.** The server sends a CSP whose `script-src`/`style-src` contain the SHA-256 hashes of the inlined script and style. Those hashes are computed once at runtime from the embedded `index.html` (single source of truth; nothing is injected at link time). Anything that needs a new source must be added to `[server.security.csp]` deliberately.
5. **User-visible strings go through `t()`.** Dates go through Intl with `uiLocale()`. See *UI translations and project locale*.
6. The visual design is final. Accessibility fixes are markup-only.

## Build output (`web/dist`, gitignored, embedded)

| File | Purpose |
|---|---|
| `index.html.gz` | The single-file app (gzip level 9), served with `Content-Encoding: gzip` when accepted |
| `hashes.json`, `script.sha256`, `style.sha256` | Build-time record of the inline script and style hashes |
| `mermaid.min.js.gz`, `mermaid.json` | Mermaid runtime and `{version, integrity}` |
| `bundled-packages.json` | Packages in the bundle, read by `cmd/genlicenses` |

`make ui-build` rebuilds only when `web/src`, `web/scripts`, `index.html`, `package.json` or `bun.lock` change. `bun run dev` (port 4321) serves the UI with hot reload and proxies `/api` to a running `jokateko serve` (`BACKEND_URL`).

## Mermaid (lazy, pinned, SRI-verified)

Mermaid is ~95 % of the JavaScript, so it is **not** in the app bundle. `utils/mermaidLoader.ts` loads it when the first diagram renders:

| Mode | Source | Verification |
|---|---|---|
| `serve` | `GET /assets/mermaid-<version>.min.js` (embedded, immutable cache) | SRI `integrity`; hash also added to CSP `script-src` |
| `build --mermaidjs=cdn` (default) | `cdn.jsdelivr.net/npm/mermaid@<version>/dist/mermaid.min.js` | SRI + `crossorigin="anonymous"` |
| `build --mermaidjs=bundled` | Inert `<script type="text/plain">` block inside the export | Embedded bytes |
| `build --mermaidjs=none` | None; diagrams stay as code blocks | n/a |

If loading fails (offline CDN, SRI mismatch), the code block stays visible. Upgrade with `bun update mermaid`: the version, URL and hash all follow from `bun.lock`.

## Testing

- **Unit:** `bun test` next to the code (`*.test.ts`).
- **E2E:** Playwright in `tests/e2e/` against a real `bin/jokateko` and a seeded temporary workspace (`helpers/test-server.ts`), covering live and static modes. Spec files are parsed as plain JS by the Bun-backed runner, so they contain no type annotations or `import type`.
- **Lighthouse:** `tests/lighthouse/` audits every route (desktop preset, one worker because of the shared CDP port). The minimums are `LH_MIN_PERFORMANCE`/`ACCESSIBILITY`/`BEST_PRACTICES` = 90 and `LH_MIN_SEO` = 80. Reports go to `lighthouse-report/`.
