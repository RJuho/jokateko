// Package proxy implements the Model Context Protocol (MCP) stdio-to-HTTP proxy
// and standalone in-process fallback runner for Jokateko.
package proxy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/RJuho/jokateko/internal/config"
	"github.com/RJuho/jokateko/internal/mcp"
	"github.com/RJuho/jokateko/internal/store"
	"github.com/RJuho/jokateko/internal/watcher"
	"github.com/RJuho/jokateko/internal/writer"
	sdk_mcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Runner manages the lifecycle of the stdio MCP bridge or standalone runner.
type Runner struct {
	cfg          *config.Config
	workspaceDir string
	in           io.Reader
	out          io.Writer
	stderr       io.Writer
	probeTimeout time.Duration
}

type nopWriteCloser struct {
	io.Writer
}

func (nopWriteCloser) Close() error {
	return nil
}

// NewRunner creates a new Runner for stdio MCP interaction.
func NewRunner(cfg *config.Config, workspaceDir string, in io.Reader, out io.Writer, stderr io.Writer) *Runner {
	if cfg == nil {
		cfg = config.Default(workspaceDir)
	}
	if in == nil {
		in = os.Stdin
	}
	if out == nil {
		out = os.Stdout
	}
	if stderr == nil {
		stderr = os.Stderr
	}
	return &Runner{
		cfg:          cfg,
		workspaceDir: workspaceDir,
		in:           in,
		out:          out,
		stderr:       stderr,
		probeTimeout: 150 * time.Millisecond,
	}
}

// SetProbeTimeout sets the timeout duration for probing daemon health.
func (r *Runner) SetProbeTimeout(d time.Duration) {
	r.probeTimeout = d
}

// Run executes the MCP proxy workflow:
// 1. Probes daemon health at GET http://127.0.0.1:<port>/api/health
// 2. If daemon is active: bridges stdio to the running daemon's /api/mcp endpoint
// 3. If daemon is inactive: runs an in-memory SQLite store, watcher, and standalone MCP server
func (r *Runner) Run(ctx context.Context) error {
	if r.probeDaemon(ctx) {
		return r.runProxy(ctx)
	}
	return r.runStandalone(ctx)
}

// ProbeDaemon checks if the Jokateko daemon is running on the configured port.
func (r *Runner) ProbeDaemon(ctx context.Context) bool {
	return r.probeDaemon(ctx)
}

func (r *Runner) probeDaemon(ctx context.Context) bool {
	port := r.cfg.Server.Port
	if port <= 0 {
		port = 8080
	}

	timeout := r.probeTimeout
	if timeout <= 0 {
		timeout = 150 * time.Millisecond
	}

	probeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	url := fmt.Sprintf("http://127.0.0.1:%d/api/health", port)
	req, err := http.NewRequestWithContext(probeCtx, http.MethodGet, url, nil)
	if err != nil {
		return false
	}

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

func (r *Runner) runProxy(ctx context.Context) error {
	port := r.cfg.Server.Port
	if port <= 0 {
		port = 8080
	}
	fmt.Fprintf(r.stderr, "jokateko: connected to running daemon on port %d (proxy mode)\n", port)

	mcpEndpoint := fmt.Sprintf("http://127.0.0.1:%d/api/mcp", port)
	daemonTransport := &sdk_mcp.SSEClientTransport{
		Endpoint: mcpEndpoint,
	}

	daemonConn, err := daemonTransport.Connect(ctx)
	if err != nil {
		return fmt.Errorf("failed to connect to daemon MCP endpoint: %w", err)
	}
	defer daemonConn.Close()

	ioTransport := &sdk_mcp.IOTransport{
		Reader: io.NopCloser(r.in),
		Writer: nopWriteCloser{Writer: r.out},
	}
	clientConn, err := ioTransport.Connect(ctx)
	if err != nil {
		return fmt.Errorf("failed to initialize stdio transport: %w", err)
	}
	defer clientConn.Close()

	errCh := make(chan error, 2)

	// Pump client -> daemon
	go func() {
		for {
			msg, err := clientConn.Read(ctx)
			if err != nil {
				errCh <- err
				return
			}
			if err := daemonConn.Write(ctx, msg); err != nil {
				errCh <- err
				return
			}
		}
	}()

	// Pump daemon -> client
	go func() {
		for {
			msg, err := daemonConn.Read(ctx)
			if err != nil {
				errCh <- err
				return
			}
			if err := clientConn.Write(ctx, msg); err != nil {
				errCh <- err
				return
			}
		}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errCh:
		if errors.Is(err, io.EOF) || errors.Is(err, context.Canceled) {
			return nil
		}
		return err
	}
}

func (r *Runner) runStandalone(ctx context.Context) error {
	fmt.Fprintf(r.stderr, "jokateko: daemon inactive, running in standalone in-process mode\n")

	// 1. Initialize in-memory SQLite store
	st, err := store.OpenMemory()
	if err != nil {
		return fmt.Errorf("failed to initialize in-memory store: %w", err)
	}
	defer func() {
		_ = st.Close()
	}()

	// 2. Writer and suppression cache
	sc := writer.NewSuppressionCache(time.Second)
	wr := writer.New(sc)

	// 3. Ingestion pipeline
	resolveDir := func(p string) string {
		if filepath.IsAbs(p) {
			return p
		}
		return filepath.Join(r.workspaceDir, p)
	}

	ingestCfg := watcher.IngestConfig{
		TasksDir:      resolveDir(r.cfg.Paths.Tasks),
		MilestonesDir: resolveDir(r.cfg.Paths.Milestones),
		StrategiesDir: resolveDir(r.cfg.Paths.Strategies),
		GlossaryDir:   resolveDir(r.cfg.Paths.Glossary),
	}
	pipeline := watcher.NewPipeline(st, sc, ingestCfg)

	if err := pipeline.ProcessAll(ctx); err != nil {
		fmt.Fprintf(r.stderr, "warning: error during initial directory scan: %v\n", err)
	}

	// 4. File watcher
	watchDirs := []string{
		ingestCfg.TasksDir,
		ingestCfg.MilestonesDir,
		ingestCfg.StrategiesDir,
		ingestCfg.GlossaryDir,
	}

	fsw, err := watcher.New(watchDirs, 50*time.Millisecond)
	if err == nil {
		defer func() {
			_ = fsw.Close()
		}()
		fsw.Start(ctx)

		go func() {
			for ev := range fsw.Events() {
				_ = pipeline.HandleEvent(ctx, ev)
			}
		}()
	}

	// 5. MCP Server
	mcpSrv := mcp.New(r.cfg, r.workspaceDir, st, wr)
	transport := &sdk_mcp.IOTransport{
		Reader: io.NopCloser(r.in),
		Writer: nopWriteCloser{Writer: r.out},
	}

	return mcpSrv.MCPServer().Run(ctx, transport)
}
