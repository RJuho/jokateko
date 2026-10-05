package proxy_test

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/RJuho/jokateko/internal/config"
	"github.com/RJuho/jokateko/internal/mcp"
	"github.com/RJuho/jokateko/internal/model"
	"github.com/RJuho/jokateko/internal/proxy"
	"github.com/RJuho/jokateko/internal/server"
	"github.com/RJuho/jokateko/internal/service"
	"github.com/RJuho/jokateko/internal/store"
	"github.com/RJuho/jokateko/internal/writer"
)

func TestProxy_ProbeDaemonRejectsBadHealth(t *testing.T) {
	dir := t.TempDir()
	tests := []struct {
		name    string
		handler http.HandlerFunc
	}{
		{"non-200", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusServiceUnavailable) }},
		{"invalid json", func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, "not json") }},
		{"missing workspace", func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, `{"status":"ok"}`) }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := httptest.NewServer(tc.handler)
			defer ts.Close()
			cfg := config.Default(dir)
			cfg.Server.Host = "127.0.0.1"
			cfg.Server.Port = ts.Listener.Addr().(*net.TCPAddr).Port
			r := proxy.NewRunner(cfg, dir, nil, nil, io.Discard)
			r.SetProbeTimeout(0) // falls back to the default timeout
			if r.ProbeDaemon(t.Context()) {
				t.Error("expected probe to fail")
			}
		})
	}
}

// rawClient speaks newline-delimited JSON-RPC to a Runner's stdio.
type rawClient struct {
	t   *testing.T
	in  *io.PipeWriter
	out *bufio.Reader
}

func (c *rawClient) send(msg string) {
	c.t.Helper()
	if _, err := io.WriteString(c.in, msg+"\n"); err != nil {
		c.t.Fatalf("write %s: %v", msg, err)
	}
}

// readTimeout bounds each read, so a lost response fails the test instead of
// hanging it.
const readTimeout = 5 * time.Second

// readLine reads the next message line while waiting for response id. On
// timeout the reader goroutine stays blocked until cleanup closes the pipe.
func (c *rawClient) readLine(id int) string {
	c.t.Helper()
	type result struct {
		line string
		err  error
	}
	ch := make(chan result, 1)
	go func() {
		line, err := c.out.ReadString('\n')
		ch <- result{line, err}
	}()
	select {
	case res := <-ch:
		if res.err != nil {
			c.t.Fatalf("read response %d: %v", id, res.err)
		}
		return res.line
	case <-time.After(readTimeout):
		c.t.Fatalf("timed out after %v waiting for response %d", readTimeout, id)
		return ""
	}
}

// waitID reads messages until the response with the given id arrives.
func (c *rawClient) waitID(id int) map[string]any {
	c.t.Helper()
	for {
		line := c.readLine(id)
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			c.t.Fatalf("decode %q: %v", line, err)
		}
		if v, ok := m["id"].(float64); ok && int(v) == id {
			return m
		}
	}
}

func startRawRunner(t *testing.T, ctx context.Context, cfg *config.Config, dir string, configure func(*proxy.Runner)) (*rawClient, *proxy.Runner, <-chan error, *io.PipeReader) {
	t.Helper()
	stdinR, stdinW := io.Pipe()
	stdoutR, stdoutW := io.Pipe()
	r := proxy.NewRunner(cfg, dir, stdinR, stdoutW, io.Discard)
	r.SetProbeTimeout(200 * time.Millisecond)
	if configure != nil {
		configure(r)
	}
	done := make(chan error, 1)
	go func() { done <- r.Run(ctx) }()
	t.Cleanup(func() {
		_ = stdinW.Close()
		_ = stdoutR.Close()
	})
	return &rawClient{t: t, in: stdinW, out: bufio.NewReader(stdoutR)}, r, done, stdoutR
}

func waitRun(t *testing.T, done <-chan error) error {
	t.Helper()
	select {
	case err := <-done:
		return err
	case <-time.After(10 * time.Second):
		t.Fatal("Run did not return")
		return nil
	}
}

const (
	initializeReq = `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"raw","version":"1"}}}`
	initializedNt = `{"jsonrpc":"2.0","method":"notifications/initialized"}`
)

func listTasksReq(id int) string {
	return `{"jsonrpc":"2.0","id":` + strconv.Itoa(id) + `,"method":"tools/call","params":{"name":"list_tasks","arguments":{}}}`
}

