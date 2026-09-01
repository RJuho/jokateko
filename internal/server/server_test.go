package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/RJuho/jokateko/internal/config"
	"github.com/RJuho/jokateko/internal/model"
	"github.com/RJuho/jokateko/internal/server"
	"github.com/RJuho/jokateko/internal/store"
	"github.com/RJuho/jokateko/internal/writer"
)

func setupTestServer(t *testing.T) (*server.Server, string, *store.Store, *server.SSEHub) {
	t.Helper()
	dir := t.TempDir()

	tasksDir := filepath.Join(dir, ".jokateko", "tasks")
	msDir := filepath.Join(dir, ".jokateko", "milestones")
	stratDir := filepath.Join(dir, ".jokateko", "strategies")
	glossDir := filepath.Join(dir, ".jokateko", "glossary")
	for _, d := range []string{tasksDir, msDir, stratDir, glossDir} {
		_ = os.MkdirAll(d, 0755)
	}

	cfg := config.Default(dir)
	cfg.Project.Name = "Test Project"
	cfg.Server.Security.CORSEnabled = true
	cfg.Server.Security.CSP.Enabled = true

	st, err := store.OpenMemory()
	if err != nil {
		t.Fatalf("failed to open in-memory store: %v", err)
	}
	t.Cleanup(func() {
		_ = st.Close()
	})

	sc := writer.NewSuppressionCache(time.Second)
	wr := writer.New(sc)
	sse := server.NewSSEHub()
	sse.Start()
	t.Cleanup(func() {
		sse.Stop()
	})

	srv := server.New(cfg, dir, st, wr, sse)
	return srv, dir, st, sse
}

func TestHealthAndVersion(t *testing.T) {
	srv, _, _, _ := setupTestServer(t)

	// Health
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}

	var health server.HealthResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &health); err != nil {
		t.Fatalf("failed to decode health response: %v", err)
	}
	if health.Status != "ok" || health.Project != "Test Project" {
		t.Errorf("unexpected health response: %+v", health)
	}

	// Version
	req = httptest.NewRequest(http.MethodGet, "/api/version", nil)
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}
}

func TestStaticUI(t *testing.T) {
	srv, _, _, _ := setupTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Errorf("expected text/html content-type, got %q", ct)
	}

	// 404 on nonexistent API endpoint
	req = httptest.NewRequest(http.MethodGet, "/api/nonexistent", nil)
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for /api/nonexistent, got %d", rec.Code)
	}
}

func TestBoardState(t *testing.T) {
	srv, _, _, _ := setupTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/board", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}

	var board model.BoardState
	if err := json.Unmarshal(rec.Body.Bytes(), &board); err != nil {
		t.Fatalf("failed to decode board state: %v", err)
	}
	if len(board.Columns) < 2 {
		t.Errorf("expected at least 2 board columns, got %d", len(board.Columns))
	}
}

func TestTasksCRUD(t *testing.T) {
	srv, dir, _, _ := setupTestServer(t)

	// 1. Create task
	createPayload := map[string]any{
		"id":           "260901-test-task",
		"title":        "Test Task",
		"status":       "ready",
		"priority":     "high",
		"tags":         []string{"backend"},
		"dependencies": []string{},
		"summary":      "A test task summary",
		"body":         "## Details\n- [ ] Checklist item\n",
	}
	body, _ := json.Marshal(createPayload)

	req := httptest.NewRequest(http.MethodPost, "/api/tasks", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", rec.Code, rec.Body.String())
	}

	var created model.Task
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("failed to decode task: %v", err)
	}
	if created.ID != "260901-test-task" || created.Title != "Test Task" {
		t.Errorf("unexpected created task: %+v", created)
	}

	// Verify file was written to disk
	taskFilePath := filepath.Join(dir, ".jokateko", "tasks", "260901-test-task.md")
	if _, err := os.Stat(taskFilePath); os.IsNotExist(err) {
		t.Errorf("expected task file to exist at %s", taskFilePath)
	}

	// 2. Get task
	req = httptest.NewRequest(http.MethodGet, "/api/tasks/260901-test-task", nil)
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}

	// 3. Update task
	updatePayload := map[string]any{
		"status": "done",
	}
	updateBody, _ := json.Marshal(updatePayload)
	req = httptest.NewRequest(http.MethodPut, "/api/tasks/260901-test-task", bytes.NewReader(updateBody))
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on update, got %d: %s", rec.Code, rec.Body.String())
	}

	var updated model.Task
	_ = json.Unmarshal(rec.Body.Bytes(), &updated)
	if updated.Status != "done" {
		t.Errorf("expected status 'done', got %q", updated.Status)
	}

	// 4. List tasks
	req = httptest.NewRequest(http.MethodGet, "/api/tasks", nil)
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}
	var list []model.Task
	_ = json.Unmarshal(rec.Body.Bytes(), &list)
	if len(list) != 1 {
		t.Errorf("expected 1 task in list, got %d", len(list))
	}

	// 5. Delete task
	req = httptest.NewRequest(http.MethodDelete, "/api/tasks/260901-test-task", nil)
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on delete, got %d", rec.Code)
	}

	// Verify file is gone
	if _, err := os.Stat(taskFilePath); !os.IsNotExist(err) {
		t.Errorf("expected task file to be deleted from %s", taskFilePath)
	}
}

