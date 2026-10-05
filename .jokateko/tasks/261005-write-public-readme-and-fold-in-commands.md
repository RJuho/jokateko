+++
title = 'Write public README and fold in COMMANDS.md'
status = 'done'
priority = 'high'
milestone = '261005-public-release-v010'
tags = ['docs', 'release']
summary = 'The README is a complete public landing page: cover image, features, install, quickstart, agent setup, CLI reference, configuration, security, dogfooding, contributing and license. COMMANDS.md is folded in and deleted.'
dependencies = ['261005-migrate-docs-into-strategies-and-glossar']
created_at = '2026-10-05T10:58:27Z'
changed_at = '2026-10-05T14:30:49Z'
+++

## Context
`README.md` is 14 lines and links to `docs/`, which is being removed. The decision is to fold `COMMANDS.md` into the README. User-facing CLI goes here; developer `make` targets go to CONTRIBUTING.md.

Note: `go install github.com/RJuho/jokateko/cmd/jokateko@latest` does **not** work, because the embedded `web/dist` is generated and gitignored. Do not advertise it. Offer release binaries, Docker, or build from source.

## Acceptance Criteria
- [x] Intro: what Jokateko is, who it is for (humans + AI agents), Tasks-as-Code / Spec-First in 2–3 sentences
- [x] Feature list (live board + calendar, strategies/glossary, MCP server + stdio proxy, static single-file export, `parse` linting, zero-CGO single binary)
- [x] Screenshot of the board, stored in the repo at a small size (e.g. `.github/assets/`)
- [x] Install: release binaries with checksum verification, Docker `ghcr.io/rjuho/jokateko` (volume mount example), build from source (`make build`, prerequisites Go 1.27+ and Bun)
- [x] Quickstart: `jokateko init`, `jokateko serve`, open the UI, create a first task
- [x] Connect an AI agent: `.mcp.json` example for `jokateko mcp`, explaining proxy vs standalone mode
- [x] Configuration: `.jokateko/config.toml` overlay model, link to `internal/config/default.toml`
- [x] CLI reference from COMMANDS.md: serve, mcp, parse/lint, build (`--mermaidjs`), init (`--replace`), version, help. The "Suggested Additional Commands" wording becomes real commands
- [x] Security note: binds to loopback by default and has no authentication; do not expose it publicly (link SECURITY.md)
- [x] "Dogfooding" section: this repo's own board lives in `.jokateko/`; how to browse it (`jokateko serve` or `jokateko build`)
- [x] Contributing + license sections linking CONTRIBUTING.md, SECURITY.md, LICENSE
- [x] `COMMANDS.md` deleted; no remaining references (`git grep COMMANDS.md`), including in `AGENTS.md`
- [x] Every command in the README is run once and verified to work as written

## Notes

### [2026-10-05 11:43 UTC]

README rewritten to be minimal (~100 lines): intro, features, install, quickstart, agent setup, one CLI table, configuration, security, dogfooding, contributing and license. COMMANDS.md is deleted. AGENTS.md and the `project-structure` strategy now point to README.md. The remaining `COMMANDS.md` mentions are only in done-task history, the milestone's decision log and `web/src/fixtures` sample data, all left alone on purpose.

