package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/RJuho/jokateko/internal/config"
	"github.com/RJuho/jokateko/internal/exporter"
	"github.com/RJuho/jokateko/internal/store"
	"github.com/RJuho/jokateko/internal/watcher"
	"github.com/RJuho/jokateko/internal/writer"
)

func cmdBuild(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("build", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dirFlag := fs.String("dir", ".", "Project root directory")
	outFlag := fs.String("out", "", "Output file path (default: dist-kanban/index.html or config paths.export)")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	workspaceDir, err := filepath.Abs(*dirFlag)
	if err != nil {
		fmt.Fprintf(stderr, "error: invalid workspace directory: %v\n", err)
		return 1
	}

	cfg, err := config.Load(workspaceDir)
	if err != nil {
		cfg = config.Default(workspaceDir)
	}

	outPath := *outFlag
	if outPath == "" {
		if cfg.Paths.Export != "" {
			outPath = cfg.Paths.Export
		} else {
			outPath = "dist-kanban/index.html"
		}
	}
	if filepath.Ext(outPath) == "" {
		outPath = filepath.Join(outPath, "index.html")
	}
	if !filepath.IsAbs(outPath) {
		outPath = filepath.Join(workspaceDir, outPath)
	}

	// 1. Initialize store
	st, err := store.OpenMemory()
	if err != nil {
		fmt.Fprintf(stderr, "error: failed to open in-memory store: %v\n", err)
		return 1
	}
	defer func() {
		_ = st.Close()
	}()

	// 2. Writer and suppression cache
	sc := writer.NewSuppressionCache(time.Second)

	// 3. Ingestion pipeline
	resolveDir := func(p string) string {
		if filepath.IsAbs(p) {
			return p
		}
		return filepath.Join(workspaceDir, p)
	}

	ingestCfg := watcher.IngestConfig{
		TasksDir:      resolveDir(cfg.Paths.Tasks),
		MilestonesDir: resolveDir(cfg.Paths.Milestones),
		StrategiesDir: resolveDir(cfg.Paths.Strategies),
		GlossaryDir:   resolveDir(cfg.Paths.Glossary),
	}
	pipeline := watcher.NewPipeline(st, sc, ingestCfg)

	ctx := context.Background()
	if err := pipeline.ProcessAll(ctx); err != nil {
		fmt.Fprintf(stderr, "warning: error during initial directory scan: %v\n", err)
	}

	// 4. Export snapshot
	size, err := exporter.Export(ctx, cfg, st, outPath)
	if err != nil {
		fmt.Fprintf(stderr, "error: failed to export snapshot: %v\n", err)
		return 1
	}

	fmt.Fprintf(stdout, "Exported self-contained Kanban snapshot to %s (%d bytes)\n", outPath, size)
	return 0
}