func TestMilestonesCRUD(t *testing.T) {
	srv, _, _, _ := setupTestServer(t)

	// Create
	createPayload := map[string]any{
		"id":          "260915-mvp",
		"title":       "MVP Milestone",
		"status":      "open",
		"target_date": "2026-09-15",
		"tags":        []string{"mvp"},
		"summary":     "MVP release target",
		"body":        "# Milestone Roadmap",
	}
	body, _ := json.Marshal(createPayload)
	req := httptest.NewRequest(http.MethodPost, "/api/milestones", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", rec.Code, rec.Body.String())
	}

	// List
	req = httptest.NewRequest(http.MethodGet, "/api/milestones", nil)
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}

	// Delete
	req = httptest.NewRequest(http.MethodDelete, "/api/milestones/260915-mvp", nil)
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}
}

func TestStrategiesCRUD(t *testing.T) {
	srv, _, _, _ := setupTestServer(t)

	createPayload := map[string]any{
		"id":      "zero-cgo",
		"title":   "Zero CGO Architecture",
		"tier":    1,
		"tags":    []string{"core"},
		"summary": "Rules for zero CGO",
		"body":    "Strict rules.",
	}
	body, _ := json.Marshal(createPayload)
	req := httptest.NewRequest(http.MethodPost, "/api/strategies", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/strategies", nil)
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodDelete, "/api/strategies/zero-cgo", nil)
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}
}

func TestGlossaryCRUD(t *testing.T) {
	srv, _, _, _ := setupTestServer(t)

	createPayload := map[string]any{
		"id":      "tac",
		"title":   "Tasks-as-Code",
		"tags":    []string{"concept"},
		"summary": "Markdown files as tasks",
		"body":    "Explanation.",
	}
	body, _ := json.Marshal(createPayload)
	req := httptest.NewRequest(http.MethodPost, "/api/glossary", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/glossary", nil)
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodDelete, "/api/glossary/tac", nil)
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}
}

func TestTagsAndSearch(t *testing.T) {
	srv, _, _, _ := setupTestServer(t)

	// Tags
	req := httptest.NewRequest(http.MethodGet, "/api/tags", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}

	// Search
	req = httptest.NewRequest(http.MethodGet, "/api/search?q=test", nil)
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}
}

func TestSSEEventStream(t *testing.T) {
	srv, _, _, sse := setupTestServer(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req := httptest.NewRequest(http.MethodGet, "/api/events", nil).WithContext(ctx)
	rec := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		srv.Handler().ServeHTTP(rec, req)
		close(done)
	}()

	// Allow subscriber to register
	time.Sleep(50 * time.Millisecond)

	// Broadcast test event
	sse.Broadcast("task.updated", map[string]string{"id": "260901-sample"})

	time.Sleep(50 * time.Millisecond)
	cancel()
	<-done

	body := rec.Body.String()
	if !strings.Contains(body, "event: connected") {
		t.Errorf("expected 'event: connected' in SSE stream, got:\n%s", body)
	}
	if !strings.Contains(body, "event: task.updated") {
		t.Errorf("expected 'event: task.updated' in SSE stream, got:\n%s", body)
	}
}

func TestSecurityHeaders(t *testing.T) {
	srv, _, _, _ := setupTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	csp := rec.Header().Get("Content-Security-Policy")
	if !strings.Contains(csp, "default-src") {
		t.Errorf("expected Content-Security-Policy header, got %q", csp)
	}

	cors := rec.Header().Get("Access-Control-Allow-Origin")
	if cors != "http://localhost:3000" {
		t.Errorf("expected Access-Control-Allow-Origin: http://localhost:3000, got %q", cors)
	}
}
