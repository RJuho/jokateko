package proxy_test

import (
	"bytes"
	"context"
	"encoding/json"
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
