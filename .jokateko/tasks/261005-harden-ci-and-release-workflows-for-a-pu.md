+++
title = 'Harden CI and release workflows for a public repo'
status = 'done'
priority = 'high'
milestone = '261005-public-release-v010'
tags = ['ci', 'release', 'security']
summary = 'CI, devcontainer and release workflows are hardened for a public repo: actions SHA-pinned, least-privilege per-job permissions, correct stable/pre-release and `latest` handling, release image built from the exact devcontainer digest, and correct E2E change detection. The fork-PR run is deferred to the release dry-run.'
created_at = '2026-10-05T10:58:16Z'
changed_at = '2026-10-05T12:10:52Z'
+++

## Context
`.github/workflows/{ci,devcontainer,release}.yml` were written for a private repo.
- Actions are pinned by tag (`actions/checkout@v7`, `softprops/action-gh-release@v3`, …), not by commit SHA. That conflicts with the project's pinned-asset security priority.
- CI and release run inside `ghcr.io/rjuho/jokateko-devcontainer:latest`. Fork PRs can pull that image only if the package is public.
- Release `docker` job: `type=raw,value=latest,enable=${{ github.ref_name == 'main' }}` is never true on a `v*.*.*` tag push, so `ghcr.io/rjuho/jokateko:latest` is never published. Meanwhile the devcontainer job pushes `latest` on every release.
- Workflow-level `permissions: contents: write, packages: write` applies to every release job.

## Acceptance Criteria
- [x] Every third-party action pinned to a full commit SHA with a `# vX.Y.Z` comment (Dependabot keeps them current; see the Dependabot task)
- [x] `permissions` scoped per job to the minimum each job needs; workflow default is `contents: read`
- [x] Docker `latest` tag published on stable `vX.Y.Z` tags, not on pre-releases (`-rc`)
- [x] Pre-release tags (`v*-rc*`) produce a GitHub pre-release
- [x] CI runs green for a PR from a fork (public devcontainer image, or documented fallback)
- [x] The CI web-change detection (`git diff HEAD~1`) is correct for multi-commit pushes, or replaced
- [x] `checksums.txt` produced and documented; provenance attestation (`actions/attest-build-provenance`) evaluated. Adding it needs human approval
- [x] Human checklist in Notes: make the `jokateko` and `jokateko-devcontainer` ghcr packages public and link them to the repo

## Notes

### [2026-10-05 12:07 UTC]

**Done**
- **SHA pinning:** every action in `ci.yml`, `devcontainer.yml` and `release.yml` is pinned to the commit of its latest release in the same major version, resolved with `git ls-remote` on 2026-10-05: checkout v7.0.1, login-action v4.6.0, devcontainers/ci v0.3.1900000450, action-gh-release v3.0.3, setup-qemu v4.4.0, setup-buildx v4.4.1, metadata-action v6.2.0, build-push-action v7.4.0. Each has a `# vX.Y.Z` comment for Dependabot.
- **Permissions:** the workflow default is `contents: read` everywhere. Write scopes are per job: release `binaries` gets `contents: write` + `packages: read`; the `devcontainer` and `docker` jobs and devcontainer CI get `packages: write`. CI stays read-only (`contents`/`packages: read`).
- **Pre-releases and `latest`:** `VERSION` comes from the tag or the dispatch input. A `-` in it (`v0.1.0-rc1`) marks a pre-release: the GitHub release gets `prerelease: true` / `make_latest: false`, and neither image gets `latest` (`flavor: latest=false` plus an explicit `latest` only for stable versions). Images are tagged with the semver without `v` (`0.1.0`, `0.1.0-rc1`). This fixes the old `enable=ref_name == 'main'` rule, which never published `latest` on tag pushes.
- **Release builds from the exact devcontainer:** the `docker` job builds from `jokateko-devcontainer@<digest>`, the image the same run just pushed, instead of the moving `:latest`. A pre-release no longer overwrites the devcontainer `latest` either.
- **CI change detection:** compares against the PR base SHA or `github.event.before`, so every commit of a multi-commit push counts. It runs E2E when the range is unknown (new branch, force push, manual run). `Makefile` and `.github/workflows/` changes now also trigger E2E. Simulated on real commits: the three docs-only commits from today → skip; the cleanup and translations commits → run; zero/unknown base and dispatch → run.
- **Validation:** `actionlint v1.7.12` (one-off `go run`, `go.mod` unchanged) reports no issues on all three workflows.
- **Checksums:** `make cross-compile` writes `checksums.txt` and the release uploads it. The README documents `sha256sum --ignore-missing -c`, and CONTRIBUTING describes releases and pre-releases.

**Provenance attestation (evaluated, not added; needs maintainer approval):** `actions/attest-build-provenance` (latest v4.2.2) would sign SLSA build provenance for the 5 binaries and both images, so users can run `gh attestation verify jokateko-linux-amd64 -R RJuho/jokateko`. Cost: one step per job and the extra permissions `id-token: write` + `attestations: write`. It is free for public repos. Recommended once the repo is public.

**Not verifiable yet:** a CI run for a PR from a fork. The workflow is fork-safe (read-only token, no secrets besides `GITHUB_TOKEN`), but forks can only pull `jokateko-devcontainer` once that package is public.

**Human checklist (GitHub settings)**
1. ghcr.io → Packages → `jokateko-devcontainer` and `jokateko` → Package settings → *Change visibility* → **Public**, and *Connect repository* → `RJuho/jokateko` (the `jokateko` package appears after the first release).
2. Repo Settings → Actions → General → *Fork pull request workflows*: require approval for first-time contributors.
3. Then open a test PR from a fork to tick the fork criterion. This is also listed in the release dry-run task.

### [2026-10-05 12:10 UTC]

Criterion 5 (CI green for a fork PR) is ticked as **deferred, not verified**, at the maintainer's request to close this task. The workflow is fork-safe and the human steps are documented above. The actual fork-PR run is now a criterion of `261005-v010-release-dry-run-and-make-repository`, because it needs the public devcontainer image. Provenance attestation was not added; that awaits a separate maintainer decision.

## Completion Summary
- **Completed At:** 2026-10-05T12:10:52Z

### What Was Done
- Pinned all 8 actions to release commit SHAs with `# vX.Y.Z` comments.
- Made each workflow default to `contents: read`, with write scopes only on the jobs that publish.
- `release.yml`:
  - Pre-release detection from a `-` in the version, used for both the GitHub release and the `latest` image tag.
  - Semver image tags.
  - Docker build pinned to the devcontainer digest pushed in the same run.
- `ci.yml`: change detection against the PR base or `event.before`, falling back to running E2E. `Makefile` and workflow changes now trigger E2E.
- `actionlint` v1.7.12 is clean. Change detection was simulated on real commits.
- CONTRIBUTING documents pre-release tags.

### Why / Rationale
A public repo runs untrusted PR code and publishes artifacts people install, so action versions must be immutable and tokens minimal. The old `latest` rule never matched tag pushes, and the release image was built from a moving base. Running E2E when unsure avoids silently skipping it. The fork-PR verification needs public packages, so it moved to the release dry-run task. Provenance attestation was evaluated and recommended, but not added without approval.
