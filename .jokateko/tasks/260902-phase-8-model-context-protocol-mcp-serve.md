+++
title = 'Phase 8: Model Context Protocol (MCP) Server & Proxy'
status = 'done'
priority = 'high'
milestone = '260902-mvp'
tags = ['backend', 'feature']
summary = 'Official Model Context Protocol server over stdio with proxy routing, 22 tools, spec-first guardrails, and prompts.'
dependencies = ['260902-phase-7-http-server-rest-api-sse-hub']
+++

# Phase 8: Model Context Protocol (MCP) Server & Proxy

AI Agent integration in `internal/mcp` and `internal/proxy`.

## Acceptance Criteria
- [x] 8.1 Initialize Official MCP Go SDK Server
- [x] 8.2 Implement MCP Task Tools (create, list, complete, update_item)
- [x] 8.3 Implement MCP Milestone Tools (create, list, get, update)
- [x] 8.4 Implement MCP Strategy & Glossary Tools
- [x] 8.5 Implement MCP Search & Tag Tools
- [x] 8.6 Implement MCP Resources & Prompts
- [x] 8.7 Implement Stdio-to-HTTP Proxy & Standalone Runner
- [x] 8.8 mcp Subcommand Implementation
- [x] 8.9 Unit & Integration Tests for MCP Server
