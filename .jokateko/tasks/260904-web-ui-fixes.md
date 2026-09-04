+++
title = 'Web UI Fixes'
status = 'done'
priority = 'medium'
tags = ['bugfix', 'frontend', 'ui']
summary = 'Resolved Web UI defects including badge styling, drag-and-drop status mutations, live-reload SSE synchronization, Mermaid rendering across modals, strategies, and glossary views with zoom toolbars, prose code spacing, and task ID copy button width stability.'
+++

# Web UI Fixes

Resolve identified UI defects, interaction issues, and UX improvements in the Web UI.

## Acceptance Criteria
- [x] Make all Tag badges `badge-ghost` by default, and `badge-soft badge-primary` when selected/active
- [x] Fix drag and drop column status changes in Web UI (resolve 405 Method Not Allowed error) and create an E2E test
- [x] Fix live reload real-time SSE broadcasting across Web UI views and create an E2E test
- [x] Fix Mermaid diagram theme switching bug (prevent black boxes on light theme and errors on theme change)
- [x] Add expansion/zoom toolbar or modal enlargement for Mermaid diagrams and task detail modal
- [x] Fix typography spacing issues in markdown prose containing inline `<code>` elements
- [x] Prevent content jump when clicking task ID by preserving element width during "Copied!" feedback state
- [x] Ensure Mermaid diagrams render and interact correctly in Strategies and Glossary views and add E2E test coverage

## Completion Summary
- **Completed At:** 2026-09-04T14:14:50Z

### What Was Done
Implemented PUT /api/tasks/{id}/status endpoint and fixed card drag-and-drop; strengthened live reload SSE event broadcasting and client handlers; added CSP support for SVG inline styles; integrated Mermaid diagram rendering and theme reactivity into task modal, strategies view, and glossary view; added diagram zoom and lightbox preview toolbars; corrected inline code prose spacing; stabilized task ID copy button dimensions; and wrote comprehensive Playwright E2E tests for all new behaviors.

### Why / Rationale
Addresses user-reported visual and functional regressions to ensure smooth real-time workflow, intuitive diagram exploration, robust dark/light theme switching, and high test confidence.
