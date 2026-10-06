+++
title = 'Use the new logo on the README cover and in the UI footer'
status = 'done'
priority = 'medium'
milestone = '261005-public-release-v010'
tags = ['docs', 'release', 'ui']
summary = 'The new logo (web/src/logo.svg) is on the README cover and social preview, in the Web UI footer and in the About modal header, all from one SVG source.'
created_at = '2026-10-06T08:48:29Z'
changed_at = '2026-10-06T13:34:56Z'
+++

## Context
`web/src/logo.svg` is the new Jokateko logo: a blue-gradient check mark between a `{ }` card and a `>_` terminal card. The cover from `make screenshots` (see `261005-write-public-readme-and-fold-in-commands`) still uses a placeholder `.logo` "J" tile. The maintainer asked for a new cover with the real logo, and for the logo in the UI's bottom bar.

## Acceptance Criteria
- [x] `web/src/logo.svg` tidied into the single source: one gradient, no full-viewBox clip-path, `role="img"` + `<title>`, same rendering
- [x] `tests/screenshots/cover.html` has a `/*LOGO*/` slot, filled from `web/src/logo.svg` by `capture.spec.ts`; cover accents retuned to the logo blues
- [x] `make screenshots` re-run; `.github/assets/cover.jpg` and `.github/assets/social-preview.jpg` refreshed (social preview under 1 MB)
- [x] Footer shows the logo (`data-testid="footer-logo"`) before the "Jokateko" link name; footer height and colours unchanged
- [x] No CSP errors for the logo in live `serve` mode or in a `jokateko build` export
- [x] E2E check for the footer logo; typecheck, lint, Go tests and e2e pass

## Notes

### [2026-10-06 08:51 UTC]

**Done, ready for review.**
- **Logo source:** `web/src/logo.svg` now has one gradient (`jokateko-logo-gradient`), no full-viewBox clip-path, no XML prolog, and `role="img"` + `<title>`. The third gradient's axis differed from the others by less than one unit, so merging it causes no visible change.
- **Cover:** `tests/screenshots/cover.html` uses a `<!--LOGO-->` slot, an HTML comment instead of the planned `/*LOGO*/`, because it sits in markup. `capture.spec.ts` fills it with the raw SVG, so `screenshots/cover.html` stays a single file. The logo is 150px wide with a soft blue `drop-shadow`. The purple glow became logo blue and the unused accent vars were removed.
- **Assets:** `make screenshots` re-run; `.github/assets/cover.jpg` is 210 KB and `social-preview.jpg` is 98 KB. Remember to re-upload the social preview under repo Settings → Social preview.
- **Footer:** `Footer.tsx` imports the SVG as text (`with { type: 'text' }`, plus a `*.svg` module declaration in `global.d.ts`) and renders it as a `data:image/svg+xml` `<img>` (`h-3.5`, decorative `alt=''`) inside the existing "Jokateko" link. The footer height and colours are unchanged. Checked in live `serve` and in a `jokateko build` export, in light and dark mode: the logo decodes (naturalWidth 178) and there are no CSP console errors.
- **Tests:** new e2e test "footer shows the logo next to the project link" in `about-licenses.spec.ts`. Typecheck, `make lint`, `bun run check` and `go test ./...` pass. Full e2e: 59/60. The one failure is a flaky, unrelated test (`timestamps.spec.ts:134`, about 1 in 10 runs), filed as `261006-fix-flaky-milestone-timeframe-e2e-test`.
- `make install` done.

### [2026-10-06 13:33 UTC]

**About modal logo (maintainer request 2026-10-06).** The placeholder "J" tile in the About modal header is now the logo: an `<img>` with `size-8 object-contain`, decorative `alt=''` and `data-testid="about-modal-logo"`. It keeps the tile's 32×32 footprint, so the header layout doesn't shift. The data URI moved to `web/src/utils/logo.ts` (`logoSrc`), which Footer and AboutModal share. The `about-licenses.spec.ts` About test now also asserts that the modal logo is visible and decodes. Checked in a `jokateko build` export in light and dark mode: it renders with no console errors. Typecheck, `bun run check` and the spec pass; `make install` done.

## Completion Summary
- **Completed At:** 2026-10-06T13:34:56Z

### What Was Done
`web/src/logo.svg` was tidied: one gradient, no full-viewBox clip-path or XML prolog, and `role="img"` + `<title>`. The cover template (`tests/screenshots/cover.html`) replaces the "J" placeholder with a `<!--LOGO-->` slot that `capture.spec.ts` fills from the SVG. The logo is 150px wide with a soft blue drop-shadow, and the purple glow became logo blue. `make screenshots` was re-run and `.github/assets/cover.jpg` (210 KB) and `social-preview.jpg` (98 KB) were refreshed. In the Web UI, `web/src/utils/logo.ts` exports the SVG as a data URI (`logoSrc`, imported `with { type: 'text' }`, plus a `*.svg` declaration in `global.d.ts`). The footer shows it before the "Jokateko" link, and the About modal shows it in place of the 32px "J" tile. `about-licenses.spec.ts` asserts that both images are visible and decode.

### Why / Rationale
There is a single SVG source for the cover and the UI, so a future logo change is one file edit plus `make screenshots`. A data URI `<img>` was chosen over inline SVG so repeated instances never duplicate gradient IDs, and because the default CSP already allows `img-src data:`, both in live serve and in static exports (verified, no console errors). The footer height, colours and About header layout are unchanged; only the icons were added, in line with the "UI look is final" rule. The social preview still has to be uploaded by hand in the GitHub repository settings.
