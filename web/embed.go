// Package web exposes embedded static assets for Jokateko's frontend Web UI.
package web

import "embed"

// Dist embeds the compiled web distribution assets.
//
//go:embed dist/*
var Dist embed.FS
