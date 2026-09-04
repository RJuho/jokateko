package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/RJuho/jokateko/internal/config"
	"github.com/RJuho/jokateko/internal/mcp"
	"github.com/RJuho/jokateko/internal/server"
	"github.com/RJuho/jokateko/internal/store"
	"github.com/RJuho/jokateko/internal/watcher"
	"github.com/RJuho/jokateko/internal/writer"
	sdk_mcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

func cmdServe(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dirFlag := fs.String("dir", ".", "Project root directory to serve")
	portFlag := fs.Int("port", -1, "HTTP server port (overrides config.toml, 0 selects random free port)")
	timeoutFlag := fs.Duration("timeout", 0, "Automatically shut down after duration (useful for smoke testing)")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	workspaceDir, err := filepath.Abs(*dirFlag)
	if err != nil {
		fmt.Fprintf(stderr, "failed to resolve directory %q: %v\n", *dirFlag, err)
		return 1
	}

	// 1. Load configuration
	cfg, err := config.Load(workspaceDir)
	if err != nil {
		fmt.Fprintf(stderr, "failed to load configuration: %v\n", err)
		return 1
	}

	if *portFlag >= 0 {
		cfg.Server.Port = *portFlag
	}

	// 2. Initialize in-memory store
	st, err := store.OpenMemory()
	if err != nil {
		fmt.Fprintf(stderr, "failed to initialize in-memory store: %v\n", err)
		return 1
	}
	defer func() {
		_ = st.Close()
	}()

	// 3. Initialize suppression cache and atomic writer
	sc := writer.NewSuppressionCache(3 * time.Second)
	wr := writer.New(sc)

	// 4. Set up ingestion pipeline
	resolveDir := func(rel string) string {
		if filepath.IsAbs(rel) {
			return rel
		}
		return filepath.Join(workspaceDir, rel)
	}

	ingestCfg := watcher.IngestConfig{
		TasksDir:      resolveDir(cfg.Paths.Tasks),
		MilestonesDir: resolveDir(cfg.Paths.Milestones),
		StrategiesDir: resolveDir(cfg.Paths.Strategies),
		GlossaryDir:   resolveDir(cfg.Paths.Glossary),
		ConfigFile:    filepath.Join(workspaceDir, ".jokateko", "config.toml"),
	}

	pipeline := watcher.NewPipeline(st, sc, ingestCfg)

	// Ingest all existing project markdown files into memory
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := pipeline.ProcessAll(ctx); err != nil {
		fmt.Fprintf(stderr, "warning: error during initial directory scan: %v\n", err)
	}

	// 5. Initialize SSE Hub
	sse := server.NewSSEHub()
	sse.Start()
	defer sse.Stop()

	// Connect pipeline changes to SSE hub
	pipeline.SetOnEntityChange(func(e watcher.IngestEvent) {
		opName := "updated"
		if e.Op == watcher.OpDelete {
			opName = "deleted"
		}
		eventName := fmt.Sprintf("%s.%s", e.EntityType, opName)

		if e.Op == watcher.OpDelete {
			sse.Broadcast(eventName, map[string]string{
				"id": e.EntityID,
			})
			return
		}

		ctx := context.Background()
		switch e.EntityType {
		case "task":
			if task, err := st.GetTask(ctx, e.EntityID); err == nil {
				sse.Broadcast(eventName, task)
				return
			}
		case "milestone":
			if ms, err := st.GetMilestone(ctx, e.EntityID); err == nil {
				sse.Broadcast(eventName, ms)
				return
			}
		case "strategy":
			if strat, err := st.GetStrategy(ctx, e.EntityID); err == nil {
				sse.Broadcast(eventName, strat)
				return
			}
		case "glossary":
			if term, err := st.GetGlossaryTerm(ctx, e.EntityID); err == nil {
				sse.Broadcast(eventName, term)
				return
			}
		}

		sse.Broadcast(eventName, map[string]string{
			"id": e.EntityID,
		})
	})

	// 6. Start filesystem watcher
	watchDirs := []string{
		ingestCfg.TasksDir,
		ingestCfg.MilestonesDir,
		ingestCfg.StrategiesDir,
		ingestCfg.GlossaryDir,
	}

	fsw, err := watcher.New(watchDirs, 50*time.Millisecond)
	if err != nil {
		fmt.Fprintf(stderr, "warning: failed to start filesystem watcher: %v\n", err)
	} else {
		defer func() {
			_ = fsw.Close()
		}()
		fsw.Start(ctx)

		go func() {
			for ev := range fsw.Events() {
				if err := pipeline.HandleEvent(ctx, ev); err != nil {
					logError(stderr, fmt.Sprintf("failed to handle file event for %s: %v", ev.Path, err))
				}
			}
		}()
	}

	// 7. Start HTTP Server with MCP support
	srv := server.New(cfg, workspaceDir, st, wr, sse)
	mcpSrv := mcp.New(cfg, workspaceDir, st, wr)
	srv.SetMCPHandler(sdk_mcp.NewSSEHandler(func(req *http.Request) *sdk_mcp.Server {
		return mcpSrv.MCPServer()
	}, nil))

	if err := srv.Start(); err != nil {
		fmt.Fprintf(stderr, "failed to start HTTP server: %v\n", err)
		return 1
	}

	fmt.Fprintf(stdout, "Jokateko daemon active: http://%s\n", srv.Addr())
	fmt.Fprintf(stdout, "Serving project %q at %s\n", cfg.Project.Name, workspaceDir)

	// Wait for termination signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	var timerCh <-chan time.Time
	if *timeoutFlag > 0 {
		timer := time.NewTimer(*timeoutFlag)
		defer timer.Stop()
		timerCh = timer.C
	}

	select {
	case sig := <-sigCh:
		fmt.Fprintf(stdout, "\nReceived signal %s, shutting down...\n", sig)
	case <-timerCh:
		fmt.Fprintln(stdout, "\nTimeout reached, shutting down...")
	case <-ctx.Done():
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		fmt.Fprintf(stderr, "error during server shutdown: %v\n", err)
	}

	return 0
}

func logError(w io.Writer, msg string) {
	fmt.Fprintf(w, "[ERROR] %s\n", msg)
}
