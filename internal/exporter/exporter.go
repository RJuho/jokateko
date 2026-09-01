package exporter

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/RJuho/jokateko/internal/config"
	"github.com/RJuho/jokateko/internal/store"
	"github.com/RJuho/jokateko/web"
)

// PayloadPlaceholder is the marker replaced with serialized snapshot JSON in the HTML bundle.
const PayloadPlaceholder = "/* JOKATEKO_PAYLOAD_PLACEHOLDER */"

// ErrPlaceholderNotFound is returned when the template HTML does not contain PayloadPlaceholder.
var ErrPlaceholderNotFound = errors.New("template HTML does not contain JOKATEKO_PAYLOAD_PLACEHOLDER")

// InjectSnapshot replaces the placeholder in the template HTML with the serialized snapshot JSON.
func InjectSnapshot(templateHTML []byte, snapshotJSON []byte) ([]byte, error) {
	placeholderBytes := []byte(PayloadPlaceholder)
	if !bytes.Contains(templateHTML, placeholderBytes) {
		return nil, ErrPlaceholderNotFound
	}

	return bytes.Replace(templateHTML, placeholderBytes, snapshotJSON, 1), nil
}

// Export builds the project snapshot from the store, injects it into the embedded Web UI HTML bundle,
// and writes the self-contained output file to outPath.
func Export(ctx context.Context, cfg *config.Config, st *store.Store, outPath string) (int64, error) {
	if outPath == "" {
		outPath = "dist-kanban/index.html"
	}

	templateBytes, err := fs.ReadFile(web.Dist, "dist/index.html")
	if err != nil {
		return 0, fmt.Errorf("failed to read embedded web dist/index.html: %w", err)
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
