+++
title = 'Write public README and fold in COMMANDS.md'
status = 'in_review'
priority = 'high'
milestone = '261005-public-release-v010'
tags = ['docs', 'release']
summary = 'Replace the 14-line README with a complete public landing page (what/why, install, quickstart, MCP setup, CLI reference, security note) and delete COMMANDS.md.'
dependencies = ['261005-migrate-docs-into-strategies-and-glossar']
created_at = '2026-10-05T10:58:27Z'
changed_at = '2026-10-05T11:43:03Z'
+++

## Context
`README.md` is 14 lines and links to `docs/`, which is being removed. The decision is to fold `COMMANDS.md` into the README. User-facing CLI goes here; developer `make` targets go to CONTRIBUTING.md.

Note: `go install github.com/RJuho/jokateko/cmd/jokateko@latest` does **not** work, because the embedded `web/dist` is generated and gitignored. Do not advertise it. Offer release binaries, Docker, or build from source.

## Acceptance Criteria
- [x] Intro: what Jokateko is, who it is for (humans + AI agents), Tasks-as-Code / Spec-First in 2–3 sentences
- [x] Feature list (live board + calendar, strategies/glossary, MCP server + stdio proxy, static single-file export, `parse` linting, zero-CGO single binary)
- [ ] Screenshot of the board, stored in the repo at a small size (e.g. `.github/assets/`)
- [x] Install: release binaries with checksum verification, Docker `ghcr.io/rjuho/jokateko` (volume mount example), build from source (`make build`, prerequisites Go 1.27+ and Bun)
- [x] Quickstart: `jokateko init`, `jokateko serve`, open the UI, create a first task
- [x] Connect an AI agent: `.mcp.json` example for `jokateko mcp`, explaining proxy vs standalone mode
- [x] Configuration: `.jokateko/config.toml` overlay model, link to `internal/config/default.toml`
- [x] CLI reference from COMMANDS.md: serve, mcp, parse/lint, build (`--mermaidjs`), init (`--replace`), version, help. The "Suggested Additional Commands" wording becomes real commands
- [x] Security note: binds to loopback by default and has no authentication; do not expose it publicly (link SECURITY.md)
- [x] "Dogfooding" section: this repo's own board lives in `.jokateko/`; how to browse it (`jokateko serve` or `jokateko build`)
- [x] Contributing + license sections linking CONTRIBUTING.md, SECURITY.md, LICENSE
- [x] `COMMANDS.md` deleted; no remaining references (`git grep COMMANDS.md`), including in `AGENTS.md`
- [ ] Every command in the README is run once and verified to work as written

## Notes

### [2026-10-05 11:43 UTC]

README rewritten to be minimal (~100 lines): intro, features, install, quickstart, agent setup, one CLI table, configuration, security, dogfooding, contributing and license. COMMANDS.md is deleted. AGENTS.md and the `project-structure` strategy now point to README.md. The remaining `COMMANDS.md` mentions are only in done-task history, the milestone's decision log and `web/src/fixtures` sample data, all left alone on purpose.

**Two criteria are intentionally open**
- **Screenshot (#3):** the maintainer will add it later (decision 2026-10-05).
- **Every command verified (#13):** all local commands were run in a fresh `git init` project and work as written: `init`, `parse`, `lint`, `serve` (health and UI 200 on 127.0.0.1:8080), `JOKATEKO_HOST=0.0.0.0 serve` (binds `[::]:8080`), `build -out`, `mcp` (standalone initialize response), `version`, `about`, `licenses`, `init -h`. `make cross-compile` + `sha256sum --ignore-missing -c checksums.txt` → OK. **Not runnable yet:** the release `curl` downloads (no release exists) and `docker run` (no Docker in the devcontainer, image not published). Task `261005-v010-release-dry-run-and-make-repository` covers both.

**Expected broken links until task 4:** `CONTRIBUTING.md` and `SECURITY.md` are linked but created by `261005-write-contributingmd-and-securitymd`, which this task blocks.

**Developer make-targets** from the old COMMANDS.md (including the `make cover` description) are left for CONTRIBUTING.md (task 4); git history has the text.
