+++
title = 'Implement Inlined Mermaid Diagram Rendering in Web UI'
status = 'done'
priority = 'medium'
tags = ['feature', 'frontend']
summary = 'Inlined client-side Mermaid diagram rendering into Web UI single-file bundle with strict security and DaisyUI theme synchronization.'
+++

# Implement Inlined Mermaid Diagram Rendering in Web UI

Add client-side Mermaid diagram rendering to the Web UI bundle so markdown descriptions and documentation containing ````mermaid ```` code blocks automatically render as rich SVG diagrams.

## Acceptance Criteria
- [x] Add `mermaid` dependency to `web/package.json` and verify Bun single-file bundling succeeds with `bun run build`
- [x] Initialize Mermaid with `securityLevel: 'strict'` to prevent XSS vulnerabilities from untrusted markdown content
- [x] Implement diagram rendering hook in `web/src/components/modal/TaskDetailModal.tsx` targeting `.language-mermaid` code blocks within rendered task bodies
- [x] Integrate DaisyUI theme detection so Mermaid diagram styling (colors/fonts) adapts to dark and light modes
- [x] Provide graceful error fallback retaining original `<pre><code>` block if Mermaid syntax fails to parse
- [x] Write end-to-end Playwright test in `tests/e2e/` verifying a task containing a Mermaid diagram renders an SVG element

## Completion Summary
- **Completed At:** 2026-09-04T12:41:23Z

### What Was Done
1. Bundled mermaid into web/package.json and updated web/scripts/bundle.ts to safely inline script content avoiding dollar-quote replacement pitfalls.
2. Implemented client-side Mermaid rendering in web/src/utils/mermaid.ts with securityLevel strict and DaisyUI dark/light theme detection.
3. Integrated diagram rendering hook and theme MutationObserver in web/src/components/modal/TaskDetailModal.tsx for task bodies and notes.
4. Fixed null safety ordering in TaskEditModal.tsx.
5. Added end-to-end tests in tests/e2e/mermaid.spec.ts verifying interactive SVG rendering and syntax error fallback.

### Why / Rationale
Enables rich visual architecture and flowchart documentation directly within Markdown tasks and notes while preserving zero-runtime external dependencies and ensuring XSS protection.
