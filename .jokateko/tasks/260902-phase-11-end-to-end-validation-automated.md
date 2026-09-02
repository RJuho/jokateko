+++
title = 'Phase 11: End-to-End Validation & Automated Testing'
status = 'in_progress'
priority = 'critical'
milestone = '260902-mvp'
tags = ['backend', 'frontend']
summary = 'Playwright test suite, automated E2E scenarios in Chromium headless, SHA-256 CSP hash signing, and project dogfooding.'
dependencies = ['260902-phase-10-bun-web-ui']
+++

# Phase 11: End-to-End Validation & Automated Testing

Deterministic automated feedback loops, Playwright E2E testing, and dogfooding.

## Acceptance Criteria
- [x] 11.1 Playwright E2E Test Suite Setup (Chromium headless, make e2e-test)
- [x] 11.2 E2E Test Scenarios (board, task mutation, static export, valibot resilience)
- [x] 11.3 Project Dogfooding (jokateko init, parse, MCP tools, build export)
- [ ] 11.4 GitHub Actions CI/CD Release Workflow (.github/workflows/release.yml)
