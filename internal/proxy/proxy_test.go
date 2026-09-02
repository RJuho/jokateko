package proxy_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/RJuho/jokateko/internal/config"
	"github.com/RJuho/jokateko/internal/mcp"
	"github.com/RJuho/jokateko/internal/model"
	"github.com/RJuho/jokateko/internal/proxy"
	"github.com/RJuho/jokateko/internal/server"
	"github.com/RJuho/jokateko/internal/store"
	"github.com/RJuho/jokateko/internal/writer"
	sdk_mcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestProxy_ProbeDaemon(t *testing.T) {
	ctx := t.Context()

	// 1. Inactive daemon
	tempDir := t.TempDir()
	cfg := config.Default(tempDir)
	cfg.Server.Port = 59999 // Unused port

	runner := proxy.NewRunner(cfg, tempDir, nil, nil, nil)
	runner.SetProbeTimeout(50 * time.Millisecond)

	if runner.ProbeDaemon(ctx) {
		t.Error("expected ProbeDaemon to return false for inactive daemon")
	}

	// 2. Active daemon (mocked HTTP server)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/health" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ok"}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	port := ts.Listener.Addr().(*net.TCPAddr).Port
	cfg.Server.Port = port

	runnerActive := proxy.NewRunner(cfg, tempDir, nil, nil, nil)
	runnerActive.SetProbeTimeout(100 * time.Millisecond)

	if !runnerActive.ProbeDaemon(ctx) {
		t.Error("expected ProbeDaemon to return true for active daemon")
	}
}

