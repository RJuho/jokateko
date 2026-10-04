// Package web exposes embedded static assets for Jokateko's frontend Web UI.
package web

import (
	"bytes"
	"compress/gzip"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"sync"
)

// Dist embeds the pre-compressed web distribution assets and the separately
// shipped Mermaid runtime (loaded lazily by the UI, or injected by the exporter).
// To keep the compiled Go binary small, only gzip variants are embedded
// (index.html.gz ~90KB, mermaid.min.js.gz ~1.5MB instead of ~5.6MB raw).
//
//go:embed dist/index.html.gz dist/hashes.json dist/script.sha256 dist/style.sha256 dist/mermaid.min.js.gz dist/mermaid.json
var Dist embed.FS

var getHTML = sync.OnceValues(func() ([]byte, error) {
	gzData, err := Dist.ReadFile("dist/index.html.gz")
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded dist/index.html.gz: %w", err)
	}
	return gunzip(gzData)
})

func gunzip(gzData []byte) ([]byte, error) {
	gzReader, err := gzip.NewReader(bytes.NewReader(gzData))
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	decompressed, err := io.ReadAll(gzReader)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress embedded asset: %w", err)
	}
	return decompressed, nil
}

// GetHTML returns the uncompressed index.html content, decompressing
// from the embedded gzip asset on first call and memoizing it in memory.
func GetHTML() ([]byte, error) {
	return getHTML()
}

// GetGzipHTML returns the raw pre-compressed index.html.gz bytes directly.
func GetGzipHTML() ([]byte, error) {
	return Dist.ReadFile("dist/index.html.gz")
}

// MermaidRuntime describes the separately shipped Mermaid runtime (dist/mermaid.min.js
// from the lockfile-pinned npm package), produced by web/scripts/bundle.ts.
type MermaidRuntime struct {
	Version   string `json:"version"`
	Integrity string `json:"integrity"`
}

// AssetPath returns the versioned same-origin path the live UI loads the runtime from.
func (m MermaidRuntime) AssetPath() string {
	return "/assets/mermaid-" + m.Version + ".min.js"
}

// CDNURL returns the jsDelivr URL serving the byte-identical npm file for this version.
func (m MermaidRuntime) CDNURL() string {
	return "https://cdn.jsdelivr.net/npm/mermaid@" + m.Version + "/dist/mermaid.min.js"
}

var getMermaidRuntime = sync.OnceValues(func() (MermaidRuntime, error) {
	var m MermaidRuntime
	data, err := Dist.ReadFile("dist/mermaid.json")
	if err != nil {
		return m, fmt.Errorf("failed to read embedded dist/mermaid.json: %w", err)
	}
	if err := json.Unmarshal(data, &m); err != nil {
		return m, fmt.Errorf("failed to parse embedded dist/mermaid.json: %w", err)
	}
	if m.Version == "" || m.Integrity == "" {
		return m, fmt.Errorf("embedded dist/mermaid.json is missing version or integrity")
	}
	return m, nil
})

// GetMermaidRuntime returns the version and SRI hash of the embedded Mermaid runtime.
func GetMermaidRuntime() (MermaidRuntime, error) {
	return getMermaidRuntime()
}

// GetGzipMermaidJS returns the raw pre-compressed mermaid.min.js.gz bytes.
func GetGzipMermaidJS() ([]byte, error) {
	return Dist.ReadFile("dist/mermaid.min.js.gz")
}

var getMermaidJS = sync.OnceValues(func() ([]byte, error) {
	gzData, err := GetGzipMermaidJS()
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded dist/mermaid.min.js.gz: %w", err)
	}
	return gunzip(gzData)
})

// GetMermaidJS returns the uncompressed Mermaid runtime, memoized after first use.
func GetMermaidJS() ([]byte, error) {
	return getMermaidJS()
}
