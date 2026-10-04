package exporter

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/RJuho/jokateko/internal/config"
	"github.com/RJuho/jokateko/internal/store"
	"github.com/RJuho/jokateko/web"
)

// PayloadPlaceholder is the marker replaced with serialized snapshot JSON in the HTML bundle.
const PayloadPlaceholder = "/* JOKATEKO_PAYLOAD_PLACEHOLDER */"

// ErrPlaceholderNotFound is returned when the template HTML does not contain PayloadPlaceholder.
var ErrPlaceholderNotFound = errors.New("template HTML does not contain JOKATEKO_PAYLOAD_PLACEHOLDER")

// MermaidMode selects how a static export provides the Mermaid diagram runtime.
type MermaidMode string

const (
	// MermaidCDN loads the same Mermaid version from jsDelivr, verified by SRI (default).
	MermaidCDN MermaidMode = "cdn"
	// MermaidBundled inlines the runtime as an inert block, executed on first use (fully offline).
	MermaidBundled MermaidMode = "bundled"
	// MermaidNone disables diagram rendering; Mermaid blocks stay as code.
	MermaidNone MermaidMode = "none"
)

// ParseMermaidMode validates a --mermaidjs flag value.
func ParseMermaidMode(value string) (MermaidMode, error) {
	switch mode := MermaidMode(strings.ToLower(strings.TrimSpace(value))); mode {
	case MermaidCDN, MermaidBundled, MermaidNone:
		return mode, nil
	default:
		return "", fmt.Errorf("invalid mermaidjs mode %q (expected bundled, cdn or none)", value)
	}
}

// InjectMermaid marks the template with the chosen Mermaid mode (read by the UI loader)
// and, for MermaidBundled, appends the runtime as a non-executing
// <script type="text/plain"> block so it costs no parse time until a diagram is shown.
func InjectMermaid(templateHTML []byte, mode MermaidMode, runtimeJS []byte) ([]byte, error) {
	headEnd := bytes.Index(templateHTML, []byte("</head>"))
	if headEnd == -1 {
		return nil, errors.New("template HTML does not contain </head>")
	}
	meta := fmt.Sprintf("<meta name=\"jokateko-mermaid\" content=\"%s\" />\n", mode)

	var out bytes.Buffer
	out.Grow(len(templateHTML) + len(meta) + len(runtimeJS) + 128)
	out.Write(templateHTML[:headEnd])
	out.WriteString(meta)

	rest := templateHTML[headEnd:]
	if mode != MermaidBundled {
		out.Write(rest)
		return out.Bytes(), nil
	}

	if bytes.Contains(bytes.ToLower(runtimeJS), []byte("</script")) {
		return nil, errors.New("mermaid runtime contains \"</script\" and cannot be inlined")
	}
	bodyEnd := bytes.LastIndex(rest, []byte("</body>"))
	if bodyEnd == -1 {
		return nil, errors.New("template HTML does not contain </body>")
	}
	out.Write(rest[:bodyEnd])
	out.WriteString(`<script type="text/plain" id="jokateko-mermaid-src">`)
	out.Write(runtimeJS)
	out.WriteString("</script>\n")
	out.Write(rest[bodyEnd:])
	return out.Bytes(), nil
}

// InjectSnapshot replaces the placeholder in the template HTML with the serialized snapshot JSON.
func InjectSnapshot(templateHTML []byte, snapshotJSON []byte) ([]byte, error) {
	placeholderBytes := []byte(PayloadPlaceholder)
	if !bytes.Contains(templateHTML, placeholderBytes) {
		return nil, ErrPlaceholderNotFound
	}

	return bytes.Replace(templateHTML, placeholderBytes, snapshotJSON, 1), nil
}

// Export builds the project snapshot from the store, injects it into the embedded Web UI HTML bundle,
// provides the Mermaid runtime according to mermaidMode, and writes the output file to outPath.
func Export(ctx context.Context, cfg *config.Config, st *store.Store, outPath string, mermaidMode MermaidMode) (int64, error) {
	if outPath == "" {
		outPath = "dist-kanban/index.html"
	}

	templateBytes, err := web.GetHTML()
	if err != nil {
		return 0, fmt.Errorf("failed to read embedded web template: %w", err)
	}

	var runtimeJS []byte
	if mermaidMode == MermaidBundled {
		if runtimeJS, err = web.GetMermaidJS(); err != nil {
			return 0, fmt.Errorf("failed to read embedded mermaid runtime: %w", err)
		}
	}
	// Inject into the template before the snapshot, so snapshot content can never match the markers
	templateBytes, err = InjectMermaid(templateBytes, mermaidMode, runtimeJS)
	if err != nil {
		return 0, fmt.Errorf("failed to inject mermaid runtime: %w", err)
	}

	snap, err := BuildSnapshot(ctx, cfg, st)
	if err != nil {
		return 0, fmt.Errorf("failed to build snapshot: %w", err)
	}

	snapJSON, err := SerializeSnapshot(snap)
	if err != nil {
		return 0, fmt.Errorf("failed to serialize snapshot: %w", err)
	}

	injectedHTML, err := InjectSnapshot(templateBytes, snapJSON)
	if err != nil {
		return 0, fmt.Errorf("failed to inject snapshot: %w", err)
	}

	outDir := filepath.Dir(outPath)
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return 0, fmt.Errorf("failed to create export directory %q: %w", outDir, err)
	}

	if err := os.WriteFile(outPath, injectedHTML, 0644); err != nil {
		return 0, fmt.Errorf("failed to write export file %q: %w", outPath, err)
	}

	return int64(len(injectedHTML)), nil
}
