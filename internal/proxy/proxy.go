// Package proxy implements the Model Context Protocol (MCP) stdio-to-HTTP proxy
// and standalone in-process fallback runner with hot-failover and auto-reconnect.
package proxy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/RJuho/jokateko/internal/config"
	"github.com/RJuho/jokateko/internal/mcp"
	"github.com/RJuho/jokateko/internal/service"
	"github.com/RJuho/jokateko/internal/store"
	"github.com/RJuho/jokateko/internal/watcher"
	"github.com/RJuho/jokateko/internal/writer"
	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	sdk_mcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

type localEngine struct {
	store        *store.Store
	stopPipeline func()
	cancel       context.CancelFunc
	mcpConn      sdk_mcp.Connection
	pipeClosers  []io.Closer
	closed       atomic.Bool
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
	if le.stopPipeline != nil {
		le.stopPipeline()
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

	// switchMu serializes backend switches. A switch requested while another
	// is in flight waits for it, then re-checks the state it finds.
	switchMu sync.Mutex

	mu            sync.RWMutex
	isProxy       bool
	activeBackend sdk_mcp.Connection
	localEngine   *localEngine
	switching     bool

	cachedInitMsg   *jsonrpc.Request
	cachedInitNotif *jsonrpc.Request
	replaySeq       atomic.Int64

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
			r.forward(runCtx, clientConn, msg)
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
				current := r.activeBackend
				r.mu.RUnlock()

				if switching {
					time.Sleep(5 * time.Millisecond)
					continue
				}
				// The read failed because a switch closed the previous backend;
				// the new backend is healthy, so just start reading from it.
				if current != backend {
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

// daemonBaseURL returns the base URL of the serve daemon for this workspace.
// Only loopback hosts are probed; anything else falls back to 127.0.0.1.
func (r *Runner) daemonBaseURL() string {
	port := r.cfg.Server.Port
	if port <= 0 {
		port = 8080
	}
	host := "127.0.0.1"
	if h := r.cfg.Server.Host; h == "localhost" {
		host = h
	} else if ip := net.ParseIP(h); ip != nil && ip.IsLoopback() {
		host = h
	}
	return "http://" + net.JoinHostPort(host, strconv.Itoa(port))
}

// probeDaemon reports whether a healthy serve daemon for this same workspace is listening.
// A daemon serving a different project on the same port is ignored.
func (r *Runner) probeDaemon(ctx context.Context) bool {
	timeout := r.probeTimeout
	if timeout <= 0 {
		timeout = 150 * time.Millisecond
	}

	probeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(probeCtx, http.MethodGet, r.daemonBaseURL()+"/api/health", nil)
	if err != nil {
		return false
	}

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false
	}

	var health struct {
		Workspace string `json:"workspace"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&health); err != nil {
		return false
	}
	return sameDir(health.Workspace, r.workspaceDir)
}

// sameDir reports whether a and b name the same directory.
func sameDir(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	if filepath.Clean(a) == filepath.Clean(b) {
		return true
	}
	ai, errA := os.Stat(a)
	bi, errB := os.Stat(b)
	return errA == nil && errB == nil && os.SameFile(ai, bi)
}

func (r *Runner) connectDaemon(ctx context.Context) (sdk_mcp.Connection, error) {
	transport := &sdk_mcp.StreamableClientTransport{
		Endpoint: r.daemonBaseURL() + "/api/mcp",
		// The daemon runs a stateless server, which has no standalone SSE stream.
		DisableStandaloneSSE: true,
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
	dirs := r.cfg.ResolveDirs(r.workspaceDir)

	localCtx, localCancel := context.WithCancel(ctx)
	logf := func(format string, args ...any) {
		fmt.Fprintf(r.stderr, format+"\n", args...)
	}
	stopPipeline, err := watcher.StartPipeline(localCtx, st, sc, dirs, nil, logf)
	if err != nil {
		logf("warning: %v", err)
	}

	// The standalone engine has no Web UI, so changes need no notifier.
	mcpSrv := mcp.New(service.New(r.cfg, r.workspaceDir, st, wr, nil))

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
		stopPipeline()
		_ = st.Close()
		_ = serverIn.Close()
		_ = serverOut.Close()
		_ = clientIn.Close()
		_ = clientOut.Close()
		return nil, nil, fmt.Errorf("failed to connect to local MCP engine: %w", err)
	}

	engine := &localEngine{
		store:        st,
		stopPipeline: stopPipeline,
		cancel:       localCancel,
		mcpConn:      localConn,
		pipeClosers:  []io.Closer{serverIn, serverOut, clientIn, clientOut},
	}

	return engine, localConn, nil
}

// replayTimeout bounds how long a handshake replay waits for the backend's
// initialize response.
const replayTimeout = 5 * time.Second

// replayIDPrefix marks request IDs the proxy issues itself. Clients use
// numbers or their own string prefixes, so these never clash with an
// in-flight client request.
const replayIDPrefix = "jokateko-proxy-replay-"

// replayHandshake brings a fresh stateful backend to the state the client's
// session is in by replaying the cached legacy initialize handshake. The
// initialize request is sent under a proxy-private ID and its response is
// consumed here, never forwarded to the client. SEP-2575 clients (which open
// with server/discover) cache nothing, so this is a no-op for them: every
// request they send carries its own protocol _meta.
func (r *Runner) replayHandshake(ctx context.Context, conn sdk_mcp.Connection) error {
	r.mu.RLock()
	initMsg := r.cachedInitMsg
	initNotif := r.cachedInitNotif
	r.mu.RUnlock()

	if initMsg == nil {
		return nil
	}

	id, err := jsonrpc.MakeID(replayIDPrefix + strconv.FormatInt(r.replaySeq.Add(1), 10))
	if err != nil {
		return fmt.Errorf("replay initialize failed: %w", err)
	}
	replay := *initMsg
	replay.ID = id

	ctx, cancel := context.WithTimeout(ctx, replayTimeout)
	defer cancel()

	if err := conn.Write(ctx, &replay); err != nil {
		return fmt.Errorf("replay initialize failed: %w", err)
	}

	// Skip anything the backend sends before the matching response, such as
	// log notifications; none of it belongs to the client.
	for {
		msg, err := conn.Read(ctx)
		if err != nil {
			return fmt.Errorf("read replay initialize response failed: %w", err)
		}
		resp, ok := msg.(*jsonrpc.Response)
		if !ok || resp.ID != id {
			continue
		}
		if resp.Error != nil {
			return fmt.Errorf("replay initialize rejected: %w", resp.Error)
		}
		break
	}

	if initNotif != nil {
		if err := conn.Write(ctx, initNotif); err != nil {
			return fmt.Errorf("replay initialized notification failed: %w", err)
		}
	}

	return nil
}

// inspectClientMessage caches the legacy initialize handshake for replay.
// server/discover is deliberately not cached: a fresh backend does not need
// it, because SEP-2575 clients repeat their protocol version and capabilities
// in the _meta of every request.
func (r *Runner) inspectClientMessage(msg jsonrpc.Message) {
	req, ok := msg.(*jsonrpc.Request)
	if !ok {
		return
	}

	switch req.Method {
	case "initialize":
		if !req.IsCall() {
			return
		}
		r.mu.Lock()
		r.cachedInitMsg = req
		r.mu.Unlock()
	case "notifications/initialized":
		r.mu.Lock()
		r.cachedInitNotif = req
		r.mu.Unlock()
	}
}

// forward delivers a client message to the active backend. When a write to
// the daemon fails, it fails over to the standalone engine and retries there.
// A call that still cannot be delivered gets an error response, so the client
// never waits for a reply that will not come.
func (r *Runner) forward(ctx context.Context, client sdk_mcp.Connection, msg jsonrpc.Message) {
	r.mu.RLock()
	backend := r.activeBackend
	isProxy := r.isProxy
	r.mu.RUnlock()

	err := errors.New("no active backend")
	if backend != nil {
		err = backend.Write(ctx, msg)
	}
	if err != nil && isProxy {
		if err = r.switchToStandalone(ctx); err == nil {
			r.mu.RLock()
			backend = r.activeBackend
			r.mu.RUnlock()
			err = backend.Write(ctx, msg)
		}
	}
	if err == nil {
		return
	}

	req, ok := msg.(*jsonrpc.Request)
	if !ok || !req.IsCall() {
		return
	}
	fmt.Fprintf(r.stderr, "jokateko: cannot deliver %q: %v\n", req.Method, err)
	_ = client.Write(ctx, &jsonrpc.Response{
		ID: req.ID,
		Error: &jsonrpc.Error{
			Code:    jsonrpc.CodeInternalError,
			Message: fmt.Sprintf("jokateko proxy: backend unavailable: %v", err),
		},
	})
}

func (r *Runner) switchToProxy(ctx context.Context) error {
	r.switchMu.Lock()
	defer r.switchMu.Unlock()

	r.mu.Lock()
	if r.isProxy {
		r.mu.Unlock()
		return nil
	}
	r.switching = true
	r.mu.Unlock()

	defer func() {
		r.mu.Lock()
		r.switching = false
		r.mu.Unlock()
	}()

	// No handshake replay: the daemon serves stateless Streamable HTTP, which
	// gives every request a fresh, already-initialized session.
	daemonConn, err := r.connectDaemon(ctx)
	if err != nil {
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
	r.switchMu.Lock()
	defer r.switchMu.Unlock()

	r.mu.Lock()
	if !r.isProxy && r.localEngine != nil {
		r.mu.Unlock()
		return nil
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
