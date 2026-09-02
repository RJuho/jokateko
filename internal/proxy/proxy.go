// Package proxy implements the Model Context Protocol (MCP) stdio-to-HTTP proxy
// and standalone in-process fallback runner with hot-failover and auto-reconnect.
package proxy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/RJuho/jokateko/internal/config"
	"github.com/RJuho/jokateko/internal/mcp"
	"github.com/RJuho/jokateko/internal/store"
	"github.com/RJuho/jokateko/internal/watcher"
	"github.com/RJuho/jokateko/internal/writer"
	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	sdk_mcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

type localEngine struct {
	store       *store.Store
	watcher     *watcher.Watcher
	cancel      context.CancelFunc
	mcpConn     sdk_mcp.Connection
	pipeClosers []io.Closer
	closed      atomic.Bool
}

func (le *localEngine) Close() {
	if le == nil || le.closed.Swap(true) {
		return
	}
	if le.mcpConn != nil {
		_ = le.mcpConn.Close()
	}
	for _, c := range le.pipeClosers {
		_ = c.Close()
	}
	if le.cancel != nil {
		le.cancel()
	}
	if le.watcher != nil {
		_ = le.watcher.Close()
	}
	if le.store != nil {
		_ = le.store.Close()
	}
}

// Runner manages the lifecycle of the stdio MCP bridge or standalone runner.
type Runner struct {
	cfg          *config.Config
	workspaceDir string
	in           io.Reader
	out          io.Writer
	stderr       io.Writer
	probeTimeout time.Duration
	pollInterval time.Duration

	mu            sync.RWMutex
	isProxy       bool
	activeBackend sdk_mcp.Connection
	localEngine   *localEngine
	switching     bool

	cachedInitMsg   jsonrpc.Message
	cachedInitNotif jsonrpc.Message

	localStarts      atomic.Int64
	localCloses      atomic.Int64
	proxyConnects    atomic.Int64
	proxyDisconnects atomic.Int64
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
		pollInterval: 5 * time.Second,
	}
}

// SetProbeTimeout sets the timeout duration for probing daemon health.
func (r *Runner) SetProbeTimeout(d time.Duration) {
	r.probeTimeout = d
}

// SetPollInterval sets the background probe interval when checking for daemon availability.
func (r *Runner) SetPollInterval(d time.Duration) {
	r.pollInterval = d
}

// IsProxyMode returns true if the runner is currently proxying to an external daemon.
func (r *Runner) IsProxyMode() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.isProxy
}

// LocalEngineRunning returns true if the local in-memory standalone engine is active.
func (r *Runner) LocalEngineRunning() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.localEngine != nil && !r.isProxy
}

// Stats returns the number of local engine starts, closes, proxy connects, and disconnects.
func (r *Runner) Stats() (starts, closes, connects, disconnects int64) {
	return r.localStarts.Load(), r.localCloses.Load(), r.proxyConnects.Load(), r.proxyDisconnects.Load()
}

