package server_test

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/RJuho/jokateko/internal/config"
	"github.com/RJuho/jokateko/internal/model"
	"github.com/RJuho/jokateko/internal/server"
	"github.com/RJuho/jokateko/internal/service"
	"github.com/RJuho/jokateko/internal/store"
	"github.com/RJuho/jokateko/internal/writer"
)

// newConfiguredServer is setupTestServer with a hook to adjust the config
// before the server is built.
func newConfiguredServer(t *testing.T, mutate func(*config.Config)) (*server.Server, *store.Store) {
	t.Helper()
	dir := t.TempDir()
	for _, d := range []string{"tasks", "milestones", "strategies", "glossary"} {
		if err := os.MkdirAll(filepath.Join(dir, ".jokateko", d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	cfg := config.Default(dir)
	cfg.Server.Host = "127.0.0.1"
	cfg.Server.Port = 0
	if mutate != nil {
		mutate(cfg)
	}
	st, err := store.OpenMemory()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	sse := server.NewSSEHub()
	sse.Start()
	t.Cleanup(sse.Stop)
	return server.New(service.New(cfg, dir, st, writer.New(writer.NewSuppressionCache(time.Second)), sse), sse), st
}

// do sends a request through the server handler and returns the recorder.
func do(t *testing.T, h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var r io.Reader
	if body != "" {
		r = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, r)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestGetEntityByID(t *testing.T) {
	srv, _, _, _ := setupTestServer(t)
	h := srv.Handler()

	seed := []struct{ path, body string }{
		{"/api/milestones", `{"id":"260915-mvp","title":"MVP","status":"open","summary":"s"}`},
		{"/api/strategies", `{"id":"zero-cgo","title":"Zero CGO","tier":1,"summary":"s"}`},
		{"/api/glossary", `{"id":"tac","title":"Tasks-as-Code","summary":"s"}`},
	}
	for _, s := range seed {
		if rec := do(t, h, http.MethodPost, s.path, s.body); rec.Code != http.StatusCreated {
			t.Fatalf("seed %s: %d %s", s.path, rec.Code, rec.Body)
		}
	}

	tests := []struct {
		name      string
		path      string
		want      int
		wantTitle string
	}{
		{"milestone found", "/api/milestones/260915-mvp", http.StatusOK, "MVP"},
		{"milestone missing", "/api/milestones/260101-nope", http.StatusNotFound, ""},
		{"milestone invalid id", "/api/milestones/Bad%20ID!", http.StatusNotFound, ""},
		{"strategy found", "/api/strategies/zero-cgo", http.StatusOK, "Zero CGO"},
		{"strategy missing", "/api/strategies/nope", http.StatusNotFound, ""},
		{"strategy invalid id", "/api/strategies/..%2F..%2Fetc", http.StatusNotFound, ""},
		{"glossary found", "/api/glossary/tac", http.StatusOK, "Tasks-as-Code"},
		{"glossary missing", "/api/glossary/nope", http.StatusNotFound, ""},
		{"glossary invalid id", "/api/glossary/UPPER_case", http.StatusNotFound, ""},
		{"task missing", "/api/tasks/260101-nope", http.StatusNotFound, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := do(t, h, http.MethodGet, tc.path, "")
			if rec.Code != tc.want {
				t.Fatalf("GET %s = %d, want %d: %s", tc.path, rec.Code, tc.want, rec.Body)
			}
			var got map[string]any
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if tc.wantTitle != "" && got["title"] != tc.wantTitle {
				t.Errorf("title = %v, want %q", got["title"], tc.wantTitle)
			}
			if tc.want != http.StatusOK && got["error"] == nil {
				t.Errorf("expected error body, got %v", got)
			}
		})
	}
}

func TestHandlersReturn500WhenStoreClosed(t *testing.T) {
	srv, _, st, _ := setupTestServer(t)
	h := srv.Handler()
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}

	tests := []struct{ method, path string }{
		{http.MethodGet, "/api/board"},
		{http.MethodGet, "/api/tasks"},
		{http.MethodGet, "/api/milestones"},
		{http.MethodGet, "/api/strategies"},
		{http.MethodGet, "/api/glossary"},
		{http.MethodGet, "/api/tags"},
		{http.MethodGet, "/api/search?q=x"},
		{http.MethodGet, "/api/tasks/some-task"},
		{http.MethodDelete, "/api/tasks/some-task"},
		{http.MethodDelete, "/api/milestones/some-ms"},
		{http.MethodDelete, "/api/strategies/some-strategy"},
		{http.MethodDelete, "/api/glossary/some-term"},
	}
	for _, tc := range tests {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			rec := do(t, h, tc.method, tc.path, "")
			if rec.Code != http.StatusInternalServerError {
				t.Fatalf("got %d, want 500: %s", rec.Code, rec.Body)
			}
		})
	}
}

