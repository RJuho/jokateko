+++
title = 'Pre-publication audit: history secrets, licenses and vulnerabilities'
status = 'backlog'
priority = 'critical'
milestone = '261005-public-release-v010'
tags = ['security', 'release']
summary = 'Before the repo goes public, scan the full git history for secrets and personal data, confirm third-party license compliance, and get govulncheck and bun audit clean.'
dependencies = ['261005-repository-cleanup-and-portable-agent-to']
created_at = '2026-10-05T10:59:00Z'
changed_at = '2026-10-05T10:59:00Z'
+++

## Context
Decision: keep the full git history (103 commits, 3.3 MB). Removed files remain in it: `PLAN.md`, a committed `web/dist/index.html`, vendored third-party skills under `.agents/skills/`, and `docs/config-specification.md`. A quick grep found no secrets, but a proper scan is required before the visibility flip, which cannot be undone in practice.

## Acceptance Criteria
- [ ] Full-history secret scan with gitleaks (or equivalent) run as a one-off tool via Docker or `go run …@version`. No change to `go.mod`/`package.json`; the human approves the tool choice. Findings triaged and recorded in Notes
- [ ] History reviewed for personal or internal data: the author email in commits is accepted. `PLAN.md` and old task notes are checked for internal URLs, hostnames, or anything that should not be public
- [ ] Vendored third-party skills that remain in history (daisyUI, playwright-cli, use-modern-go, valibot) have licenses that allow redistribution; recorded in Notes
- [ ] `go_vulncheck` clean (or findings documented as not reachable)
- [ ] `bun audit` (in `web/`) clean, or findings triaged
- [ ] `internal/version/licenses.json` and `web/src/data/licenses.json` regenerated (`make generate`) and current; the About/licenses page shows them
- [ ] LICENSE holder/year correct; licenses of all direct Go and npm dependencies are MIT-compatible
- [ ] Security review pass on the HTTP server and MCP mutation paths (bind address, Cross-Origin, Host check, path traversal in slugs/IDs, CSP); issues found become separate tasks
