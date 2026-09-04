+++
title = 'Implement Inlined Mermaid Diagram Rendering in Web UI'
status = 'backlog'
priority = 'medium'
tags = ['frontend', 'feature']
summary = 'Bundle official mermaid library into Web UI via Bun to render fenced mermaid code blocks as interactive SVGs with DaisyUI theme matching.'
+++

# Implement Inlined Mermaid Diagram Rendering in Web UI

Add client-side Mermaid diagram rendering to the Web UI bundle so markdown descriptions and documentation containing ````mermaid ```` code blocks automatically render as rich SVG diagrams.

## Acceptance Criteria
- [ ] Add `mermaid` dependency to `web/package.json` and verify Bun single-file bundling succeeds with `bun run build`
- [ ] Initialize Mermaid with `securityLevel: 'strict'` to prevent XSS vulnerabilities from untrusted markdown content
- [ ] Implement diagram rendering hook in `web/src/components/modal/TaskDetailModal.tsx` targeting `.language-mermaid` code blocks within rendered task bodies
- [ ] Integrate DaisyUI theme detection so Mermaid diagram styling (colors/fonts) adapts to dark and light modes
- [ ] Provide graceful error fallback retaining original `<pre><code>` block if Mermaid syntax fails to parse
- [ ] Write end-to-end Playwright test in `tests/e2e/` verifying a task containing a Mermaid diagram renders an SVG element
