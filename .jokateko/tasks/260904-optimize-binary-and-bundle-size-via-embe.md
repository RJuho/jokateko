+++
title = 'Optimize Binary and Bundle Size via Embedded Web UI Gzip Compression and Stripped Symbols'
status = 'done'
priority = 'high'
tags = ['backend', 'build', 'optimization', 'performance']
summary = 'Optimized Jokateko standalone binary size from 26MB to 16MB by embedding pre-compressed gzip Web UI assets and stripping DWARF debug info.'
+++

# Optimize Binary and Bundle Size via Embedded Web UI Gzip Compression and Stripped Symbols

Reduce standalone Jokateko binary and Web UI asset footprint by compressing the embedded web distribution asset and stripping debug symbols.

## Acceptance Criteria
- [x] Update `web/scripts/bundle.ts` to generate both `dist/index.html` and pre-compressed `dist/index.html.gz`
- [x] Update `web/embed.go` to embed the pre-compressed `dist/index.html.gz` instead of the raw 3.6MB HTML file, reducing embedded `.rodata` size
- [x] Update `internal/server/server.go` to serve the pre-compressed gzipped HTML directly with `Content-Encoding: gzip` when clients support gzip (with decompression fallback)
- [x] Update `internal/exporter/exporter.go` and tests to transparently decompress the embedded gzipped template when generating standalone offline exports
- [x] Verify build with `-ldflags="-s -w"` reduces final executable size to ~16MB
- [x] Validate that all Go backend tests, Web UI build, and Playwright E2E tests pass deterministically

## Completion Summary
- **Completed At:** 2026-09-04T12:47:05Z

### What Was Done
1. Updated web/scripts/bundle.ts to generate both raw web/dist/index.html and maximum-level pre-compressed web/dist/index.html.gz.
2. Updated web/embed.go to explicitly embed dist/index.html.gz and metadata, saving ~2.7MB in .rodata, while providing thread-safe sync.OnceValues memoized uncompressed HTML access via web.GetHTML() and direct gzip stream access via web.GetGzipHTML().
3. Updated internal/server/server.go to serve pre-compressed gzip bytes with Content-Encoding: gzip when clients support gzip (instant response, ~984KB transfer), with transparent fallback to uncompressed HTML.
4. Updated internal/exporter/exporter.go to decompress the template on the fly when exporting offline static HTML snapshots.
5. Added unit and integration tests verifying both gzipped and uncompressed HTTP responses and decompression fidelity.
6. Reduced final executable size from 26MB to 16MB (16.3MB) when built with -ldflags="-s -w".

### Why / Rationale
Reduces the executable footprint by ~38% (saving 10MB) and network transfer size by 73% (from 3.6MB to 984KB) with zero runtime performance penalty, avoiding the antivirus false positives and memory mapping drawbacks of UPX.
