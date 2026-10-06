+++
title = 'One-line installer served from jokateko.dev'
status = 'done'
priority = 'high'
milestone = '261005-public-release-v010'
tags = ['ci', 'release']
summary = '`curl -fsSL https://jokateko.dev/install.sh | sh` installs the release binary on Linux and macOS after checking its SHA-256, without sudo. The script is served by GitHub Pages, which the release workflow publishes for stable tags.'
created_at = '2026-10-06T15:37:15Z'
changed_at = '2026-10-06T17:15:10Z'
+++

## Context
Users should be able to install with `curl -fsSL https://jokateko.dev/install.sh | sh`, as with Claude Code and Antigravity. Binaries stay on GitHub Releases. GitHub Pages can't send HTTP redirects, so Pages serves the script itself. Pages deploys only from the release workflow on a tag, so the live script always matches a reviewed release, not whatever is on `main`.

## Design
- `install.sh`: POSIX `sh`, `set -eu`, with everything inside `main()` called on the last line so a truncated download never runs. Detects linux/darwin and amd64/arm64.
- Downloads `jokateko-<os>-<arch>` and `checksums.txt` from `releases/latest/download/` (or `releases/download/$JOKATEKO_VERSION/`), with no GitHub API calls and therefore no rate limit. Verifies SHA-256 with `sha256sum` or `shasum -a 256` and refuses to install on a mismatch or missing entry.
- Installs to `$JOKATEKO_INSTALL_DIR` (default `~/.local/bin`) without sudo. If that folder isn't on PATH, prints a hint rather than editing shell config files.
- Release workflow: runs the script against the freshly built binaries before publishing, attaches `install.sh` to the release, and deploys a GitHub Pages site (`install.sh`, `CNAME`) for stable tags only.
- Windows keeps the manual download in the README.

## Acceptance Criteria
- [x] `install.sh` implemented as designed; passes under `dash`
- [x] Checksum mismatch and unsupported OS/arch fail with a clear message and install nothing
- [x] `make test-install` runs the installer end to end against locally cross-compiled binaries
- [x] Release workflow verifies the installer, attaches `install.sh`, and deploys Pages on stable tags
- [x] README install section leads with the one-liner and shows an inspect-first alternative
- [x] Human: `jokateko.dev` DNS (apex A/AAAA + `www` CNAME to GitHub Pages), domain verified in GitHub account settings, Pages source set to "GitHub Actions", custom domain set, HTTPS enforced
- [x] Human: immutable releases enabled in repository settings

## Notes

### [2026-10-06 15:39 UTC]

Implemented:
- `site/install.sh` and `site/CNAME` (`jokateko.dev`).
- `tests/install/run.sh` and `make test-install`, with 6 checks under dash: install, reinstall, checksum mismatch, missing checksum entry, unsupported OS (fake `uname`), malformed version.
- release.yml: a "Verify Installer" step before publishing; `site/install.sh` as a release asset; a new `pages` job (stable tags only, after `binaries`) using upload-pages-artifact v5.0.0 and deploy-pages v5.0.1, pinned by SHA.
- README install section, CHANGELOG entry, CONTRIBUTING target table.

Decision: prereleases are now created as **drafts** (`draft: ${{ env.PRERELEASE == 'true' }}`). The pinned softprops v3.0.3 publishes prereleases before uploading assets, so with immutable releases on the upload would fail. Stable releases already upload before publishing. The maintainer publishes rc drafts by hand.

Not covered: CI on `main` doesn't run test-install (it needs a cross-compile); the release workflow checks it before publishing. The script can't run against real GitHub URLs until the repo is public.

Human setup for jokateko.dev:
1. GitHub → Settings (account) → Pages → add and verify `jokateko.dev` (TXT record). This prevents domain takeover.
2. DNS apex: A records 185.199.108.153, 185.199.109.153, 185.199.110.153 and 185.199.111.153; AAAA 2606:50c0:8000::153, 2606:50c0:8001::153, 2606:50c0:8002::153 and 2606:50c0:8003::153. `www`: CNAME to `rjuho.github.io`.
3. Repo → Settings → Pages: Source "GitHub Actions", custom domain `jokateko.dev`, Enforce HTTPS (.dev is HSTS-preloaded, so HTTPS is required).
4. Repo → Settings → General → Releases: enable immutable releases.

### [2026-10-06 16:00 UTC]

