+++
title = 'Snapshot injection'
tags = ['frontend', 'backend']
summary = "How `jokateko build` makes a static export: it replaces the `/* JOKATEKO_PAYLOAD_PLACEHOLDER */` comment inside the embedded UI's `<script id=\"jokateko-data\">` tag with the project's JSON snapshot."
+++

Bun already inlines all CSS and JS into one `index.html` at build time, so exporting needs no bundling at runtime. `internal/exporter` serializes tasks, milestones, strategies, glossary and the UI-facing config (`model.Snapshot`), safely escapes the JSON for a `<script>` context, and swaps it in for the placeholder. In `serve` mode the placeholder stays, and the UI boots in live mode instead.

On load, the UI validates the snapshot with Valibot. Invalid parts produce a warning banner, not a blank page.
