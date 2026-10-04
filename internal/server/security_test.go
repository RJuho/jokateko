package server_test

import (
	"bufio"
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/RJuho/jokateko/internal/config"
	internalmcp "github.com/RJuho/jokateko/internal/mcp"
	"github.com/RJuho/jokateko/internal/server"
	"github.com/RJuho/jokateko/internal/service"
	"github.com/RJuho/jokateko/internal/store"
	"github.com/RJuho/jokateko/internal/writer"
	sdk_mcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestCreateRejectsPathTraversalID(t *testing.T) {
	srv, dir, _, _ := setupTestServer(t)

	for _, path := range []string{"/api/tasks", "/api/milestones", "/api/strategies", "/api/glossary"} {
		body := `{"id":"../../escape","title":"Evil","summary":"s"}`
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("POST %s with traversal id: got %d, want 400 (%s)", path, rec.Code, rec.Body.String())
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "escape.md")); err == nil {
		t.Fatal("traversal id wrote a file outside .jokateko")
	}
}

func TestCreateDuplicateIDConflicts(t *testing.T) {
	srv, _, _, _ := setupTestServer(t)

	post := func() int {
		req := httptest.NewRequest(http.MethodPost, "/api/strategies", strings.NewReader(`{"id":"zero-cgo","title":"Zero CGO"}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, req)
		return rec.Code
	}
	if code := post(); code != http.StatusCreated {
		t.Fatalf("first create: got %d", code)
	}
	if code := post(); code != http.StatusConflict {
		t.Fatalf("duplicate create: got %d, want 409", code)
	}
}

func TestCrossOriginMutationRejected(t *testing.T) {
	srv, _, _, _ := setupTestServer(t)

	body := `{"title":"CSRF","summary":"s"}`
	req := httptest.NewRequest(http.MethodPost, "/api/tasks", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	req.Header.Set("Origin", "https://evil.example")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("cross-site POST: got %d, want 403", rec.Code)
	}

	// A configured CORS origin remains trusted.
	req = httptest.NewRequest(http.MethodPost, "/api/tasks", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	req.Header.Set("Origin", "http://localhost:8080")
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("trusted-origin POST: got %d, want 201 (%s)", rec.Code, rec.Body.String())
	}
}

func TestNonJSONContentTypeRejected(t *testing.T) {
	srv, _, _, _ := setupTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "/api/tasks", strings.NewReader(`{"title":"x","summary":"s"}`))
	req.Header.Set("Content-Type", "text/plain")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("text/plain POST: got %d, want 415", rec.Code)
	}
}

func TestOversizedBodyRejected(t *testing.T) {
	srv, _, _, _ := setupTestServer(t)

	big := `{"title":"x","summary":"s","body":"` + strings.Repeat("a", 2<<20) + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/tasks", strings.NewReader(big))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized POST: got %d, want 413", rec.Code)
	}
}

// startLiveServer starts a daemon on a loopback port with the REST API, SSE hub
// and MCP endpoint wired to one shared service, as `jokateko serve` does.
func startLiveServer(t *testing.T) *server.Server {
	t.Helper()
	dir := t.TempDir()
	for _, d := range []string{"tasks", "milestones", "strategies", "glossary"} {
		_ = os.MkdirAll(filepath.Join(dir, ".jokateko", d), 0o755)
	}
	cfg := config.Default(dir)
	cfg.Server.Host = "127.0.0.1"
	cfg.Server.Port = 0

	st, err := store.OpenMemory()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })

	sse := server.NewSSEHub()
	sse.Start()
	svc := service.New(cfg, dir, st, writer.New(writer.NewSuppressionCache(time.Second)), sse)
	srv := server.New(svc, sse)
	srv.SetMCPHandler(internalmcp.New(svc).HTTPHandler())
	if err := srv.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = srv.Close() })
	return srv
}

func TestDNSRebindingHostRejected(t *testing.T) {
	srv := startLiveServer(t)

	for host, want := range map[string]int{
		"evil.example:80":  http.StatusForbidden,
		"localhost":        http.StatusOK,
		srv.Addr():         http.StatusOK,
		"app.localhost:80": http.StatusOK,
	} {
		req, _ := http.NewRequest(http.MethodGet, "http://"+srv.Addr()+"/api/health", nil)
		req.Host = host
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode != want {
			t.Errorf("Host %q: got %d, want %d", host, resp.StatusCode, want)
		}
	}
}

func TestMCPMutationReachesSSE(t *testing.T) {
	srv := startLiveServer(t)
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	// Subscribe to the Web UI event stream.
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+srv.Addr()+"/api/events", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	events := make(chan string, 16)
	go func() {
		sc := bufio.NewScanner(resp.Body)
		for sc.Scan() {
			if name, ok := strings.CutPrefix(sc.Text(), "event: "); ok {
				events <- name
			}
		}
	}()
	waitEvent := func(want string) {
		t.Helper()
		for {
			select {
			case got := <-events:
				if got == want {
					return
				}
			case <-ctx.Done():
				t.Fatalf("timed out waiting for SSE event %q", want)
			}
		}
	}
	waitEvent("connected")

	// Mutate through MCP over Streamable HTTP.
	client := sdk_mcp.NewClient(&sdk_mcp.Implementation{Name: "sse-test", Version: "1"}, nil)
	session, err := client.Connect(ctx, &sdk_mcp.StreamableClientTransport{
		Endpoint:             "http://" + srv.Addr() + "/api/mcp",
		DisableStandaloneSSE: true,
	}, nil)
	if err != nil {
		t.Fatalf("MCP connect: %v", err)
	}
	defer session.Close()

	res, err := session.CallTool(ctx, &sdk_mcp.CallToolParams{
		Name:      "create_task",
		Arguments: map[string]any{"title": "From agent", "summary": "Created via MCP", "body": "- [ ] item"},
	})
	if err != nil || res.IsError {
		t.Fatalf("create_task: %v %+v", err, res)
	}
	waitEvent("task.created")

	var buf bytes.Buffer
	for _, c := range res.Content {
		if tc, ok := c.(*sdk_mcp.TextContent); ok {
			buf.WriteString(tc.Text)
		}
	}
	if !strings.Contains(buf.String(), "from-agent") {
		t.Fatalf("unexpected create_task result: %s", buf.String())
	}
}
