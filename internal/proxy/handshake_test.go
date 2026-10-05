package proxy_test

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
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
	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	sdk_mcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// handshakeWorkspace seeds a workspace with one task on disk, which only the
// standalone engine sees, and reserves a loopback port for a test daemon.
func handshakeWorkspace(t *testing.T) (dir string, cfg *config.Config) {
	t.Helper()
	dir = t.TempDir()
	tasksDir := filepath.Join(dir, ".jokateko", "tasks")
	if err := os.MkdirAll(tasksDir, 0o755); err != nil {
		t.Fatal(err)
	}
	diskTask := "+++\ntitle = \"Disk Task\"\nstatus = \"ready\"\nsummary = \"on disk\"\n+++\n"
	if err := os.WriteFile(filepath.Join(tasksDir, "260901-disk-task.md"), []byte(diskTask), 0o644); err != nil {
		t.Fatal(err)
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	_ = ln.Close()

	cfg = config.Default(dir)
	cfg.Server.Host = "127.0.0.1"
	cfg.Server.Port = port
	return dir, cfg
}

// startTestDaemon serves the workspace on cfg's port with a single in-memory
// task, so responses show which backend answered. The returned func stops it.
func startTestDaemon(t *testing.T, dir string, cfg *config.Config, taskID string) func() {
	t.Helper()
	st, err := store.OpenMemory()
	if err != nil {
		t.Fatal(err)
	}
	if err := st.UpsertTask(t.Context(), model.Task{ID: taskID, Title: taskID, Status: "ready", Summary: "s"}); err != nil {
		t.Fatal(err)
	}
	sse := server.NewSSEHub()
	sse.Start()
	svc := service.New(cfg, dir, st, writer.New(writer.NewSuppressionCache(time.Second)), sse)
	srv := server.New(svc, sse)
	srv.SetMCPHandler(mcp.New(svc).HTTPHandler())
	if err := srv.Start(); err != nil {
		t.Fatal(err)
	}
	var once sync.Once
	stop := func() {
		once.Do(func() {
			_ = srv.Close()
			_ = st.Close()
		})
	}
	t.Cleanup(stop)
	return stop
}

func waitMode(t *testing.T, r *proxy.Runner, wantProxy bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if r.IsProxyMode() == wantProxy && r.LocalEngineRunning() == !wantProxy {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for proxy mode = %v", wantProxy)
}

// recordingTransport records the method of every request the client sends.
type recordingTransport struct {
	sdk_mcp.Transport
	mu      sync.Mutex
	methods []string
}

func (rt *recordingTransport) Connect(ctx context.Context) (sdk_mcp.Connection, error) {
	conn, err := rt.Transport.Connect(ctx)
	if err != nil {
		return nil, err
	}
	return &recordingConn{Connection: conn, rt: rt}, nil
}

func (rt *recordingTransport) Methods() []string {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	return append([]string(nil), rt.methods...)
}

type recordingConn struct {
	sdk_mcp.Connection
	rt *recordingTransport
}

func (c *recordingConn) Write(ctx context.Context, msg jsonrpc.Message) error {
	if req, ok := msg.(*jsonrpc.Request); ok {
		c.rt.mu.Lock()
		c.rt.methods = append(c.rt.methods, req.Method)
		c.rt.mu.Unlock()
	}
	return c.Connection.Write(ctx, msg)
}

// TestProxy_GoSDK18ClientSurvivesSwitches connects a go-sdk 1.8 client, which
// opens with the SEP-2575 server/discover RPC instead of initialize, and checks
// that tools keep working across standalone -> daemon -> standalone switches.
func TestProxy_GoSDK18ClientSurvivesSwitches(t *testing.T) {
	dir, cfg := handshakeWorkspace(t)

	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
	defer cancel()

	// startRawRunner wires the Runner's stdio; the SDK client speaks over it
	// through the same pipes.
	c, r, _, stdout := startRawRunner(t, ctx, cfg, dir, func(r *proxy.Runner) {
		r.SetProbeTimeout(50 * time.Millisecond)
		r.SetPollInterval(30 * time.Millisecond)
	})
	waitMode(t, r, false)

	rt := &recordingTransport{Transport: &sdk_mcp.IOTransport{Reader: stdout, Writer: c.in}}
	client := sdk_mcp.NewClient(&sdk_mcp.Implementation{Name: "sdk18-client", Version: "1"}, nil)
	session, err := client.Connect(ctx, rt, nil)
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer session.Close()

	methods := rt.Methods()
	if len(methods) == 0 || methods[0] != "server/discover" {
		t.Fatalf("client handshake = %v, want server/discover first", methods)
	}
	if strings.Contains(strings.Join(methods, ","), "initialize") {
		t.Fatalf("client fell back to initialize: %v", methods)
	}

	check := func(stage, wantTask string) {
		t.Helper()
		tools, err := session.ListTools(ctx, nil)
		if err != nil || len(tools.Tools) == 0 {
			t.Fatalf("%s: ListTools = %v, %v", stage, tools, err)
		}
		res, err := session.CallTool(ctx, &sdk_mcp.CallToolParams{Name: "list_tasks", Arguments: map[string]any{}})
		if err != nil || res.IsError {
			t.Fatalf("%s: CallTool(list_tasks) = %+v, %v", stage, res, err)
		}
		b, _ := json.Marshal(res.Content)
		if !strings.Contains(string(b), wantTask) {
			t.Fatalf("%s: list_tasks = %s, want %q", stage, b, wantTask)
		}
	}

	check("initial standalone", "260901-disk-task")

	stop := startTestDaemon(t, dir, cfg, "daemon-task")
	waitMode(t, r, true)
	check("after standalone -> daemon", "daemon-task")

	stop()
	check("after daemon -> standalone", "260901-disk-task")
	waitMode(t, r, false)
}

// expectResponse reads the next response, skipping notifications, and fails if
// it is not the response to id. It catches replay responses leaking to the
// client, which waitID would silently skip.
func (c *rawClient) expectResponse(id int) map[string]any {
	c.t.Helper()
	for {
		line, err := c.out.ReadString('\n')
		if err != nil {
			c.t.Fatalf("read response %d: %v", id, err)
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			c.t.Fatalf("decode %q: %v", line, err)
		}
		rawID, ok := m["id"]
		if !ok {
			continue
		}
		if v, ok := rawID.(float64); !ok || int(v) != id {
			c.t.Fatalf("got response for id %v, want %d: %s", rawID, id, line)
		}
		return m
	}
}

// TestProxy_LegacyInitializeSurvivesSwitches drives an initialize-only client
// through standalone -> daemon -> standalone. The return to standalone needs
// the replay; its response must not reach the client.
func TestProxy_LegacyInitializeSurvivesSwitches(t *testing.T) {
	dir, cfg := handshakeWorkspace(t)

	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
	defer cancel()
	c, r, done, _ := startRawRunner(t, ctx, cfg, dir, func(r *proxy.Runner) {
		r.SetProbeTimeout(50 * time.Millisecond)
		r.SetPollInterval(30 * time.Millisecond)
	})
	waitMode(t, r, false)

	c.send(initializeReq)
	if resp := c.expectResponse(1); resp["result"] == nil {
		t.Fatalf("initialize failed: %v", resp)
	}
	c.send(initializedNt)

	call := func(id int, want string) {
		t.Helper()
		c.send(listTasksReq(id))
		b, _ := json.Marshal(c.expectResponse(id))
		if !strings.Contains(string(b), want) {
			t.Fatalf("response %d does not contain %q: %s", id, want, b)
		}
	}

	call(2, "260901-disk-task")

	stop := startTestDaemon(t, dir, cfg, "daemon-task")
	waitMode(t, r, true)
	call(3, "daemon-task")

	stop()
	call(4, "260901-disk-task")
	waitMode(t, r, false)
	call(5, "260901-disk-task")

	_ = c.in.Close()
	if err := waitRun(t, done); err != nil {
		t.Fatalf("Run: %v", err)
	}
}
