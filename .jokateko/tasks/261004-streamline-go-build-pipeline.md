+++
title = 'Streamline Go build pipeline'
status = 'done'
priority = 'high'
tags = ['build', 'ci', 'devcontainer']
summary = 'The devcontainer now installs the Go build for its own architecture, and the GOTMPDIR storage workaround is gone. The Makefile tracks real file dependencies and has install and clean-cache targets. The release Dockerfile cross-compiles, and a .dockerignore keeps the build context small.'
created_at = '2026-10-04T15:26:18Z'
changed_at = '2026-10-04T16:24:58Z'
+++

## Context
Host Docker storage previously filled up; Makefile routed GOTMPDIR into workspace `.gopath/tmp` as a workaround. Review found the devcontainer installs amd64 Go on an aarch64 host (emulated compiler, amd64 binaries), stale UI embedding, no .dockerignore (~800MB context), redundant CSP hash ldflags.

## Acceptance Criteria
- [x] Devcontainer downloads Go for `TARGETARCH`
- [x] GOTMPDIR/TMPDIR workaround and `.gopath/` removed
- [x] Makefile uses file targets for UI bundle, node_modules, licenses.json
- [x] CSP hash ldflags removed (runtime fallback in server.go)
- [x] `make install` copies binary to ~/.local/bin; `-trimpath`; `clean-cache`; cross-compile loop
- [x] Release Dockerfile cross-compiles from `$BUILDPLATFORM`
- [x] `.dockerignore` added
- [x] AGENTS.md workflow mentions `make install`
- [x] make test / lint / install / cross-compile verified

## Notes

### [2026-10-04 15:43 UTC]

Implemented. Verified in the devcontainer: `make lint` clean, `make test` all 17 packages ok, `make e2e-test` 41 passed, `make cross-compile` produced 5 binaries + checksums. UI staleness tracking works (touching web/src triggers a rebuild, a second run skips it). CSP header hashes match web/dist/*.sha256 through the runtime fallback. Installed a native arm64 binary with `GOARCH=arm64 make install` (ELF e_machine b7).

Extra fix: e2e failed because the image's unpinned Playwright browser (rev 1243) did not match the project's @playwright/test 1.62.1 (rev 1234). `e2e-test` now runs `bunx playwright install chromium` first.

Not verified here (no Docker in the container): the devcontainer rebuild with the TARGETARCH Go, and the release Dockerfile buildx cross-compile. Human: rebuild the devcontainer, then confirm `go env GOARCH` → arm64.

## Completion Summary
- **Completed At:** 2026-10-04T16:24:58Z

### What Was Done
- .devcontainer/Dockerfile: Go download uses ARG GO_VERSION (the user later bumped it to 1.27.1) and ARG TARGETARCH, replacing the hardcoded linux-amd64.
- Makefile: removed the GOTMPDIR/TMPDIR workaround and the .gopath directory. Added file targets for web/node_modules, web/dist/index.html.gz (depends on web/src, web/scripts, index.html, package.json, bun.lock) and licenses.json (generated only if missing). Removed the SCRIPT_HASH/STYLE_HASH ldflags. Builds now use -trimpath. Added `install` (INSTALL_DIR ?= ~/.local/bin) and `clean-cache`. cross-compile is now a loop over PLATFORMS. e2e-test runs `bunx playwright install chromium` before the tests. .PHONY is complete.
- Dockerfile: builder stage uses FROM --platform=$BUILDPLATFORM and cross-compiles with GOOS=$TARGETOS GOARCH=$TARGETARCH.
- .dockerignore added. .gitignore no longer lists .gopath/. AGENTS.md workflow now says `make install`.
- Verified: lint passes, all 17 Go test packages pass, 41 e2e tests pass, cross-compile produces 5 binaries plus checksums, UI staleness tracking works, CSP hashes are correct. ~/.local/bin/jokateko is a native arm64 binary.

### Why / Rationale
The host is Apple Silicon, but the image shipped amd64 Go. Builds ran emulated and produced amd64 binaries, and the arm64 release image would have contained an amd64 binary. Go's temp files are short-lived and never caused the storage problem. The real waste was the build context with no .dockerignore (about 800 MB) and leftover Chromium temp profiles. File-based Make targets stop stale UI from being embedded. Computing the CSP hashes at runtime from the embedded HTML gives one source of truth, so the hashes can't drift from the page. Cross-compiling from BUILDPLATFORM avoids QEMU in buildx. Playwright browser revisions drifted because the image installs the browser unpinned, so e2e-test now installs the revision the project's Playwright version expects.
