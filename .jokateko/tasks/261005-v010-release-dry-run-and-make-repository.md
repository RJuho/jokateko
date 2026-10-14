+++
title = 'v0.1.0 release dry-run and make repository public'
status = 'done'
priority = 'critical'
milestone = '261005-public-release-v010'
tags = ['release']
summary = "v0.1.0 is released and the repository is public; stable releases also publish Jokateko's own board at jokateko.dev/jokateko/ next to the installer."
dependencies = ['261005-dependabot-config-and-changelog-for-v010', '261005-harden-ci-and-release-workflows-for-a-pu', '261005-pre-publication-audit-history-secrets-li', '261005-stop-raw-html-in-markdown-from-running-s', '261005-write-contributingmd-and-securitymd', '261005-write-public-readme-and-fold-in-commands', '261006-fix-ci-e2e-pinned-playwright-resolution', '261006-one-line-installer-served-from-jokatekod']
created_at = '2026-10-05T10:59:07Z'
changed_at = '2026-10-14T06:47:03Z'
+++

## Context
Last step of the milestone, performed by the maintainer. Agents can prepare and verify, but tagging, publishing, and changing repository visibility are human actions.

## Acceptance Criteria
- [x] Tag `v0.1.0-rc1`; release workflow green; GitHub pre-release has all 5 binaries + `checksums.txt`
- [x] Binaries smoke-tested on Linux, macOS (arm64), and Windows: `version`, `init`, `serve`, `mcp`, `build`
- [x] Docker image `ghcr.io/rjuho/jokateko:<rc>` runs `serve` with a mounted workspace
- [x] README install, quickstart, and MCP steps followed verbatim on a clean machine
- [x] ghcr packages (`jokateko`, `jokateko-devcontainer`) public and linked to the repo
- [x] Repo settings: description, topics, website; private vulnerability reporting, secret scanning, and push protection enabled; branch protection on `main` requiring CI; fork PR workflows require approval for first-time contributors
- [x] Repository visibility switched to public
- [x] CI runs green for a test pull request from a fork (deferred from `261005-harden-ci-and-release-workflows-for-a-pu`; needs the public devcontainer image)
- [x] Tag `v0.1.0`; CHANGELOG release date set; `latest` Docker tag published
- [x] Milestone `Public Release v0.1.0` closed

## Notes

### [2026-10-06 17:20 UTC]

Checked anonymously, 2026-10-06:
- Repo visibility is `public`.
- ghcr `rjuho/jokateko` (tags 0.1.0-rc1, 0.1.0-rc2) and `rjuho/jokateko-devcontainer` (latest, v0.1.0, 0.1.0-rc1, 0.1.0-rc2): anonymous pull tokens are issued and both package pages return 200, so they're public and linked to the repo.
- Installer: `jokateko.dev/install.sh` installs and verifies the release binary (see task 261006-one-line-installer-served-from-jokatekod).

Not yet visible from outside: the repo description, homepage and topics are empty in the public API and on the repo page. `branches/main` reports `protected: false`, and there are no rulesets, so `main` has no branch protection and no required CI. Private vulnerability reporting, secret scanning and push protection can't be checked anonymously.

Oddity: the devcontainer image has a `v0.1.0` tag, but the release workflow's semver pattern produces `0.1.0`. It probably comes from an earlier manual run. Check it before the real release.

### [2026-10-14 06:47 UTC]

Board export on GitHub Pages (2026-10-14):
- `pages.yml` takes a `version` input (passed by `release.yml`; falls back to the selected ref on manual runs). After checkout it installs that release with `site/install.sh` (checksum-verified) into `$RUNNER_TEMP/bin`, runs `jokateko build -dir . -out site/jokateko -mermaidjs=cdn` and checks that the snapshot is embedded. The board ships in the same Pages artifact as `install.sh`, because each deployment replaces the whole site.
- Bug found while verifying: `parseSnapshotContent` (`web/src/state/bootstrap.ts`) threw away any payload containing `JOKATEKO_PAYLOAD_PLACEHOLDER`. This repo's own strategy and glossary mention it, so the v0.1.0 export fell back to live mode and showed an empty board over HTTP. Now only a `/* … */` comment holding the token counts as unfilled; there's a regression test in `bootstrap.test.ts`.
- The v0.1.0 binary has the bug, so the published board renders correctly only from the next stable release. Don't redeploy Pages manually on `v0.1.0`.
- Verified locally: the v0.1.0 installer path and an export with the fixed binary served over HTTP open in static mode with tasks and no console errors.

## Completion Summary
- **Completed At:** 2026-10-14T06:47:03Z

### What Was Done
Release candidates and v0.1.0 tagged and verified (binaries, Docker images, README steps, repo settings, visibility). The Pages workflow now installs the release binary via site/install.sh and exports the repo board with `jokateko build` into site/jokateko/ in the same deployment as install.sh. Fixed the static-mode bootstrap rejecting snapshots whose text mentions the payload placeholder token.

### Why / Rationale
Tagging, publishing and visibility stay human actions. The board export reuses the published installer and the released binary, so the live demo matches a reviewed release and also checks the installer end to end; it must share the Pages artifact because each deployment replaces the whole site.