func TestDeleteHandlers4xx(t *testing.T) {
	srv, _, _, _ := setupTestServer(t)
	h := srv.Handler()
	for _, path := range []string{"/api/tasks/nope", "/api/milestones/nope", "/api/strategies/nope", "/api/glossary/nope"} {
		t.Run(path, func(t *testing.T) {
			if rec := do(t, h, http.MethodDelete, path, ""); rec.Code != http.StatusNotFound {
				t.Fatalf("DELETE %s = %d, want 404: %s", path, rec.Code, rec.Body)
			}
		})
	}
}

func TestMalformedJSONBodies(t *testing.T) {
	srv, _, _, _ := setupTestServer(t)
	h := srv.Handler()
	tests := []struct{ method, path string }{
		{http.MethodPost, "/api/tasks"},
		{http.MethodPut, "/api/tasks/any"},
		{http.MethodPost, "/api/milestones"},
		{http.MethodPost, "/api/strategies"},
		{http.MethodPost, "/api/glossary"},
		{http.MethodPost, "/api/tasks/any/dependencies"},
		{http.MethodPost, "/api/tasks/any/notes"},
		{http.MethodPut, "/api/tasks/any/status"},
	}
	for _, tc := range tests {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			rec := do(t, h, tc.method, tc.path, `{"broken":`)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("got %d, want 400: %s", rec.Code, rec.Body)
			}
		})
	}
}

