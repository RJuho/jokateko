+++
title = 'Open Source License Generator, CLI Commands (about, licenses), and Web Modal'
status = 'ready'
priority = 'high'
tags = ['feature', 'cli', 'frontend', 'docs']
summary = "Adopt the MIT License for Jokateko, automate third-party license extraction across Go and Frontend dependencies, and provide 'about' and 'licenses' CLI commands and a Web UI modal."
+++

# Open Source License Generator, CLI Commands (about, licenses), and Web Modal

Add MIT License to Jokateko, implement automated build-time extraction of all Go and Web third-party open-source licenses, and surface license and project details in CLI commands and a Web UI modal.

## Background & Rationale
Jokateko is an auditable, single-binary distribution. To ensure complete open-source transparency and compliance without manual maintenance burden, licenses from all Go modules and bundled Web UI dependencies must be automatically harvested at build/generate time and embedded directly into the binary.

## Requirements & Scope
1. **Jokateko MIT License**:
   - Add root `LICENSE` file under the MIT License for Jokateko.
2. **Automated License Harvester**:
   - Create a build generator (e.g. Go script or `cmd/genlicenses`) that automatically inspects Go module dependencies (`go.mod`/`go.sum`) and frontend packages (`web/package.json` / `node_modules`).
   - Extract package name, version, SPDX license identifier, author/repository URL, and full license text.
   - Output an embedded asset (e.g. `internal/version/licenses.json` or embedded Go struct) so no manual picking is required.
3. **CLI Commands**:
   - `jokateko about`: Prints Jokateko project info, version, commit, build date, MIT license summary, and references `jokateko licenses` for third-party dependencies.
   - `jokateko licenses`: Prints Jokateko's MIT license and a formatted summary of all bundled dependencies and their licenses, pointing back to `about` for project info.
   - Support `--full` or `--json` flags on `jokateko licenses` to inspect full texts or output machine-readable JSON.
4. **Web UI "About & Licenses" Modal**:
   - Add an "About / Licenses" trigger in the Web UI navigation.
   - Display Jokateko version, repository link, and MIT license.
   - Display a searchable table/list of third-party dependencies with license tags and expandable license texts.

## Acceptance Criteria
- [ ] Add MIT `LICENSE` file to the root of the repository
- [ ] Build automated license extraction generator for Go and Web dependencies with zero manual picking
- [ ] Embed harvested licenses directly into the Jokateko executable
- [ ] Implement `jokateko about` CLI command displaying project metadata and pointing to `licenses`
- [ ] Implement `jokateko licenses` CLI command listing all third-party licenses and pointing to `about`
- [ ] Add "About & Licenses" modal to Web UI with searchable dependency licenses and full text views
- [ ] Add unit tests for CLI commands and Playwright E2E test for the Web UI About & Licenses modal
