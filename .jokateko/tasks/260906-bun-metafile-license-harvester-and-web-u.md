+++
title = 'Bun Metafile License Harvester and Web UI License Link Fixes'
status = 'done'
priority = 'high'
tags = ['bugfix', 'build', 'frontend', 'ui']
summary = 'Harvest Bun metafile bundled packages and fix license/project URLs in Web UI'
dependencies = ['260904-open-source-license-generator-cli-comman']
created_at = '2026-09-06T14:32:38Z'
changed_at = '2026-09-06T15:02:50Z'
+++

# Bun Metafile License Harvester and Web UI License & Project URL Link Fixes

Enhance open-source license harvesting by extracting only packages bundled into the client bundle via Bun metafile, and fix repository and project URL links in the Web UI.

## Background & Rationale
Jokateko is distributed as an auditable, single-executable application. Currently, `cmd/genlicenses` harvests npm dependencies by traversing `dependencies` in `web/package.json` and walking `node_modules` recursively. This results in over 80+ packages listed in the licenses view, including:
- TypeScript type declaration stubs (`@types/d3`, `@types/geojson`, etc.) which contain zero executable code and are never shipped.
- Build-time CSS utilities (`tailwindcss`, `daisyui`, `bun-plugin-tailwind`, `@tailwindcss/typography`) that only generate CSS and are not runtime JavaScript libraries.
- CLI sub-dependencies like `commander` and `tinyexec`.

Furthermore, in `web/src/components/modal/AboutModal.tsx`, package repository links from npm packages frequently contain shorthand strings (e.g. `preactjs/preact`, `github.com/markedjs/marked`, `github:fabiospampinato/khroma`). Without an explicit `https://` protocol prefix, browser `<a href>` elements treat these as relative links on `http://localhost:8080/`, resulting in 404 errors when clicked.

## Requirements & Scope
1. **Bun Metafile / Bundled Packages Extraction**:
   - Update `web/scripts/bundle.ts` to capture Bun bundler inputs (via `Bun.build({ ... metafile: true })`) and export the list of bundled package names.
   - Update `cmd/genlicenses/main.go`'s `harvestNpmLicenses` to only collect licenses for packages whose code is actually bundled into the single-file Web UI.
   - Exclude `@types/*` declaration packages and build-time CSS utilities from the harvested license list.

2. **URL Normalization & Protocol Prefixing**:
   - Update `extractNpmURL` in `cmd/genlicenses/main.go` to normalize GitHub shorthands (`owner/repo`, `github:owner/repo`, `github.com/...`) and ensure all emitted repository URLs begin with `https://`.
   - Update `AboutModal.tsx` to ensure any displayed external link (for project or dependencies) is guaranteed to use `https://` and open safely in a new tab (`target="_blank" rel="noopener noreferrer"`).
   - Ensure clicking the "Licenses" trigger in the footer opens `AboutModal` directly on the "Open Source Licenses" tab rather than defaulting to the "About" tab.

3. **Validation & Testing**:
   - Re-run license generation (`go run ./cmd/genlicenses` / `make generate`).
   - Verify `licenses.json` contains only actually bundled packages and all URLs are valid `https://` links.
   - Update or add unit tests for `genlicenses` URL parsing.
   - Run Playwright E2E tests to verify clicking repository links in the modal does not attempt relative navigation.

## Acceptance Criteria
- [x] Export bundled node module package list from `web/scripts/bundle.ts` using Bun bundler metafile
- [x] Update `cmd/genlicenses/main.go` to harvest only packages present in the bundled frontend metadata
- [x] Exclude `@types/*` declaration packages and build-only CSS tools from the generated license report
- [x] Normalize shorthand repository URLs in `cmd/genlicenses` to ensure all URLs start with `https://`
- [x] Sanitize external URLs in `web/src/components/modal/AboutModal.tsx` so links open external repositories safely without relative 404s
- [x] Update footer "Licenses" button to open the About modal directly with the "Licenses" tab active
- [x] Verify license count and validity with `jokateko licenses` and Playwright E2E tests
- [x] Ensure `go test ./...` and `bun run check` pass cleanly

## Completion Summary
- **Completed At:** 2026-09-06T15:02:50Z

### What Was Done
Configured Bun bundler metafile export to bundled-packages.json, updated genlicenses to extract only bundled packages, normalized repository URLs to https://, updated AboutModal and Footer to sanitize URLs and trigger the licenses tab directly.

### Why / Rationale
Prevents shipping unused sub-dependencies and CSS tooling in license views, and resolves relative 404 links when clicking external repositories.