func TestProxy_StandaloneMode(t *testing.T) {
	dir := t.TempDir()
	tasksDir := filepath.Join(dir, ".jokateko", "tasks")
	_ = os.MkdirAll(tasksDir, 0755)

	// Create sample task file
	sampleTask := `+++
title = "Standalone Task"
status = "backlog"
priority = "medium"
summary = "Testing standalone MCP runner"
tags = ["backend"]
dependencies = []
+++

## Description
Testing standalone runner.
`
	_ = os.WriteFile(filepath.Join(tasksDir, "260901-standalone-task.md"), []byte(sampleTask), 0644)

	cfg := config.Default(dir)
	cfg.Server.Port = 59998 // Ensure daemon probe fails

	clientReader, serverWriter := io.Pipe()
	serverReader, clientWriter := io.Pipe()
	var stderrBuf bytes.Buffer

	runner := proxy.NewRunner(cfg, dir, serverReader, serverWriter, &stderrBuf)
	runner.SetProbeTimeout(50 * time.Millisecond)

	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()

	// Run standalone runner in background
	go func() {
		_ = runner.Run(ctx)
	}()

	// Connect MCP client to clientReader / clientWriter
	transport := &sdk_mcp.IOTransport{
		Reader: clientReader,
		Writer: clientWriter,
	}

	client := sdk_mcp.NewClient(&sdk_mcp.Implementation{Name: "test-client", Version: "1.0.0"}, nil)
	clientSession, err := client.Connect(ctx, transport, nil)
	if err != nil {
		t.Fatalf("failed to connect client to standalone runner: %v", err)
	}
	defer clientSession.Close()

	// Call list_tasks tool
	res, err := clientSession.CallTool(ctx, &sdk_mcp.CallToolParams{
		Name:      "list_tasks",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("CallTool(list_tasks) failed: %v", err)
	}
	if res.IsError {
		t.Fatalf("tool call returned error: %+v", res)
	}
	if len(res.Content) == 0 {
		t.Fatal("expected content in tool result")
	}

	var tasks []mcp.TaskSummary
	tc := res.Content[0].(*sdk_mcp.TextContent)
	if err := json.Unmarshal([]byte(tc.Text), &tasks); err != nil {
		t.Fatalf("failed to unmarshal tasks: %v", err)
	}
	if len(tasks) != 1 || tasks[0].Title != "Standalone Task" {
		t.Errorf("unexpected tasks returned: %+v", tasks)
	}
}

func TestProxy_ProxyMode(t *testing.T) {
	dir := t.TempDir()
	tasksDir := filepath.Join(dir, ".jokateko", "tasks")
	_ = os.MkdirAll(tasksDir, 0755)

	cfg := config.Default(dir)
	cfg.Server.Port = 0 // OS dynamic port

	st, err := store.OpenMemory()
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer st.Close()

	_ = st.UpsertTask(t.Context(), model.Task{
		ID:       "260901-daemon-task",
		Title:    "Daemon Task",
		Status:   "ready",
		Priority: model.PriorityHigh,
		Summary:  "Daemon task summary",
	})

	sc := writer.NewSuppressionCache(time.Second)
	wr := writer.New(sc)
	sse := server.NewSSEHub()
	sse.Start()
	defer sse.Stop()

	// Start daemon HTTP server with MCP handler
	srv := server.New(cfg, dir, st, wr, sse)
	mcpSrv := mcp.New(cfg, dir, st, wr)
	srv.SetMCPHandler(sdk_mcp.NewSSEHandler(func(req *http.Request) *sdk_mcp.Server {
		return mcpSrv.MCPServer()
	}, nil))

	if err := srv.Start(); err != nil {
		t.Fatalf("failed to start daemon server: %v", err)
	}
	defer func() { _ = srv.Shutdown(t.Context()) }()

	// Configure proxy runner with daemon's bound port
	cfg.Server.Port = srv.Port()

	clientReader, serverWriter := io.Pipe()
	serverReader, clientWriter := io.Pipe()
	var stderrBuf bytes.Buffer

	runner := proxy.NewRunner(cfg, dir, serverReader, serverWriter, &stderrBuf)
	runner.SetProbeTimeout(200 * time.Millisecond)

	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()

	// Run proxy in background
	go func() {
		_ = runner.Run(ctx)
	}()

	// Connect MCP client to proxy via clientReader / clientWriter
	transport := &sdk_mcp.IOTransport{
		Reader: clientReader,
		Writer: clientWriter,
	}

	client := sdk_mcp.NewClient(&sdk_mcp.Implementation{Name: "test-client", Version: "1.0.0"}, nil)
	clientSession, err := client.Connect(ctx, transport, nil)
	if err != nil {
		t.Fatalf("failed to connect client to proxy: %v", err)
	}
	defer clientSession.Close()

	// Call list_tasks tool through proxy
	res, err := clientSession.CallTool(ctx, &sdk_mcp.CallToolParams{
		Name:      "list_tasks",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("CallTool(list_tasks) through proxy failed: %v", err)
	}
	if res.IsError {
		t.Fatalf("tool call returned error: %+v", res)
	}
	if len(res.Content) == 0 {
		t.Fatal("expected content in tool result")
	}

	var tasks []mcp.TaskSummary
	tc := res.Content[0].(*sdk_mcp.TextContent)
	if err := json.Unmarshal([]byte(tc.Text), &tasks); err != nil {
		t.Fatalf("failed to unmarshal tasks: %v", err)
	}
	if len(tasks) != 1 || tasks[0].ID != "260901-daemon-task" {
		t.Errorf("unexpected tasks returned from proxy: %+v", tasks)
	}
}

func TestProxy_MultiCycleFailoverAndReconnect(t *testing.T) {
	dir := t.TempDir()
	tasksDir := filepath.Join(dir, ".jokateko", "tasks")
	_ = os.MkdirAll(tasksDir, 0755)

	// Seed disk task for standalone mode
	diskTask := `+++
title = "Disk Task"
status = "ready"
priority = "medium"
summary = "Task residing on disk"
tags = ["backend"]
dependencies = []
+++

## Description
This task is read directly from disk by the standalone engine.
`
	_ = os.WriteFile(filepath.Join(tasksDir, "260901-disk-task.md"), []byte(diskTask), 0644)

	// Pick a dedicated port for daemon cycles
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to allocate test port: %v", err)
	}
	daemonPort := ln.Addr().(*net.TCPAddr).Port
	_ = ln.Close()

	cfg := config.Default(dir)
	cfg.Server.Host = "127.0.0.1"
	cfg.Server.Port = daemonPort

	clientReader, serverWriter := io.Pipe()
	serverReader, clientWriter := io.Pipe()
	var stderrBuf bytes.Buffer

	runner := proxy.NewRunner(cfg, dir, serverReader, serverWriter, &stderrBuf)
	runner.SetProbeTimeout(30 * time.Millisecond)
	runner.SetPollInterval(30 * time.Millisecond)

	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	defer cancel()

	go func() {
		_ = runner.Run(ctx)
	}()

	// Connect client to runner stdio transport (client remains connected across all cycles)
	transport := &sdk_mcp.IOTransport{
		Reader: clientReader,
		Writer: clientWriter,
	}
	client := sdk_mcp.NewClient(&sdk_mcp.Implementation{Name: "multi-cycle-test-client", Version: "1.0.0"}, nil)
	clientSession, err := client.Connect(ctx, transport, nil)
	if err != nil {
		t.Fatalf("client.Connect failed: %v", err)
	}
	defer clientSession.Close()

	callListTasks := func(t *testing.T) []mcp.TaskSummary {
		res, err := clientSession.CallTool(ctx, &sdk_mcp.CallToolParams{
			Name:      "list_tasks",
			Arguments: map[string]any{},
		})
		if err != nil {
			t.Fatalf("CallTool(list_tasks) failed: %v", err)
		}
		if res.IsError {
			t.Fatalf("tool returned error: %+v", res)
		}
		if len(res.Content) == 0 {
			t.Fatal("expected content in tool response")
		}
		var tasks []mcp.TaskSummary
		tc := res.Content[0].(*sdk_mcp.TextContent)
		if err := json.Unmarshal([]byte(tc.Text), &tasks); err != nil {
			t.Fatalf("unmarshal tasks error: %v", err)
		}
		return tasks
	}

	waitFor := func(t *testing.T, timeout time.Duration, cond func() bool, desc string) {
		deadline := time.Now().Add(timeout)
		for time.Now().Before(deadline) {
			if cond() {
				return
			}
			time.Sleep(15 * time.Millisecond)
		}
		t.Fatalf("timed out waiting for: %s", desc)
	}

	startDaemon := func(taskID, taskTitle string) (*server.Server, func()) {
		st, err := store.OpenMemory()
		if err != nil {
			t.Fatalf("open store error: %v", err)
		}
		_ = st.UpsertTask(ctx, model.Task{
			ID:       taskID,
			Title:    taskTitle,
			Status:   "ready",
			Priority: model.PriorityHigh,
			Summary:  "Daemon in-memory task",
		})
		sc := writer.NewSuppressionCache(time.Second)
		wr := writer.New(sc)
		sse := server.NewSSEHub()
		sse.Start()

		dCfg := config.Default(dir)
		dCfg.Server.Host = "127.0.0.1"
		dCfg.Server.Port = daemonPort

		srv := server.New(dCfg, dir, st, wr, sse)
		mcpSrv := mcp.New(dCfg, dir, st, wr)
		srv.SetMCPHandler(sdk_mcp.NewSSEHandler(func(req *http.Request) *sdk_mcp.Server {
			return mcpSrv.MCPServer()
		}, nil))

		if err := srv.Start(); err != nil {
			t.Fatalf("failed to start daemon server on port %d: %v", daemonPort, err)
		}

		cleanup := func() {
			_ = srv.Close()
			_ = st.Close()
		}
		return srv, cleanup
	}

	// Initial State: Daemon is not running -> Runner boots into standalone mode
	waitFor(t, 2*time.Second, func() bool {
		return !runner.IsProxyMode() && runner.LocalEngineRunning()
	}, "initial boot into standalone mode")

	initialTasks := callListTasks(t)
	if len(initialTasks) != 1 || initialTasks[0].ID != "260901-disk-task" {
		t.Fatalf("expected initial task from disk, got: %+v", initialTasks)
	}

	// Execute 3 full start/stop cycles
	for cycle := 1; cycle <= 3; cycle++ {
		t.Logf("=== Beginning Cycle %d: Start Daemon ===", cycle)
		taskID := fmt.Sprintf("daemon-task-%d", cycle)
		taskTitle := fmt.Sprintf("Daemon Task Cycle %d", cycle)

		_, stopDaemon := startDaemon(taskID, taskTitle)

		// Wait for runner to detect daemon and switch to proxy mode (releasing local resources)
		waitFor(t, 2*time.Second, func() bool {
			return runner.IsProxyMode() && !runner.LocalEngineRunning()
		}, fmt.Sprintf("cycle %d: switch to proxy mode", cycle))

		// Verify tool call reaches daemon
		daemonTasks := callListTasks(t)
		if len(daemonTasks) != 1 || daemonTasks[0].ID != taskID {
			t.Fatalf("cycle %d: expected task %q from daemon, got: %+v", cycle, taskID, daemonTasks)
		}

		// Verify local engine was closed
		_, closes, _, _ := runner.Stats()
		if int(closes) < cycle {
			t.Fatalf("cycle %d: expected at least %d local engine closes, got: %d", cycle, cycle, closes)
		}

		t.Logf("=== Cycle %d: Stop Daemon (Triggering Hot-Failover to Standalone) ===", cycle)
		stopDaemon()

		// Allow failover on next tool call or disconnect
		time.Sleep(30 * time.Millisecond)

		// Tool call immediately triggers/verifies standalone fallback
		diskTasks := callListTasks(t)
		if len(diskTasks) != 1 || diskTasks[0].ID != "260901-disk-task" {
			t.Fatalf("cycle %d: expected fallback to disk task, got: %+v", cycle, diskTasks)
		}

		waitFor(t, 2*time.Second, func() bool {
			return !runner.IsProxyMode() && runner.LocalEngineRunning()
		}, fmt.Sprintf("cycle %d: fallback to standalone mode", cycle))

		// Small cooldown before next cycle
		time.Sleep(50 * time.Millisecond)
	}

	// Final verification of counters: at least 3 starts and 3 closes of local engine
	starts, closes, connects, _ := runner.Stats()
	t.Logf("Multi-cycle complete! Starts=%d, Closes=%d, Connects=%d", starts, closes, connects)

	if starts < 4 { // 1 initial + 3 failovers = 4 starts
		t.Errorf("expected at least 4 local engine starts, got %d", starts)
	}
	if closes < 3 { // 3 transitions to proxy = 3 closes
		t.Errorf("expected at least 3 local engine closes, got %d", closes)
	}
}