Domain checked 2026-10-06:
- Apex A (4) and AAAA (4) records point to GitHub Pages.
- Let's Encrypt certificate `CN=jokateko.dev`, valid until 2027-01-04.
- `www` CNAME points to `rjuho.github.io`.
- `https://jokateko.dev/install.sh` returns 404 from GitHub.com. This is expected: nothing is deployed until the first stable tag runs the `pages` job.

Open: the certificate doesn't cover `www.jokateko.dev`, so HTTPS there fails with a name mismatch. The install URL uses the apex, so the installer isn't affected. To fix: remove and re-add the custom domain in Pages settings so GitHub requests a certificate for both names.

### [2026-10-06 16:09 UTC]

Changed on the maintainer's request: Pages has its own workflow, `.github/workflows/pages.yml`, and is no longer part of the release workflow.
- Trigger: push to `main` touching `site/**`, `tests/install/**` or the workflow file, or `workflow_dispatch`.
- The `test` job runs `make test-install` in the devcontainer image (cross-compile plus the 6 installer checks). `deploy` runs only if it passes and uploads `site/` with upload-pages-artifact and deploy-pages.
- Concurrency group `pages` without cancel-in-progress, so a running deployment finishes.
- `site/` is the Pages root: `site/install.sh` is served at `https://jokateko.dev/install.sh`. When the Astro docs site arrives, move `install.sh` and `CNAME` to Astro's `public/` folder so the URL stays the same.
- release.yml keeps "Verify Installer" and the `install.sh` release asset; its `pages` job is removed.

Trade-off accepted: the live script now follows `main`, not the latest release. It always downloads `releases/latest`, so it works as long as `main` doesn't rely on release assets that haven't been published yet.

### [2026-10-06 16:11 UTC]

Reverted to release-tied publishing, on the maintainer's call (it also suits future docs releases):
- `pages.yml` is now a reusable workflow (`workflow_call` and `workflow_dispatch`) holding only the deploy job. It's the home for the future Astro build step.
- release.yml's `pages` job calls it after `binaries`, for stable tags only, with `contents: read`, `pages: write` and `id-token: write`.
- The installer is still tested by "Verify Installer" in the release `binaries` job before anything is published.
- Manual redeploy: Actions → Pages → Run workflow, with a tag selected under "Use workflow from".

Human setup still needed: the `github-pages` environment only allows deploys from `main` by default, which rejects tag-triggered deploys. In Repo → Settings → Environments → github-pages → Deployment branches and tags, add the tag rule `v*`. Keep `main` too if you want manual runs from it.

## Completion Summary
- **Completed At:** 2026-10-06T17:15:10Z

### What Was Done
- `site/install.sh` (POSIX sh, run through `main()`): detects OS/arch (native arm64 under Rosetta), downloads from GitHub Releases with no API calls, verifies against `checksums.txt`, installs atomically to `~/.local/bin` (or `JOKATEKO_INSTALL_DIR`), and prints a PATH hint. `JOKATEKO_VERSION` pins a tag and `JOKATEKO_DOWNLOAD_URL` sets a mirror. `site/CNAME` is `jokateko.dev`.
- `tests/install/run.sh` and `make test-install`: 6 checks under dash against cross-compiled assets.
- release.yml: "Verify Installer" runs before publishing; `install.sh` is attached as a release asset; prereleases start as drafts (immutable releases); a `pages` job for stable tags calls the reusable `pages.yml` (`workflow_call` and `workflow_dispatch`), which deploys `site/`.
- README install section, CHANGELOG and CONTRIBUTING updated.
- Human setup done: jokateko.dev DNS and TLS, Pages source set to Actions, immutable releases on, `v*` tag rule on the github-pages environment.
- Verified live: the jokateko.dev script installed `v0.1.0-rc1` with `JOKATEKO_VERSION` pinned, and the plain one-liner worked after the rc was marked as the latest release.

### Why / Rationale
Binaries stay on GitHub Releases, which are served from GitHub's download servers with no bandwidth limit for public repos. The `releases/latest/download` link isn't subject to the GitHub API's hourly limit for anonymous requests. GitHub Pages can't send HTTP redirects, so the custom domain serves the script itself. Publishing Pages only from the release workflow keeps the live script and future docs in step with a released version, never with unreleased `main`. A separate reusable `pages.yml` leaves room for the Astro docs build. Checksum verification, the `main()` wrapper against truncated downloads, a no-sudo install and no shell-profile edits follow the project's security-first approach to distributed assets.
