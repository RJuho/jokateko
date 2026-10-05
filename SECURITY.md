# Security

## Supported versions

Only the latest release receives security fixes.

## Reporting a vulnerability

Please **do not open a public issue**. Report it privately through [GitHub Security Advisories](https://github.com/RJuho/jokateko/security/advisories/new) instead. Include the version (`jokateko version`), steps to reproduce, and the impact you see.

Jokateko is maintained on a best-effort basis. We will respond as soon as we can, keep you updated, and credit you in the advisory unless you prefer otherwise.

## Threat model

Jokateko is a **single-user tool for your own machine**.

- **No authentication.** Anyone who can reach the server can read and change the whole board, and the board's files are in your repository.
- **Loopback only by default.** `serve` binds `127.0.0.1:8080`. Never expose it on a network or the internet. Use `0.0.0.0` (or `JOKATEKO_HOST=0.0.0.0`) only inside a trusted container where only the forwarded port is reachable, as this repository's devcontainer does.
- **Browser protections**, against other websites you visit:
  - Cross-site requests that change state are rejected (Go `CrossOriginProtection`), and API writes must be JSON.
  - On loopback connections, the `Host` header must be a local name, which blocks DNS rebinding. The check does **not** apply to non-loopback connections, which is one more reason not to bind to `0.0.0.0` outside a container.
  - CORS only allows the origins in `[server.security] cors_allowed_origins`.
  - A strict Content Security Policy only allows the bundled script and style (by hash).
- **Untrusted Markdown.** `.jokateko/` content can come from anyone who opens a pull request or publishes a repository you clone. Raw HTML in task, milestone, strategy and glossary bodies is shown as escaped text (HTML comments are hidden), and links with `javascript:`, `vbscript:`, `file:` or non-image `data:` URLs are dropped, so a body can never inject live markup or script.
- **Pinned assets.** Mermaid is loaded from the binary or from jsDelivr at the exact `bun.lock` version, verified with Subresource Integrity.
- **Static exports contain everything.** A `jokateko build` HTML file includes all tasks, strategies and glossary terms. Share it only with people who may see the repository. Each export carries the same Content Security Policy as `serve` in a `<meta http-equiv>` tag, plus only the Mermaid source its `--mermaidjs` mode needs: the exact jsDelivr URL and SRI hash for `cdn`, or the SRI hash of the inline runtime for `bundled`. Setting `[server.security.csp] enabled = false` also removes it from exports.
- **Agents.** MCP tools can change everything in `.jokateko/`. Set `[mcp] allow_mutations = false` for read-only agents.

Details: the [REST API and SSE](.jokateko/strategies/rest-api-and-sse.md) strategy.