func TestHandleUpdateTask(t *testing.T) {
	srv, _, _, _ := setupTestServer(t)
	h := srv.Handler()

	if rec := do(t, h, http.MethodPost, "/api/milestones", `{"id":"260915-mvp","title":"MVP","summary":"s"}`); rec.Code != http.StatusCreated {
		t.Fatalf("seed milestone: %d %s", rec.Code, rec.Body)
	}
	if rec := do(t, h, http.MethodPost, "/api/tasks", `{"id":"260101-dep","title":"Dep","summary":"s"}`); rec.Code != http.StatusCreated {
		t.Fatalf("seed dep: %d %s", rec.Code, rec.Body)
	}
	rec := do(t, h, http.MethodPost, "/api/tasks", `{"id":"260101-main","title":"Main","summary":"s","body":"original"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("seed main: %d %s", rec.Code, rec.Body)
	}

	tests := []struct {
		name  string
		id    string
		body  string
		want  int
		check func(t *testing.T, task model.Task)
	}{
		{"invalid status", "260101-main", `{"status":"nowhere"}`, http.StatusBadRequest, nil},
		{"invalid target_at", "260101-main", `{"target_at":"not a date"}`, http.StatusBadRequest, nil},
		{"unknown task", "260101-missing", `{"title":"x"}`, http.StatusNotFound, nil},
		{
			"all fields", "260101-main",
			`{"title":"  New title  ","priority":"high","milestone":" 260915-mvp ","tags":["backend","api"],"summary":" new summary ","dependencies":["260101-dep"],"target_at":"2026-12-24","body":"edited"}`,
			http.StatusOK,
			func(t *testing.T, task model.Task) {
				if task.Title != "New title" || task.Priority != model.PriorityHigh || task.Milestone != "260915-mvp" ||
					len(task.Tags) != 2 || task.Summary != "new summary" || len(task.Dependencies) != 1 ||
					task.TargetAt == "" || task.Body != "edited" {
					t.Errorf("unexpected task: %+v", task)
				}
			},
		},
		{"invalid priority is rejected", "260101-main", `{"priority":"ultra"}`, http.StatusBadRequest, nil},
		{"empty title is rejected", "260101-main", `{"title":"  "}`, http.StatusBadRequest, nil},
		{"unknown milestone", "260101-main", `{"milestone":"260101-nope"}`, http.StatusNotFound, nil},
		{"unknown dependency", "260101-main", `{"dependencies":["260101-nope"]}`, http.StatusNotFound, nil},
		{"self dependency", "260101-main", `{"dependencies":["260101-main"]}`, http.StatusBadRequest, nil},
		{"tag outside vocabulary", "260101-main", `{"tags":["nope"]}`, http.StatusBadRequest, nil},
		{"move out of editable state", "260101-main", `{"status":" in_progress "}`, http.StatusOK, nil},
		{"body edit outside editable state", "260101-main", `{"body":"rewritten"}`, http.StatusConflict, nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := do(t, h, http.MethodPut, "/api/tasks/"+tc.id, tc.body)
			if rec.Code != tc.want {
				t.Fatalf("got %d, want %d: %s", rec.Code, tc.want, rec.Body)
			}
			if tc.check != nil {
				var task model.Task
				if err := json.Unmarshal(rec.Body.Bytes(), &task); err != nil {
					t.Fatal(err)
				}
				tc.check(t, task)
			}
		})
	}
}

func TestAddDependencyRequiresID(t *testing.T) {
	srv, _, _, _ := setupTestServer(t)
	rec := do(t, srv.Handler(), http.MethodPost, "/api/tasks/any/dependencies", `{"dependency_id":"   "}`)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "dependency_id is required") {
		t.Fatalf("got %d %s, want 400 dependency_id is required", rec.Code, rec.Body)
	}
}

func TestSearchQueryParameters(t *testing.T) {
	srv, _, _, _ := setupTestServer(t)
	h := srv.Handler()
	seed := []struct{ path, body string }{
		{"/api/tasks", `{"id":"260101-fts","title":"Searchable task","summary":"needle","tags":["docs"]}`},
		{"/api/milestones", `{"id":"260101-ms","title":"Searchable milestone","summary":"needle","tags":["docs"]}`},
		{"/api/strategies", `{"id":"strat","title":"Searchable strategy","tier":1,"summary":"needle","tags":["docs"]}`},
		{"/api/glossary", `{"id":"term","title":"Searchable term","summary":"needle","tags":["docs"]}`},
	}
	for _, s := range seed {
		if rec := do(t, h, http.MethodPost, s.path, s.body); rec.Code != http.StatusCreated {
			t.Fatalf("seed %s: %d %s", s.path, rec.Code, rec.Body)
		}
	}

	tests := []struct {
		query string
		wantN int
	}{
		{"q=needle", 4},
		{"q=needle&limit=2", 2},
		{"q=needle&limit=abc", 4},
		{"q=needle&limit=-5", 4},
		{"q=needle&type=task", 1},
		{"q=needle&type=milestone", 1},
		{"q=needle&type=strategy", 1},
		{"q=needle&type=glossary&tag=docs", 1},
		{"q=needle&tag=missing", 0},
		{"q=", 0},
	}
	for _, tc := range tests {
		t.Run(tc.query, func(t *testing.T) {
			rec := do(t, h, http.MethodGet, "/api/search?"+tc.query, "")
			if rec.Code != http.StatusOK {
				t.Fatalf("got %d: %s", rec.Code, rec.Body)
			}
			var res []model.SearchResult
			if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
				t.Fatal(err)
			}
			if res == nil || len(res) != tc.wantN {
				t.Fatalf("got %d results (%s), want %d", len(res), rec.Body, tc.wantN)
			}
		})
	}
}

func TestBoardDefaultsWhenConfigEmpty(t *testing.T) {
	srv, _ := newConfiguredServer(t, func(c *config.Config) {
		c.Board.EditableStates = nil
		c.Board.CreatableStates = nil
		c.Board.DefaultCreateState = ""
	})
	rec := do(t, srv.Handler(), http.MethodGet, "/api/board", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("got %d: %s", rec.Code, rec.Body)
	}
	var board model.BoardState
	if err := json.Unmarshal(rec.Body.Bytes(), &board); err != nil {
		t.Fatal(err)
	}
	if strings.Join(board.EditableStates, ",") != "backlog" || strings.Join(board.CreatableStates, ",") != "backlog" || board.DefaultCreateState != "backlog" {
		t.Errorf("unexpected board defaults: editable=%v creatable=%v default=%q", board.EditableStates, board.CreatableStates, board.DefaultCreateState)
	}
}

func TestBoardCarriesLocaleAndTranslations(t *testing.T) {
	srv, _ := newConfiguredServer(t, func(c *config.Config) {
		c.Project.Locale = "fi-FI"
		c.Translations["board"] = "Taulu"
	})
	rec := do(t, srv.Handler(), http.MethodGet, "/api/board", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("got %d: %s", rec.Code, rec.Body)
	}
	var board model.BoardState
	if err := json.Unmarshal(rec.Body.Bytes(), &board); err != nil {
		t.Fatal(err)
	}
	if board.Locale != "fi-FI" {
		t.Errorf("locale = %q, want %q", board.Locale, "fi-FI")
	}
	if board.Translations["board"] != "Taulu" {
		t.Errorf("translations[board] = %q, want %q", board.Translations["board"], "Taulu")
	}
}

func TestMiddlewareEdgeCases(t *testing.T) {
	srv, _ := newConfiguredServer(t, func(c *config.Config) {
		c.Server.Security.CORSEnabled = true
		c.Server.Security.CORSAllowedOrigins = []string{"*", "::not a url::"}
	})
	srv.SetMCPHandler(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}))
	h := srv.Handler()

	t.Run("panic is recovered as 500", func(t *testing.T) {
		rec := do(t, h, http.MethodGet, "/api/mcp", "")
		if rec.Code != http.StatusInternalServerError || !strings.Contains(rec.Body.String(), "internal server error") {
			t.Fatalf("got %d %s", rec.Code, rec.Body)
		}
	})

	t.Run("CORS preflight", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/api/tasks", nil)
		req.Header.Set("Origin", "http://example.test")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("got %d, want 204", rec.Code)
		}
		if rec.Header().Get("Access-Control-Allow-Origin") != "http://example.test" {
			t.Errorf("missing CORS allow-origin header: %v", rec.Header())
		}
	})
}

func TestMCPEndpointNotConfigured(t *testing.T) {
	srv, _, _, _ := setupTestServer(t)
	if rec := do(t, srv.Handler(), http.MethodGet, "/api/mcp", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("got %d, want 404", rec.Code)
	}
}

func TestServerLifecycle(t *testing.T) {
	t.Run("not started", func(t *testing.T) {
		srv, _ := newConfiguredServer(t, func(c *config.Config) { c.Server.Port = 4321 })
		if got := srv.Addr(); got != "127.0.0.1:4321" {
			t.Errorf("Addr() = %q", got)
		}
		if got := srv.Port(); got != 4321 {
			t.Errorf("Port() = %d", got)
		}
		if err := srv.Shutdown(t.Context()); err != nil {
			t.Errorf("Shutdown of unstarted server: %v", err)
		}
		if err := srv.Close(); err != nil {
			t.Errorf("Close of unstarted server: %v", err)
		}
	})

	t.Run("live port and graceful shutdown", func(t *testing.T) {
		srv, _ := newConfiguredServer(t, nil)
		if err := srv.Start(); err != nil {
			t.Fatal(err)
		}
		port := srv.Port()
		if port == 0 || !strings.HasSuffix(srv.Addr(), ":"+strconv.Itoa(port)) {
			t.Fatalf("Port() = %d, Addr() = %q", port, srv.Addr())
		}
		resp, err := http.Get("http://" + srv.Addr() + "/api/health")
		if err != nil {
			t.Fatal(err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("health = %d", resp.StatusCode)
		}

		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			t.Fatalf("Shutdown: %v", err)
		}
		if _, err := http.Get("http://" + srv.Addr() + "/api/health"); err == nil {
			t.Error("expected request to fail after Shutdown")
		}
	})

	t.Run("port already in use", func(t *testing.T) {
		live := startLiveServer(t)
		srv, _ := newConfiguredServer(t, func(c *config.Config) { c.Server.Port = live.Port() })
		if err := srv.Start(); err == nil || !strings.Contains(err.Error(), "failed to bind") {
			_ = srv.Close()
			t.Fatalf("Start on busy port: err = %v", err)
		}
	})
}

func TestSSEBroadcastWithoutSubscribers(t *testing.T) {
	t.Run("nil hub", func(t *testing.T) {
		var hub *server.SSEHub
		hub.Broadcast("task.updated", nil) // must not panic
	})

	t.Run("started hub without clients", func(t *testing.T) {
		hub := server.NewSSEHub()
		hub.Start()
		defer hub.Stop()
		for range 10 {
			hub.Broadcast("task.updated", map[string]string{"id": "x"})
		}
	})

	t.Run("full buffer drops instead of blocking", func(t *testing.T) {
		hub := server.NewSSEHub() // not started: nothing drains the buffer
		done := make(chan struct{})
		go func() {
			for range 200 {
				hub.Broadcast("task.updated", nil)
			}
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("Broadcast blocked on a full buffer")
		}
		hub.Stop()
		hub.Stop() // idempotent
	})
}

type noFlushWriter struct {
	header http.Header
	code   int
}

func (w *noFlushWriter) Header() http.Header         { return w.header }
func (w *noFlushWriter) Write(b []byte) (int, error) { return len(b), nil }
func (w *noFlushWriter) WriteHeader(code int)        { w.code = code }

func TestSSEStreamingUnsupported(t *testing.T) {
	hub := server.NewSSEHub()
	w := &noFlushWriter{header: http.Header{}}
	hub.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/events", nil))
	if w.code != http.StatusInternalServerError {
		t.Fatalf("got %d, want 500", w.code)
	}
}

func TestSSESkipsUnmarshalablePayloadAndClosesOnStop(t *testing.T) {
	hub := server.NewSSEHub()
	hub.Start()
	ts := httptest.NewServer(hub)
	defer ts.Close()
	defer hub.Stop()

	resp, err := http.Get(ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	reader := bufio.NewReader(resp.Body)

	readEvent := func() string {
		t.Helper()
		var lines []string
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				return strings.Join(lines, "|") + "|EOF"
			}
			line = strings.TrimRight(line, "\n")
			if line == "" {
				return strings.Join(lines, "|")
			}
			lines = append(lines, line)
		}
	}

	if ev := readEvent(); !strings.HasPrefix(ev, "event: connected") {
		t.Fatalf("first event = %q", ev)
	}
	// The client is registered once the handshake has been read.
	hub.Broadcast("bad", map[string]any{"fn": func() {}})
	hub.Broadcast("good", map[string]string{"id": "x"})
	if ev := readEvent(); ev != `event: good|data: {"id":"x"}` {
		t.Fatalf("expected the unmarshalable event to be skipped, got %q", ev)
	}

	hub.Stop()
	if ev := readEvent(); !strings.HasSuffix(ev, "EOF") {
		t.Fatalf("expected stream to end after Stop, got %q", ev)
	}
}

func TestTaskArchivedMilestoneNeedsReopen(t *testing.T) {
	srv, _, _, _ := setupTestServer(t)
	h := srv.Handler()

	if rec := do(t, h, http.MethodPost, "/api/milestones", `{"id":"260101-closed","title":"Closed","status":"closed","summary":"s"}`); rec.Code != http.StatusCreated {
		t.Fatalf("seed milestone: %d %s", rec.Code, rec.Body)
	}

	rec := do(t, h, http.MethodPost, "/api/tasks", `{"id":"260101-a","title":"A","summary":"s","milestone":"260101-closed"}`)
	if rec.Code != http.StatusConflict {
		t.Fatalf("create on closed milestone: got %d, want 409: %s", rec.Code, rec.Body)
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["code"] != service.CodeMilestoneArchived {
		t.Fatalf("code = %q, want %q", body["code"], service.CodeMilestoneArchived)
	}

	if rec := do(t, h, http.MethodPost, "/api/tasks", `{"id":"260101-a","title":"A","summary":"s","milestone":"260101-closed","reopen_milestone":true}`); rec.Code != http.StatusCreated {
		t.Fatalf("create with reopen: got %d: %s", rec.Code, rec.Body)
	}
	if rec := do(t, h, http.MethodPost, "/api/tasks", `{"id":"260101-b","title":"B","summary":"s"}`); rec.Code != http.StatusCreated {
		t.Fatalf("seed task: %d %s", rec.Code, rec.Body)
	}
	if rec := do(t, h, http.MethodPut, "/api/tasks/260101-b", `{"milestone":"260101-closed"}`); rec.Code != http.StatusConflict {
		t.Fatalf("update onto closed milestone: got %d, want 409: %s", rec.Code, rec.Body)
	}
	if rec := do(t, h, http.MethodPut, "/api/tasks/260101-b", `{"milestone":"260101-closed","reopen_milestone":true}`); rec.Code != http.StatusOK {
		t.Fatalf("update with reopen: got %d: %s", rec.Code, rec.Body)
	}
	// Unchanged milestone: unrelated edits are not blocked.
	if rec := do(t, h, http.MethodPut, "/api/tasks/260101-b", `{"title":"B2","milestone":"260101-closed"}`); rec.Code != http.StatusOK {
		t.Fatalf("edit with unchanged milestone: got %d: %s", rec.Code, rec.Body)
	}
}
