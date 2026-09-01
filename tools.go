//go:build tools

// Package tools tracks tool and build dependencies for Jokateko.
// See: https://go.dev/wiki/Modules#how-can-i-track-tool-dependencies-for-a-module
package tools

import (
	_ "github.com/fsnotify/fsnotify"
	_ "github.com/modelcontextprotocol/go-sdk"
	_ "github.com/pelletier/go-toml/v2"
	_ "github.com/yuin/goldmark"
	_ "modernc.org/sqlite"
)
