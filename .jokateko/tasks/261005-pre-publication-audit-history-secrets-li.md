+++
title = 'Pre-publication audit: history secrets, licenses and vulnerabilities'
status = 'in_review'
priority = 'critical'
milestone = '261005-public-release-v010'
tags = ['release', 'security']
summary = 'Before the repo goes public, scan the full git history for secrets and personal data, confirm third-party license compliance, and get govulncheck and bun audit clean.'
dependencies = ['261005-repository-cleanup-and-portable-agent-to']
created_at = '2026-10-05T10:59:00Z'
changed_at = '2026-10-05T13:27:02Z'
+++

## Context
Decision: keep the full git history (103 commits, 3.3 MB). Removed files remain in it: `PLAN.md`, a committed `web/dist/index.html`, vendored third-party skills under `.agents/skills/`, and `docs/config-specification.md`. A quick grep found no secrets, but a proper scan is required before the visibility flip, which cannot be undone in practice.

## Acceptance Criteria
- [x] Full-history secret scan with gitleaks (or equivalent) run as a one-off tool via Docker or `go run …@version`. No change to `go.mod`/`package.json`; the human approves the tool choice. Findings triaged and recorded in Notes
- [x] History reviewed for personal or internal data: the author email in commits is accepted. `PLAN.md` and old task notes are checked for internal URLs, hostnames, or anything that should not be public
- [x] Vendored third-party skills that remain in history (daisyUI, playwright-cli, use-modern-go, valibot) have licenses that allow redistribution; recorded in Notes
- [x] `go_vulncheck` clean (or findings documented as not reachable)
- [x] `bun audit` (in `web/`) clean, or findings triaged
- [x] `internal/version/licenses.json` and `web/src/data/licenses.json` regenerated (`make generate`) and current; the About/licenses page shows them
- [x] LICENSE holder/year correct; licenses of all direct Go and npm dependencies are MIT-compatible
- [x] Security review pass on the HTTP server and MCP mutation paths (bind address, Cross-Origin, Host check, path traversal in slugs/IDs, CSP); issues found become separate tasks

## Notes

### [2026-10-05 12:18 UTC]

**1. Secret scan (tool approved by the maintainer):** `gitleaks v8.30.1` via one-off `go run` (`go.mod`/`go.sum` unchanged), `git` mode with `--redact` over the full history. **109 commits scanned, no leaks found.** The only refs are `main`/`origin/main`, with no stashes or tags, so that is everything that becomes public. (110 commits exist; gitleaks skips commits with no text diff.)

**2. Personal and internal data (full `git log -p` grep):**
- Emails: the maintainer address (accepted), `noreply@anthropic.com` co-author lines, and authors from third-party license metadata.
- IPs: loopback, `0.0.0.0` and version-like numbers.
- Hosts: public docs, CDNs and `example.*` domains.
- Home paths: `/home/bun` (devcontainer) and `/home/dev` (doc example).
- No internal TLDs, phone numbers or tokens. The `sk-…` hits are task slugs.
- `PLAN.md` (in history until c31b35c) is a purely technical implementation plan.

**3. Third-party skills in history** (89 files, commits 860f2f2…a4efc58): daisyui MIT, open-circle/agent-skills (valibot) MIT, microsoft/playwright-cli Apache-2.0, JetBrains/go-modern-guidelines Apache-2.0 (checked via the GitHub API). All allow redistribution. Minor gap: the historical copies of the two Apache-2.0 skills have no LICENSE file next to them. They are unmodified upstream docs in old commits only and not in the current tree. Acceptable; no history rewrite (per the keep-history decision).

**4. govulncheck** (gopls 0.23, vuln DB 2026-10-01): **no findings.**

**5. bun audit:** 1 low, GHSA-p98j-92pf-mc4p, dompurify 3.4.14 (≥3.4.13 ≤3.4.15) via `mermaid > dompurify`. **Not exploitable:** the app never imports dompurify. The shipped `mermaid.min.js` (prebuilt, embedded and served via CDN) bundles its own **DOMPurify 3.4.12**, which is outside the affected range, and never uses `IN_PLACE`. The node_modules copy is never shipped. Left for Dependabot, since a lock bump changes nothing we ship.

**6. License data and `make generate`:** the license JSONs were current (no diff, 10 packages). **Bug found and fixed:** `make generate` broke the build, because `internal/store/queries.sql.go` had been hand-edited (`interface{}` → `string` for `MIN`/`MAX` columns) and sqlc regeneration reverted that, giving 4 compile errors in `milestones.go`. Fix in the source: `CAST(… AS TEXT)` for `target_start_at`/`target_end_at` in `queries.sql`, so sqlc now generates `string`. `make generate` is idempotent, `go build ./...` OK, store/model/mcp tests pass.

**7. Licenses:** LICENSE is MIT, "© 2026 Juho Räsänen and Jokateko Contributors" (OK). Direct deps: Go fsnotify BSD-3, go-sdk Apache-2.0, go-toml MIT, goldmark MIT, modernc sqlite BSD-3; npm @preact/signals MIT, lucide-preact ISC, mermaid MIT, preact MIT, valibot MIT. All MIT-compatible. **Open question for the maintainer:** `cmd/genlicenses` deliberately excludes tailwindcss, daisyui and @tailwindcss/typography as "build-time CSS". Their generated CSS does ship in the bundle. Tailwind keeps its `/*! … MIT License */` banner, but daisyUI's banner has no license text and typography has none. Listing them on the licenses page would make attribution complete. Not changed, pending a decision.

**8. Security review of server and mutation paths:**
- Bind address, Cross-Origin, Host check and JSON-only writes are as documented in *REST API and SSE*. The known limit is that the Host check only applies to loopback connections (documented in SECURITY.md).
- Path traversal: none. Explicit and generated IDs both pass `ValidateID` (`^[a-z0-9][a-z0-9-]{0,99}$`), existing paths come from the index, and config paths are checked by CFG-005.
- **Issue found → separate task `261005-stop-raw-html-in-markdown-from-running-s` (critical, now blocks the release dry-run):** goldmark `html.WithUnsafe()` + `dangerouslySetInnerHTML`, and static exports have no CSP. Reproduced: `<img onerror>` in a task body runs script in a `jokateko build` export; in `serve` the CSP blocks it.

### [2026-10-05 13:26 UTC]

**Resolved the open license question (maintainer decision 2026-10-05: list the CSS packages).** `cmd/genlicenses` now excludes only `@types/*` and `bun-plugin-tailwind`, whose code never ships. `tailwindcss` 4.3.3, `daisyui` 5.7.47 and `@tailwindcss/typography` 0.5.20 (all MIT, with full license texts) are listed, because their generated CSS is in the bundle. The license data was regenerated: 13 packages (5 Go, 8 npm). The unit-test table, the harvest fixture (`bun-plugin-tailwind` is now the excluded example) and the E2E comment are updated. `go test ./cmd/genlicenses`, `about-licenses.spec.ts`, full `make test` and `make lint` pass.

### [2026-10-05 13:27 UTC]

Correction to the previous note: `make lint` first **failed** (gofmt alignment in `cmd/genlicenses/helpers_test.go` after the fixture edit). Fixed with `gofmt -w`; `make lint` and `go test ./cmd/genlicenses` now pass.