func TestProxy_RunReturnsNilOnClientEOF(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Default(dir)
	cfg.Server.Port = 59996
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	c, r, done, _ := startRawRunner(t, ctx, cfg, dir, func(r *proxy.Runner) { r.SetPollInterval(0) })
	c.send(initializeReq)
	if resp := c.waitID(1); resp["result"] == nil {
		t.Fatalf("initialize failed: %v", resp)
	}
	if !r.LocalEngineRunning() {
		t.Error("expected standalone engine")
	}
	_ = c.in.Close()
	if err := waitRun(t, done); err != nil {
		t.Fatalf("Run returned %v, want nil on client EOF", err)
	}
}

func TestProxy_RunReturnsErrorWhenStdoutCloses(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Default(dir)
	cfg.Server.Port = 59995
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	c, _, done, stdoutR := startRawRunner(t, ctx, cfg, dir, nil)
	_ = stdoutR.Close()
	c.send(initializeReq)
	if err := waitRun(t, done); err == nil {
		t.Fatal("expected Run to fail when stdout is closed")
	}
}

func TestProxy_RunReturnsContextError(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Default(dir)
	cfg.Server.Port = 59994
	ctx, cancel := context.WithCancel(t.Context())

	c, _, done, _ := startRawRunner(t, ctx, cfg, dir, nil)
	c.send(initializeReq)
	c.waitID(1)
	cancel()
	if err := waitRun(t, done); err == nil {
		t.Fatal("expected context error")
	}
}

// TestProxy_FailoverReplaysHandshake covers switchToStandalone when the daemon
// dies while proxying: the cached initialize handshake is replayed to the new
// local engine and the in-flight request is answered from disk.
func TestProxy_FailoverReplaysHandshake(t *testing.T) {
	dir := t.TempDir()
	tasksDir := filepath.Join(dir, ".jokateko", "tasks")
	if err := os.MkdirAll(tasksDir, 0o755); err != nil {
		t.Fatal(err)
	}
	diskTask := "+++\ntitle = \"Disk Task\"\nstatus = \"ready\"\nsummary = \"on disk\"\n+++\n"
	if err := os.WriteFile(filepath.Join(tasksDir, "260901-disk-task.md"), []byte(diskTask), 0o644); err != nil {
		t.Fatal(err)
	}

	// Daemon serving the same workspace with an in-memory-only task.
	dCfg := config.Default(dir)
	dCfg.Server.Host = "127.0.0.1"
	dCfg.Server.Port = 0
	st, err := store.OpenMemory()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = st.Close() }()
	if err := st.UpsertTask(t.Context(), model.Task{ID: "daemon-task", Title: "Daemon Task", Status: "ready", Summary: "s"}); err != nil {
		t.Fatal(err)
	}
	sse := server.NewSSEHub()
	sse.Start()
	svc := service.New(dCfg, dir, st, writer.New(writer.NewSuppressionCache(time.Second)), sse)
	srv := server.New(svc, sse)
	srv.SetMCPHandler(mcp.New(svc).HTTPHandler())
	if err := srv.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = srv.Close() }()

	cfg := config.Default(dir)
	cfg.Server.Host = "127.0.0.1"
	cfg.Server.Port = srv.Port()

	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	defer cancel()
	c, r, done, _ := startRawRunner(t, ctx, cfg, dir, func(r *proxy.Runner) { r.SetPollInterval(time.Hour) })

	c.send(initializeReq)
	if resp := c.waitID(1); resp["result"] == nil {
		t.Fatalf("initialize via daemon failed: %v", resp)
	}
	c.send(initializedNt)
	if !r.IsProxyMode() {
		t.Fatal("expected proxy mode")
	}
	callListTasks(t, c, 3, "daemon-task")

	// Kill the daemon; the next request must fail over to the local engine.
	_ = srv.Close()
	callListTasks(t, c, 4, "260901-disk-task")

	if r.IsProxyMode() || !r.LocalEngineRunning() {
		t.Error("expected standalone mode after failover")
	}
	_, _, connects, disconnects := r.Stats()
	if connects != 1 || disconnects != 1 {
		t.Errorf("connects=%d disconnects=%d, want 1/1", connects, disconnects)
	}

	_ = c.in.Close()
	if err := waitRun(t, done); err != nil {
		t.Fatalf("Run: %v", err)
	}
}

func callListTasks(t *testing.T, c *rawClient, id int, want string) {
	t.Helper()
	c.send(listTasksReq(id))
	resp := c.waitID(id)
	b, _ := json.Marshal(resp)
	if !strings.Contains(string(b), want) {
		t.Fatalf("response %d does not contain %q: %s", id, want, b)
	}
}
