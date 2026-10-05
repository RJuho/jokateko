+++
title = 'Stop raw HTML in Markdown from running script in static exports'
status = 'done'
priority = 'critical'
milestone = '261005-public-release-v010'
tags = ['backend', 'frontend', 'security']
summary = 'Raw HTML in entity bodies is rendered as escaped text and dangerous link URLs are dropped. Static exports carry the same Content-Security-Policy as `serve` (via a `<meta http-equiv>`), so a malicious `.jokateko/` body can no longer run script in a shared export.'
created_at = '2026-10-05T12:18:43Z'
changed_at = '2026-10-05T13:43:50Z'
+++

## Context
Found by the pre-publication security review (`261005-pre-publication-audit-history-secrets-li`) on 2026-10-05.

- `internal/parser/markdown.go:25` renders Markdown with goldmark `html.WithUnsafe()`, so raw HTML in a body passes through unescaped into `body_html`.
- The Web UI inserts `body_html` with `dangerouslySetInnerHTML` (`TaskDetailModal.tsx:564`, `StrategiesView.tsx:357`, `GlossaryView.tsx:190`).
- **Live mode** (`serve`) sends a strict CSP, so inline handlers are blocked. HTML injection (fake links, forms, styling) still works.
- **Static export** (`build`) has **no CSP** (no `<meta http-equiv="Content-Security-Policy">`), so injected script runs.

**Reproduced:** a fresh `jokateko init` project, with `<img src="x" onerror="document.title='PWNED'">` appended to a task body, then `jokateko build` and open `#task/<id>` in Chromium → the title becomes `PWNED`. Under `serve`, the same task triggers one CSP violation and no execution.

**Threat:** `.jokateko/` content comes from anyone who can open a PR or whose repository you clone, and static exports are meant to be shared ("email or host it"). Anyone viewing the export runs attacker script in a `file://` or hosting origin.

## Acceptance Criteria
- [x] Raw HTML in Markdown bodies no longer reaches the DOM as live markup. Decide and document one approach: drop `html.WithUnsafe()` (raw HTML is escaped) or sanitize with an allowlist. Check that GFM tables, task lists, code and Mermaid blocks, and the prose typography still render (`markdown-typography.spec.ts`, `mermaid.spec.ts`)
- [x] Static exports carry a `<meta http-equiv="Content-Security-Policy">` equivalent to the live policy (inline script/style hashes; Mermaid CDN host + SRI for `--mermaidjs=cdn`; the inert bundled block for `bundled`). Verify in all three `--mermaidjs` modes that Mermaid still renders
- [x] Regression tests: a Go test showing that raw `<script>`/`onerror` HTML in a body is not emitted as live markup, and a Playwright test that opens a static export with a malicious body and asserts no script ran
- [x] SECURITY.md and the *REST API and SSE* / *Web UI architecture* strategies updated (CSP in exports; how raw HTML is handled)
- [x] `make test`, `make e2e-test`, `jokateko parse` pass

## Notes

### [2026-10-05 13:36 UTC]

**Implementation (2026-10-05).**

- **Raw HTML:** `html.WithUnsafe()` removed from `internal/parser/markdown.go`. A custom goldmark node renderer, `rawHTMLEscaper`, renders `RawHTML` and `HTMLBlock` nodes as escaped text instead of goldmark's `<!-- raw HTML omitted -->`, so text like `Vec<T>` stays visible.
  - Pure HTML comments are dropped.
  - Safe mode also empties `javascript:`, `vbscript:`, `file:` and non-image `data:` link URLs.
  - A raw `<input type="checkbox">` can no longer shift `data-checkbox-index`.
- **CSP:** the policy builder moved from `internal/server` into the new `internal/csp` package (`csp.Build`, `csp.WebAssetHashes`). The server header is unchanged.
  - `exporter.InjectCSP` adds `<meta http-equiv="Content-Security-Policy">` right after the charset declaration, before the inline style and script.
  - Extra `script-src` for each mode: for `cdn`, the exact jsDelivr URL plus the SRI hash. For `bundled`, the SRI hash: the inline script's text is the npm file, so its sha384 equals the SRI hash, which a Go test asserts. For `none`, nothing.
  - The export follows `[server.security.csp].enabled`.
- **Tests:**
  - Go: `TestRenderHTMLNeutralizesRawHTML`, `TestRenderHTMLKeepsSafeMarkdown`, `csp.TestBuild`, `TestInjectCSP`, `TestExport_ContentSecurityPolicy`.
  - Playwright: new `static-export-xss.spec.ts`, confirmed to fail against the old renderer. `static-export-mermaid.spec.ts` now asserts the CSP meta is present and there are no `securitypolicyviolation` events in all three modes.
- **Verified:** `make test`, `make lint`, `make e2e-test` (57/57) and `jokateko parse` pass, and `make install` is done.
- **Docs:** SECURITY.md, CHANGELOG (0.1.0 Security), and the strategies *REST API and SSE*, *Web UI architecture*, *Go packages and data flow* and *Project Structure*.

## Completion Summary
- **Completed At:** 2026-10-05T13:43:50Z

### What Was Done
- **`internal/parser/markdown.go`:**
  - Removed goldmark `html.WithUnsafe()`.
  - Added `rawHTMLEscaper`, a node renderer for `RawHTML`/`HTMLBlock` registered at priority 100. It writes escaped text, and drops pure HTML comments.
  - Safe mode now empties `javascript:`, `vbscript:`, `file:` and non-image `data:` URLs.
- **New package `internal/csp`:**
  - `csp.Build(cfg, extraScriptSrc...)` and `csp.WebAssetHashes()`, moved out of `internal/server`.
  - The server's `buildCSP` now delegates to them; the output is unchanged.
- **`internal/exporter`:**
  - `InjectCSP` inserts the CSP meta right after the charset declaration.
  - `mermaidScriptSrc` adds the Mermaid sources each mode needs: the exact CDN URL plus the SRI hash for `cdn`, the SRI hash for `bundled`, and nothing for `none`.
  - The export follows `[server.security.csp].enabled`.
- **Tests:**
  - Go parser tests: neutralization and safe Markdown.
  - `csp.TestBuild`, `TestInjectCSP` and `TestExport_ContentSecurityPolicy`, which also asserts that the bundled runtime's sha384 equals its SRI integrity.
  - New Playwright `static-export-xss.spec.ts`.
  - `static-export-mermaid.spec.ts` now asserts the CSP meta is present and there are no CSP violations in all three modes.
- **Docs:** SECURITY.md, CHANGELOG, and the strategies *REST API and SSE*, *Web UI architecture*, *Go packages and data flow* and *Project Structure*.

### Why / Rationale
The fix has two independent layers.

- **Escaping at render time** removes the root cause without new dependencies. An allowlist sanitizer would have needed a new library, which needs approval.
- **Escaping instead of goldmark's silent omission** keeps author text such as `Vec<T>` visible.
- **The meta CSP** protects exports even if a future renderer change regresses. Sharing one builder keeps live and static policies from drifting apart.
- **Mermaid sources by exact URL and hash, not by host**, allow only the pinned file. For bundled mode the SRI hash doubles as the inline-script hash, because the inline block holds the same file.
