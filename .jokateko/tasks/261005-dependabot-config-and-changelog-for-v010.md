+++
title = 'Dependabot config and CHANGELOG for v0.1.0'
status = 'done'
priority = 'medium'
milestone = '261005-public-release-v010'
tags = ['ci', 'dependencies', 'release']
summary = 'Dependabot keeps Go modules, Bun packages, GitHub Actions and the pinned devcontainer base image current with weekly grouped PRs. CHANGELOG.md (Keep a Changelog) holds the v0.1.0 highlights, and CONTRIBUTING documents the CHANGELOG-based release flow.'
dependencies = ['261005-harden-ci-and-release-workflows-for-a-pu']
created_at = '2026-10-05T10:58:42Z'
changed_at = '2026-10-05T12:14:48Z'
+++

## Context
After the actions are SHA-pinned (CI hardening task), Dependabot keeps them and the locked dependencies current. Dependabot only proposes PRs. Each dependency PR is still reviewed by a human against the strict-dependency policy, and Dependabot never adds new dependencies.

## Acceptance Criteria
- [x] `.github/dependabot.yml` with ecosystems: `gomod` (`/`), `npm` (`/web`; confirm Dependabot handles `bun.lock`, otherwise document the manual update flow), `github-actions` (`/`), `docker` (`/.devcontainer`, `/`)
- [x] Weekly schedule, grouped minor/patch updates per ecosystem, sensible open-PR limit
- [x] Verified that a Dependabot bump of `mermaid` in `web/bun.lock` still regenerates the pinned CDN version + SRI (`web/dist/mermaid.json`) via the normal build
- [x] `CHANGELOG.md` in Keep a Changelog format with an `[Unreleased]` section and `v0.1.0` summarizing the shipped features (from the done tasks / MVP milestone)
- [x] Release process documented in CONTRIBUTING: update CHANGELOG, tag, and how GitHub's generated release notes relate to it

## Notes

### [2026-10-05 12:12 UTC]

**Dependabot (`.github/dependabot.yml`):** `gomod` /, `bun` /web, `github-actions` /, `docker` /.devcontainer. Weekly; minor and patch grouped into one PR per ecosystem; majors come separately; at most 5 open PRs per ecosystem.
- **Bun:** GitHub supports a native `bun` ecosystem (Bun ≥ 1.1.39, text `bun.lock`) instead of `npm`. It does **version updates only, no security updates** (GitHub docs, checked 2026-10-05), so web security checks stay `bun audit`, which is in the audit task.
- **Docker:** Dependabot cannot bump a floating tag, so `.devcontainer/Dockerfile` now pins `FROM oven/bun:1.4.2-slim@sha256:cb3bbbb0…`. That is the same image `slim` resolves to today (identical digest, checked against Docker Hub), so the devcontainer doesn't change. The root `Dockerfile` is excluded: it builds from our own devcontainer via `ARG`, and the release workflow pins that by digest.
- **Not covered, documented in the file header:** the Go toolchain (`ARG GO_VERSION`, the `go` line in go.mod) is updated by hand.
- `dependabot.yml` parses (Bun YAML), and `actionlint` is clean.

**Mermaid bump verified end to end (throwaway clone):** `bun add mermaid@12.0.0` → `make ui-build` → `web/dist/mermaid.json` = `{version 12.0.0, sha384-xzghz1…}`, which is **byte-identical** to the SRI of `cdn.jsdelivr.net/npm/mermaid@12.0.0/dist/mermaid.min.js`. `bundle.ts` reads the version from the installed package and hashes the file on every build, so a Dependabot lockfile bump needs no manual step.

**CHANGELOG.md:** Keep a Changelog 1.1.0 + SemVer. It has an empty `[Unreleased]` and `[0.1.0] - TBD` summarizing the shipped features (from the done tasks), plus a Security section and compare links. The release dry-run task sets the date.

**Release process:** CONTRIBUTING now says: update the CHANGELOG (move Unreleased → version/date, fix links), tag, push. The release page body links to `CHANGELOG.md` at that tag, followed by GitHub's generated PR list. Pre-releases need no CHANGELOG section.

## Completion Summary
- **Completed At:** 2026-10-05T12:14:48Z

### What Was Done
- Added `.github/dependabot.yml` covering gomod, bun (/web), github-actions and docker (/.devcontainer): weekly, minor and patch grouped, 5 open-PR limit, with the exclusions documented in the file header.
- Pinned the devcontainer base to `oven/bun:1.4.2-slim@sha256` (same image as today's `slim`).
- Verified end to end, in a throwaway clone, that a Mermaid bump regenerates the version and an SRI that matches jsDelivr.
- Added `CHANGELOG.md` with `[Unreleased]` and `[0.1.0] - TBD`.
- CONTRIBUTING release steps now include the CHANGELOG. The release body links to CHANGELOG.md at the tag.

### Why / Rationale
The SHA-pinned actions and locked dependencies need automated, reviewable updates, or they go stale. The native `bun` ecosystem understands `bun.lock`; it has no security updates, so `bun audit` stays the security check. Floating base-image tags can't be updated or audited, so the image is pinned by digest. A curated CHANGELOG gives users the highlights, while GitHub's generated notes keep the PR-level detail.