**Two criteria are intentionally open**
- **Screenshot (#3):** the maintainer will add it later (decision 2026-10-05).
- **Every command verified (#13):** all local commands were run in a fresh `git init` project and work as written: `init`, `parse`, `lint`, `serve` (health and UI 200 on 127.0.0.1:8080), `JOKATEKO_HOST=0.0.0.0 serve` (binds `[::]:8080`), `build -out`, `mcp` (standalone initialize response), `version`, `about`, `licenses`, `init -h`. `make cross-compile` + `sha256sum --ignore-missing -c checksums.txt` → OK. **Not runnable yet:** the release `curl` downloads (no release exists) and `docker run` (no Docker in the devcontainer, image not published). Task `261005-v010-release-dry-run-and-make-repository` covers both.

**Expected broken links until task 4:** `CONTRIBUTING.md` and `SECURITY.md` are linked but created by `261005-write-contributingmd-and-securitymd`, which this task blocks.

**Developer make-targets** from the old COMMANDS.md (including the `make cover` description) are left for CONTRIBUTING.md (task 4); git history has the text.

### [2026-10-05 14:20 UTC]

**Automated screenshots and cover (criterion #3, groundwork; maintainer request 2026-10-05).** `make screenshots` builds the binary and runs `playwright.screenshots.config.ts`. Output goes to the gitignored root `screenshots/` folder (`/screenshots/` is anchored so `tests/screenshots/` stays tracked):
- `views/*.png`: 9 captures at 1440×900 (board, task detail, calendar month, milestone board, strategy, glossary). 4 are in dark mode via `emulateMedia({ colorScheme })`, because the theme follows the OS preference when none is saved.
- `cover.html`: built from `tests/screenshots/cover.html`. It has a logo placeholder on the left and a tilted 3-column screenshot wall on the right that fades out under the logo. It can be opened and tweaked by hand.
- `cover.png` (2560×1280, 2x, for the README) and `cover-social.jpg` (1280×640, about 95 KB, under the 1 MB GitHub social-preview limit).

**Data:** `tests/screenshots/demo-workspace.ts` seeds an initialized temp workspace from `web/src/fixtures/sampleData.ts` (216 tasks, 8 milestones, strategies, glossary) through a new `seed` hook in `tests/e2e/helpers/test-server.ts`, then serves it with the real binary. Sample IDs like `task-ui/ux-199` are slugified to match `ValidateID`. Open tasks get target dates in 2026-10, so the calendar has content. The repo's own board was not used because it is 53 done / 2 backlog with an empty calendar.

**Gotcha:** in this Bun + Playwright setup, `*.spec.ts` files are parsed as plain JS (any type annotation fails with "N errors building"). Imported helpers do get TypeScript. That is why the existing specs carry no types.

**Still open:** the real logo (the placeholder `.logo` element in `cover.html`) and copying the final cover into `.github/assets/` + README.

### [2026-10-05 14:30 UTC]

**Cover added (#3 done).** `.github/assets/cover.jpg` (2560×1280 JPEG, about 200 KB; the spec now also renders this next to the `cover.png` master) is shown at the top of the README. `.github/assets/social-preview.jpg` (1280×640, about 95 KB) is committed for the maintainer to upload under repo Settings → Social preview, since GitHub has no file-based setting for it. Refreshing them is manual (copy from `screenshots/`, documented in CONTRIBUTING), because the calendar's "today" marker makes each run differ.

**#13 ticked with a handoff (maintainer decision 2026-10-05):** all local commands are verified (see the 11:43 note). The release `curl` downloads and `docker run` are **not** verified yet. They need a published release and are covered by `261005-v010-release-dry-run-and-make-repository` ("README install, quickstart, and MCP steps followed verbatim on a clean machine" and the Docker image criterion).

## Completion Summary
- **Completed At:** 2026-10-05T14:30:49Z

### What Was Done
The README was rewritten earlier (about 100 lines, see notes) and COMMANDS.md was deleted. This step added the cover: `.github/assets/cover.jpg` (2560×1280 JPEG, about 200 KB) at the top of the README, and `.github/assets/social-preview.jpg` (1280×640, about 95 KB) to upload as the GitHub social preview. Both come from the new `make screenshots` target. It seeds a demo workspace from the web UI sample data, serves it with the real binary, captures 9 views (4 in dark mode) and renders `tests/screenshots/cover.html`: a logo placeholder on the left and a tilted screenshot wall on the right. CONTRIBUTING.md documents the target and how to refresh the committed assets. Criterion #13 was ticked with a handoff: every local command is verified, while the release `curl` downloads and `docker run` will be verified in `261005-v010-release-dry-run-and-make-repository`.

### Why / Rationale
The cover is generated from a realistic sample workspace instead of hand-made screenshots, so it can be refreshed in one command whenever the UI changes. Copying it into `.github/assets/` is a manual step on purpose, because the calendar's "today" marker would otherwise change tracked files on every run. JPEG keeps the committed asset about 7× smaller than the PNG master. The release-dependent commands can't be checked before a release exists, so the release dry-run, which already verifies the README end to end, owns them.
