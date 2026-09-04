// Package web exposes embedded static assets for Jokateko's frontend Web UI.
package web

import (
	"bytes"
	"compress/gzip"
	"embed"
	"fmt"
	"io"
	"sync"
)

// Dist embeds the pre-compressed web distribution assets.
// To keep the compiled Go binary small, we embed dist/index.html.gz (~980KB)
// rather than the raw 3.6MB uncompressed HTML file, saving ~2.7MB in .rodata.
//
//go:embed dist/index.html.gz dist/hashes.json dist/script.sha256 dist/style.sha256
var Dist embed.FS

var getHTML = sync.OnceValues(func() ([]byte, error) {
	gzData, err := Dist.ReadFile("dist/index.html.gz")
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded dist/index.html.gz: %w", err)
	}

	gzReader, err := gzip.NewReader(bytes.NewReader(gzData))
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	decompressed, err := io.ReadAll(gzReader)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress embedded web UI: %w", err)
	}
	return decompressed, nil
})

// GetHTML returns the uncompressed index.html content, decompressing
// from the embedded gzip asset on first call and memoizing it in memory.
func GetHTML() ([]byte, error) {
	return getHTML()
}

// GetGzipHTML returns the raw pre-compressed index.html.gz bytes directly.
func GetGzipHTML() ([]byte, error) {
	return Dist.ReadFile("dist/index.html.gz")
}
