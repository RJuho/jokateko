package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/RJuho/jokateko/internal/config"
	"github.com/RJuho/jokateko/internal/mcp"
	"github.com/RJuho/jokateko/internal/server"
	"github.com/RJuho/jokateko/internal/service"
	"github.com/RJuho/jokateko/internal/store"
	"github.com/RJuho/jokateko/internal/watcher"
	"github.com/RJuho/jokateko/internal/writer"
)

func cmdServe(ctx context.Context, args []string, stdout, stderr io.Writer) int {
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

	if *timeoutFlag > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, *timeoutFlag)
		defer cancel()
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

	// 3. Suppression cache, atomic writer, SSE hub and the shared mutation service.
	// REST and MCP mutations both go through svc, so both reach the Web UI via SSE.
	sc := writer.NewSuppressionCache(3 * time.Second)
	wr := writer.New(sc)

	sse := server.NewSSEHub()
	sse.Start()
	defer sse.Stop()

	svc := service.New(cfg, workspaceDir, st, wr, sse)

	// 4. Load project files and keep the store in sync with external edits.
	logf := func(format string, args ...any) {
		fmt.Fprintf(stderr, format+"\n", args...)
	}
	stopPipeline, err := watcher.StartPipeline(ctx, st, sc, svc.Dirs(), func(e watcher.IngestEvent) {
		broadcastIngest(ctx, sse, st, e)
	}, logf)
	if err != nil {
		logf("warning: %v", err)
	}
	defer stopPipeline()

	// 5. Start HTTP Server with MCP support (Streamable HTTP transport)
	srv := server.New(svc, sse)
	srv.SetMCPHandler(mcp.New(svc).HTTPHandler())

	if err := srv.Start(); err != nil {
		fmt.Fprintf(stderr, "failed to start HTTP server: %v\n", err)
		return 1
	}

	fmt.Fprintf(stdout, "Jokateko daemon active: http://%s\n", srv.Addr())
	fmt.Fprintf(stdout, "Serving project %q at %s\n", cfg.Project.Name, workspaceDir)

	<-ctx.Done()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		fmt.Fprintln(stdout, "\nTimeout reached, shutting down...")
	} else {
		fmt.Fprintln(stdout, "\nReceived shutdown signal, shutting down...")
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		fmt.Fprintf(stderr, "error during server shutdown: %v\n", err)
	}

	return 0
}

// broadcastIngest forwards an externally made file change to Web UI clients,
// sending the freshly indexed entity when it can be loaded.
func broadcastIngest(ctx context.Context, sse *server.SSEHub, st *store.Store, e watcher.IngestEvent) {
	if e.Op == watcher.OpDelete {
		sse.Broadcast(e.EntityType+".deleted", map[string]string{"id": e.EntityID})
		return
	}

	eventName := e.EntityType + ".updated"
	var (
		payload any
		err     error
	)
	switch e.EntityType {
	case "task":
		payload, err = st.GetTask(ctx, e.EntityID)
	case "milestone":
		payload, err = st.GetMilestone(ctx, e.EntityID)
	case "strategy":
		payload, err = st.GetStrategy(ctx, e.EntityID)
	case "glossary":
		payload, err = st.GetGlossaryTerm(ctx, e.EntityID)
	}
	if payload == nil || err != nil {
		payload = map[string]string{"id": e.EntityID}
	}
	sse.Broadcast(eventName, payload)
}