// Run executes the MCP proxy workflow:
// 1. Probes daemon health at GET http://127.0.0.1:<port>/api/health
// 2. Bridges stdio to daemon if active, or runs in-process standalone engine if inactive
// 3. Dynamically hot-fails over if daemon disconnects and auto-reconnects when daemon respawns
func (r *Runner) Run(ctx context.Context) error {
	// Initial backend selection
	if r.probeDaemon(ctx) {
		if err := r.switchToProxy(ctx); err != nil {
			if err := r.switchToStandalone(ctx); err != nil {
				return fmt.Errorf("failed to initialize standalone engine: %w", err)
			}
		}
	} else {
		if err := r.switchToStandalone(ctx); err != nil {
			return fmt.Errorf("failed to initialize standalone engine: %w", err)
		}
	}

	defer func() {
		r.mu.Lock()
		if r.activeBackend != nil {
			_ = r.activeBackend.Close()
		}
		if r.localEngine != nil {
			r.localEngine.Close()
		}
		r.mu.Unlock()
	}()

	// Stdio connection to client (AI agent)
	ioTransport := &sdk_mcp.IOTransport{
		Reader: io.NopCloser(r.in),
		Writer: nopWriteCloser{Writer: r.out},
	}
	clientConn, err := ioTransport.Connect(ctx)
	if err != nil {
		return fmt.Errorf("failed to initialize stdio transport: %w", err)
	}
	defer clientConn.Close()

	runCtx, runCancel := context.WithCancel(ctx)
	defer runCancel()

	// Background daemon probe loop for auto-reconnect
	pollInterval := r.pollInterval
	if pollInterval <= 0 {
		pollInterval = 5 * time.Second
	}

	go func() {
		ticker := time.NewTicker(pollInterval)
		defer ticker.Stop()

		for {
			select {
			case <-runCtx.Done():
				return
			case <-ticker.C:
				r.mu.RLock()
				isProxy := r.isProxy
				r.mu.RUnlock()

				if isProxy {
					continue
				}

				if r.probeDaemon(runCtx) {
					_ = r.switchToProxy(runCtx)
				}
			}
		}
	}()

	errCh := make(chan error, 2)

	// Pump Client -> Backend
	go func() {
		for {
			msg, err := clientConn.Read(runCtx)
			if err != nil {
				errCh <- err
				return
			}

			r.inspectClientMessage(msg)

			r.mu.RLock()
			backend := r.activeBackend
			isProxy := r.isProxy
			r.mu.RUnlock()

			if backend == nil {
				continue
			}

			if err := backend.Write(runCtx, msg); err != nil {
				if isProxy {
					if switchErr := r.switchToStandalone(runCtx); switchErr == nil {
						r.mu.RLock()
						newBackend := r.activeBackend
						r.mu.RUnlock()
						if newBackend != nil {
							_ = newBackend.Write(runCtx, msg)
						}
					}
				}
			}
		}
	}()

	// Pump Backend -> Client
	go func() {
		for {
			select {
			case <-runCtx.Done():
				return
			default:
			}

			r.mu.RLock()
			backend := r.activeBackend
			r.mu.RUnlock()

			if backend == nil {
				select {
				case <-runCtx.Done():
					return
				case <-time.After(10 * time.Millisecond):
					continue
				}
			}

			msg, err := backend.Read(runCtx)
			if err != nil {
				if runCtx.Err() != nil {
					return
				}

				r.mu.RLock()
				switching := r.switching
				isProxy := r.isProxy
				r.mu.RUnlock()

				if switching {
					time.Sleep(5 * time.Millisecond)
					continue
				}

				if isProxy {
					_ = r.switchToStandalone(runCtx)
					continue
				}

				time.Sleep(10 * time.Millisecond)
				continue
			}

			if err := clientConn.Write(runCtx, msg); err != nil {
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

func (r *Runner) connectDaemon(ctx context.Context) (sdk_mcp.Connection, error) {
	port := r.cfg.Server.Port
	if port <= 0 {
		port = 8080
	}
	mcpEndpoint := fmt.Sprintf("http://127.0.0.1:%d/api/mcp", port)
	transport := &sdk_mcp.SSEClientTransport{
		Endpoint: mcpEndpoint,
	}
	return transport.Connect(ctx)
}

func (r *Runner) startLocalEngine(ctx context.Context) (*localEngine, sdk_mcp.Connection, error) {
	st, err := store.OpenMemory()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize in-memory store: %w", err)
	}

	sc := writer.NewSuppressionCache(time.Second)
	wr := writer.New(sc)

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

	localCtx, localCancel := context.WithCancel(ctx)
	watchDirs := []string{
		ingestCfg.TasksDir,
		ingestCfg.MilestonesDir,
		ingestCfg.StrategiesDir,
		ingestCfg.GlossaryDir,
	}

	var fsw *watcher.Watcher
	w, err := watcher.New(watchDirs, 50*time.Millisecond)
	if err == nil {
		fsw = w
		fsw.Start(localCtx)
		go func() {
			for ev := range fsw.Events() {
				_ = pipeline.HandleEvent(localCtx, ev)
			}
		}()
	}

	mcpSrv := mcp.New(r.cfg, r.workspaceDir, st, wr)

	serverIn, clientOut := io.Pipe()
	clientIn, serverOut := io.Pipe()

	localServerTransport := &sdk_mcp.IOTransport{
		Reader: serverIn,
		Writer: serverOut,
	}
	go func() {
		_ = mcpSrv.MCPServer().Run(localCtx, localServerTransport)
	}()

	localConnTransport := &sdk_mcp.IOTransport{
		Reader: clientIn,
		Writer: clientOut,
	}
	localConn, err := localConnTransport.Connect(localCtx)
	if err != nil {
		localCancel()
		if fsw != nil {
			_ = fsw.Close()
		}
		_ = st.Close()
		_ = serverIn.Close()
		_ = serverOut.Close()
		_ = clientIn.Close()
		_ = clientOut.Close()
		return nil, nil, fmt.Errorf("failed to connect to local MCP engine: %w", err)
	}

	engine := &localEngine{
		store:       st,
		watcher:     fsw,
		cancel:      localCancel,
		mcpConn:     localConn,
		pipeClosers: []io.Closer{serverIn, serverOut, clientIn, clientOut},
	}

	return engine, localConn, nil
}

func (r *Runner) replayHandshake(ctx context.Context, conn sdk_mcp.Connection) error {
	r.mu.RLock()
	initMsg := r.cachedInitMsg
	initNotif := r.cachedInitNotif
	r.mu.RUnlock()

	if initMsg == nil {
		return nil
	}

	if err := conn.Write(ctx, initMsg); err != nil {
		return fmt.Errorf("replay initialize failed: %w", err)
	}

	resp, err := conn.Read(ctx)
	if err != nil {
		return fmt.Errorf("read replay initialize response failed: %w", err)
	}
	_ = resp

	if initNotif != nil {
		if err := conn.Write(ctx, initNotif); err != nil {
			return fmt.Errorf("replay initialized notification failed: %w", err)
		}
	}

	return nil
}

type rpcMethodDetector struct {
	Method string `json:"method"`
}

func (r *Runner) inspectClientMessage(msg jsonrpc.Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	var d rpcMethodDetector
	if err := json.Unmarshal(data, &d); err != nil {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if d.Method == "initialize" {
		r.cachedInitMsg = msg
	} else if d.Method == "notifications/initialized" {
		r.cachedInitNotif = msg
	}
}

func (r *Runner) switchToProxy(ctx context.Context) error {
	r.mu.Lock()
	if r.isProxy {
		r.mu.Unlock()
		return nil
	}
	if r.switching {
		r.mu.Unlock()
		for i := 0; i < 50; i++ {
			time.Sleep(10 * time.Millisecond)
			r.mu.RLock()
			isProxy := r.isProxy
			switching := r.switching
			r.mu.RUnlock()
			if !switching && isProxy {
				return nil
			}
		}
		return errors.New("timeout waiting for concurrent switch")
	}
	r.switching = true
	r.mu.Unlock()

	defer func() {
		r.mu.Lock()
		r.switching = false
		r.mu.Unlock()
	}()

	daemonConn, err := r.connectDaemon(ctx)
	if err != nil {
		return err
	}

	if err := r.replayHandshake(ctx, daemonConn); err != nil {
		_ = daemonConn.Close()
		return err
	}

	r.mu.Lock()
	oldConn := r.activeBackend
	oldEngine := r.localEngine

	r.activeBackend = daemonConn
	r.isProxy = true
	r.localEngine = nil
	r.proxyConnects.Add(1)
	r.mu.Unlock()

	if oldConn != nil {
		_ = oldConn.Close()
	}
	if oldEngine != nil {
		oldEngine.Close()
		r.localCloses.Add(1)
	}

	port := r.cfg.Server.Port
	if port <= 0 {
		port = 8080
	}
	fmt.Fprintf(r.stderr, "jokateko: connected to running daemon on port %d (proxy mode)\n", port)
	return nil
}

func (r *Runner) switchToStandalone(ctx context.Context) error {
	r.mu.Lock()
	if !r.isProxy && r.localEngine != nil {
		r.mu.Unlock()
		return nil
	}
	if r.switching {
		r.mu.Unlock()
		for i := 0; i < 50; i++ {
			time.Sleep(10 * time.Millisecond)
			r.mu.RLock()
			isStandalone := !r.isProxy && r.localEngine != nil
			switching := r.switching
			r.mu.RUnlock()
			if !switching && isStandalone {
				return nil
			}
		}
		return errors.New("timeout waiting for concurrent switch")
	}
	r.switching = true
	r.mu.Unlock()

	defer func() {
		r.mu.Lock()
		r.switching = false
		r.mu.Unlock()
	}()

	engine, localConn, err := r.startLocalEngine(ctx)
	if err != nil {
		return err
	}

	if err := r.replayHandshake(ctx, localConn); err != nil {
		engine.Close()
		_ = localConn.Close()
		return err
	}

	r.mu.Lock()
	oldConn := r.activeBackend

	r.activeBackend = localConn
	r.isProxy = false
	r.localEngine = engine
	r.localStarts.Add(1)
	if oldConn != nil {
		r.proxyDisconnects.Add(1)
	}
	r.mu.Unlock()

	if oldConn != nil {
		_ = oldConn.Close()
	}

	fmt.Fprintf(r.stderr, "jokateko: daemon inactive, running in standalone in-process mode\n")
	return nil
}
