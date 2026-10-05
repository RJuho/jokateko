+++
title = 'Stop raw HTML in Markdown from running script in static exports'
status = 'backlog'
priority = 'critical'
milestone = '261005-public-release-v010'
tags = ['security', 'frontend', 'backend']
summary = 'Task, strategy and glossary bodies are rendered with goldmark `html.WithUnsafe()` and inserted via `dangerouslySetInnerHTML`, and static exports have no CSP, so raw HTML such as `<img onerror=…>` in any entity body runs script when someone opens the export. Neutralize raw HTML and give exports a CSP before the public release.'
created_at = '2026-10-05T12:18:43Z'
changed_at = '2026-10-05T12:18:43Z'
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
- [ ] Raw HTML in Markdown bodies no longer reaches the DOM as live markup. Decide and document one approach: drop `html.WithUnsafe()` (raw HTML is escaped) or sanitize with an allowlist. Check that GFM tables, task lists, code and Mermaid blocks, and the prose typography still render (`markdown-typography.spec.ts`, `mermaid.spec.ts`)
- [ ] Static exports carry a `<meta http-equiv="Content-Security-Policy">` equivalent to the live policy (inline script/style hashes; Mermaid CDN host + SRI for `--mermaidjs=cdn`; the inert bundled block for `bundled`). Verify in all three `--mermaidjs` modes that Mermaid still renders
- [ ] Regression tests: a Go test showing that raw `<script>`/`onerror` HTML in a body is not emitted as live markup, and a Playwright test that opens a static export with a malicious body and asserts no script ran
- [ ] SECURITY.md and the *REST API and SSE* / *Web UI architecture* strategies updated (CSP in exports; how raw HTML is handled)
- [ ] `make test`, `make e2e-test`, `jokateko parse` pass
