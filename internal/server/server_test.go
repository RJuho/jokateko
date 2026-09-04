package server_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"io"
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

	// 1. Uncompressed request (without Accept-Encoding: gzip)
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
	ce := rec.Header().Get("Content-Encoding")
	if ce != "" {
		t.Errorf("expected no Content-Encoding for non-gzip request, got %q", ce)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("<!DOCTYPE html>")) {
		t.Errorf("expected <!DOCTYPE html> in uncompressed body")
	}

	// 2. Compressed request (with Accept-Encoding: gzip)
	reqGz := httptest.NewRequest(http.MethodGet, "/", nil)
	reqGz.Header.Set("Accept-Encoding", "gzip, deflate, br")
	recGz := httptest.NewRecorder()
	srv.Handler().ServeHTTP(recGz, reqGz)

	if recGz.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for gzip request, got %d", recGz.Code)
	}
	if recGz.Header().Get("Content-Encoding") != "gzip" {
		t.Errorf("expected Content-Encoding: gzip, got %q", recGz.Header().Get("Content-Encoding"))
	}

	gzReader, err := gzip.NewReader(recGz.Body)
	if err != nil {
		t.Fatalf("failed to create gzip reader from response: %v", err)
	}
	defer gzReader.Close()

	decompressed, err := io.ReadAll(gzReader)
	if err != nil {
		t.Fatalf("failed to decompress gzip response: %v", err)
	}
	if !bytes.Contains(decompressed, []byte("<!DOCTYPE html>")) {
		t.Errorf("expected <!DOCTYPE html> in decompressed gzip body")
	}

	// 3. 404 on nonexistent API endpoint
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
	req.Header.Set("Origin", "http://localhost:8080")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	csp := rec.Header().Get("Content-Security-Policy")
	if !strings.Contains(csp, "default-src") {
		t.Errorf("expected Content-Security-Policy header, got %q", csp)
	}

	// Verify SHA-256 hashes are injected into script-src and style-src
	if !strings.Contains(csp, "script-src 'self' 'sha256-") {
		t.Errorf("expected script-src to contain 'self' and 'sha256-...', got %q", csp)
	}
	if !strings.Contains(csp, "style-src 'self' 'sha256-") {
		t.Errorf("expected style-src to contain 'self' and 'sha256-...', got %q", csp)
	}

	// Verify unsafe-inline is NOT in script-src or style-src
	if strings.Contains(csp, "script-src 'self' 'unsafe-inline'") || strings.Contains(csp, "script-src 'unsafe-inline'") {
		t.Errorf("script-src must NOT contain 'unsafe-inline', got %q", csp)
	}
	if strings.Contains(csp, "style-src 'self' 'unsafe-inline'") || strings.Contains(csp, "style-src 'unsafe-inline'") {
		t.Errorf("style-src must NOT contain 'unsafe-inline', got %q", csp)
	}

	// Verify style-src-attr is set for element style attributes
	if !strings.Contains(csp, "style-src-attr 'unsafe-inline'") {
		t.Errorf("expected style-src-attr 'unsafe-inline', got %q", csp)
	}

	// Verify style-src-elem allows inline style elements for dynamic Mermaid diagrams
	if !strings.Contains(csp, "style-src-elem 'self' 'unsafe-inline'") {
		t.Errorf("expected style-src-elem 'self' 'unsafe-inline', got %q", csp)
	}

	cors := rec.Header().Get("Access-Control-Allow-Origin")
	if cors != "http://localhost:8080" {
		t.Errorf("expected Access-Control-Allow-Origin: http://localhost:8080, got %q", cors)
	}
}

func TestTaskDeletionSafeguards(t *testing.T) {
	srv, dir, st, _ := setupTestServer(t)

	// Create task A
	taskAPayload := map[string]any{
		"id":       "260901-task-a",
		"title":    "Task A",
		"status":   "todo",
		"priority": "medium",
		"tags":     []string{"backend"},
		"summary":  "Base task",
		"body":     "# Task A",
	}
	bodyA, _ := json.Marshal(taskAPayload)
	req := httptest.NewRequest(http.MethodPost, "/api/tasks", bytes.NewReader(bodyA))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for task A, got %d: %s", rec.Code, rec.Body.String())
	}

	// Create task B that depends on task A
	taskBPayload := map[string]any{
		"id":           "260902-task-b",
		"title":        "Task B",
		"status":       "todo",
		"priority":     "medium",
		"tags":         []string{"backend"},
		"summary":      "Dependent task",
		"dependencies": []string{"260901-task-a"},
		"body":         "# Task B",
	}
	bodyB, _ := json.Marshal(taskBPayload)
	req = httptest.NewRequest(http.MethodPost, "/api/tasks", bytes.NewReader(bodyB))
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for task B, got %d: %s", rec.Code, rec.Body.String())
	}

	// Attempt to delete task A without force -> should return 409 Conflict
	req = httptest.NewRequest(http.MethodDelete, "/api/tasks/260901-task-a", nil)
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409 Conflict deleting task with dependents, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "depend on it") {
		t.Errorf("expected conflict message mentioning dependencies, got %s", rec.Body.String())
	}

	// Delete task A with force=true -> should succeed with 200 OK
	req = httptest.NewRequest(http.MethodDelete, "/api/tasks/260901-task-a?force=true", nil)
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK with force=true, got %d: %s", rec.Code, rec.Body.String())
	}

	// Verify task A is removed from store and filesystem
	_, err := st.GetTask(t.Context(), "260901-task-a")
	if !errors.Is(err, store.ErrNotFound) {
		t.Errorf("expected task A to be deleted from store, got error: %v", err)
	}
	taskFilePath := filepath.Join(dir, ".jokateko", "tasks", "260901-task-a.md")
	if _, err := os.Stat(taskFilePath); !os.IsNotExist(err) {
		t.Errorf("expected task file to be removed from %s", taskFilePath)
	}
}

func TestMilestoneDeletionSafeguards(t *testing.T) {
	srv, dir, st, _ := setupTestServer(t)

	// Create milestone
	msPayload := map[string]any{
		"id":          "260915-mvp",
		"title":       "MVP Milestone",
		"status":      "open",
		"target_date": "2026-09-15",
		"tags":        []string{"mvp"},
		"summary":     "MVP roadmap target",
		"body":        "# MVP Roadmap",
	}
	bodyMS, _ := json.Marshal(msPayload)
	req := httptest.NewRequest(http.MethodPost, "/api/milestones", bytes.NewReader(bodyMS))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for milestone, got %d: %s", rec.Code, rec.Body.String())
	}

	// Create task assigned to milestone
	taskPayload := map[string]any{
		"id":        "260901-assigned-task",
		"title":     "Assigned Task",
		"status":    "todo",
		"priority":  "medium",
		"milestone": "260915-mvp",
		"tags":      []string{"backend"},
		"summary":   "Task assigned to milestone",
		"body":      "# Assigned Task",
	}
	bodyTask, _ := json.Marshal(taskPayload)
	req = httptest.NewRequest(http.MethodPost, "/api/tasks", bytes.NewReader(bodyTask))
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for task, got %d: %s", rec.Code, rec.Body.String())
	}

	// Attempt to delete milestone without force -> should return 409 Conflict
	req = httptest.NewRequest(http.MethodDelete, "/api/milestones/260915-mvp", nil)
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409 Conflict deleting milestone with assigned tasks, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "are assigned to it") {
		t.Errorf("expected conflict message mentioning assigned tasks, got %s", rec.Body.String())
	}

	// Delete milestone with force=true -> should succeed with 200 OK
	req = httptest.NewRequest(http.MethodDelete, "/api/milestones/260915-mvp?force=true", nil)
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK with force=true, got %d: %s", rec.Code, rec.Body.String())
	}

	// Verify milestone is removed from store and filesystem
	_, err := st.GetMilestone(t.Context(), "260915-mvp")
	if !errors.Is(err, store.ErrNotFound) {
		t.Errorf("expected milestone to be deleted from store, got error: %v", err)
	}
	msFilePath := filepath.Join(dir, ".jokateko", "milestones", "260915-mvp.md")
	if _, err := os.Stat(msFilePath); !os.IsNotExist(err) {
		t.Errorf("expected milestone file to be removed from %s", msFilePath)
	}
}

func TestTaskDependenciesEndpoints(t *testing.T) {
	srv, _, st, _ := setupTestServer(t)

	// Create task A
	taskA := map[string]any{
		"id":      "260901-task-a",
		"title":   "Task A",
		"status":  "todo",
		"summary": "Task A summary",
		"body":    "# Task A",
	}
	bodyA, _ := json.Marshal(taskA)
	req := httptest.NewRequest(http.MethodPost, "/api/tasks", bytes.NewReader(bodyA))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("failed to create task A, got %d: %s", rec.Code, rec.Body.String())
	}

	// Create task B
	taskB := map[string]any{
		"id":      "260902-task-b",
		"title":   "Task B",
		"status":  "todo",
		"summary": "Task B summary",
		"body":    "# Task B",
	}
	bodyB, _ := json.Marshal(taskB)
	req = httptest.NewRequest(http.MethodPost, "/api/tasks", bytes.NewReader(bodyB))
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("failed to create task B, got %d: %s", rec.Code, rec.Body.String())
	}

	// 1. Self dependency rejection (400)
	selfDepPayload, _ := json.Marshal(map[string]string{"dependency_id": "260901-task-a"})
	req = httptest.NewRequest(http.MethodPost, "/api/tasks/260901-task-a/dependencies", bytes.NewReader(selfDepPayload))
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request for self-dependency, got %d", rec.Code)
	}

	// 2. Non-existent dependency (404)
	missingDepPayload, _ := json.Marshal(map[string]string{"dependency_id": "non-existent"})
	req = httptest.NewRequest(http.MethodPost, "/api/tasks/260901-task-a/dependencies", bytes.NewReader(missingDepPayload))
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 Not Found for missing dependency task, got %d", rec.Code)
	}

	// 3. Add dependency: B depends on A
	addDepPayload, _ := json.Marshal(map[string]string{"dependency_id": "260901-task-a"})
	req = httptest.NewRequest(http.MethodPost, "/api/tasks/260902-task-b/dependencies", bytes.NewReader(addDepPayload))
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK adding dependency, got %d: %s", rec.Code, rec.Body.String())
	}
	var resTask model.Task
	_ = json.Unmarshal(rec.Body.Bytes(), &resTask)
	if len(resTask.Dependencies) != 1 || resTask.Dependencies[0] != "260901-task-a" {
		t.Errorf("unexpected dependencies in response: %+v", resTask.Dependencies)
	}

	// 4. Duplicate edge handling (200 OK, idempotent, no duplicates)
	req = httptest.NewRequest(http.MethodPost, "/api/tasks/260902-task-b/dependencies", bytes.NewReader(addDepPayload))
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on duplicate dependency add, got %d: %s", rec.Code, rec.Body.String())
	}
	var resDup model.Task
	_ = json.Unmarshal(rec.Body.Bytes(), &resDup)
	if len(resDup.Dependencies) != 1 {
		t.Errorf("expected 1 dependency after duplicate add, got: %d", len(resDup.Dependencies))
	}

	// 5. Cycle rejection (409 Conflict) when adding A -> B (since B already depends on A)
	cyclePayload, _ := json.Marshal(map[string]string{"dependency_id": "260902-task-b"})
	req = httptest.NewRequest(http.MethodPost, "/api/tasks/260901-task-a/dependencies", bytes.NewReader(cyclePayload))
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409 Conflict on circular dependency, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "circular dependency detected") {
		t.Errorf("expected circular dependency error message, got: %s", rec.Body.String())
	}

	// 6. Remove non-existent dependency (404)
	req = httptest.NewRequest(http.MethodDelete, "/api/tasks/260902-task-b/dependencies/non-existent-dep", nil)
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 Not Found removing non-existent dependency, got %d", rec.Code)
	}

	// 7. Remove existing dependency (200 OK)
	req = httptest.NewRequest(http.MethodDelete, "/api/tasks/260902-task-b/dependencies/260901-task-a", nil)
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK removing dependency, got %d: %s", rec.Code, rec.Body.String())
	}
	var resRemoved model.Task
	_ = json.Unmarshal(rec.Body.Bytes(), &resRemoved)
	if len(resRemoved.Dependencies) != 0 {
		t.Errorf("expected 0 dependencies after removal, got: %+v", resRemoved.Dependencies)
	}

	// Verify in store
	taskInStore, err := st.GetTask(t.Context(), "260902-task-b")
	if err != nil {
		t.Fatalf("failed to get task from store: %v", err)
	}
	if len(taskInStore.Dependencies) != 0 {
		t.Errorf("expected 0 dependencies in store, got: %+v", taskInStore.Dependencies)
	}
}

func TestTaskNotesAndEditableStatesEndpoints(t *testing.T) {
	srv, _, st, _ := setupTestServer(t)

	// 1. Create a task via POST /api/tasks (starts in backlog)
	createBody := `{"title":"API Notes Task","summary":"Testing notes and edit endpoints","body":"## Criteria\n- [ ] Initial criterion\n"}`
	req := httptest.NewRequest(http.MethodPost, "/api/tasks", strings.NewReader(createBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", rec.Code, rec.Body.String())
	}
	var createdTask model.Task
	_ = json.Unmarshal(rec.Body.Bytes(), &createdTask)
	taskID := createdTask.ID

	// 2. In backlog (editable state), PUT with modified body succeeds (200 OK)
	editBody := `{"body":"## Criteria\n- [ ] Modified criterion\n"}`
	req = httptest.NewRequest(http.MethodPut, "/api/tasks/"+taskID, strings.NewReader(editBody))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK modifying body in backlog, got %d: %s", rec.Code, rec.Body.String())
	}

	// 3. Transition to "in_progress" (non-editable status)
	putStatus := `{"status":"in_progress"}`
	req = httptest.NewRequest(http.MethodPut, "/api/tasks/"+taskID, strings.NewReader(putStatus))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK updating status, got %d: %s", rec.Code, rec.Body.String())
	}

	// 4. In in_progress, PUT that changes body text fails with 409 Conflict
	rewriteBody := `{"body":"## Criteria\n- [ ] Attempted rewrite\n"}`
	req = httptest.NewRequest(http.MethodPut, "/api/tasks/"+taskID, strings.NewReader(rewriteBody))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409 Conflict modifying body in in_progress, got %d: %s", rec.Code, rec.Body.String())
	}

	// 5. In in_progress, PUT that ONLY toggles checkbox succeeds (200 OK)
	toggleBody := `{"body":"## Criteria\n- [x] Modified criterion\n"}`
	req = httptest.NewRequest(http.MethodPut, "/api/tasks/"+taskID, strings.NewReader(toggleBody))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK toggling checkbox in in_progress, got %d: %s", rec.Code, rec.Body.String())
	}
	var toggledTask model.Task
	_ = json.Unmarshal(rec.Body.Bytes(), &toggledTask)
	if toggledTask.CompletedCriteria != 1 {
		t.Errorf("expected completed criteria 1, got %d", toggledTask.CompletedCriteria)
	}

	// 6. POST /api/tasks/{id}/notes with empty note fails with 400 Bad Request
	emptyNote := `{"note":""}`
	req = httptest.NewRequest(http.MethodPost, "/api/tasks/"+taskID+"/notes", strings.NewReader(emptyNote))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request on empty note, got %d", rec.Code)
	}

	// 7. POST /api/tasks/non-existent/notes fails with 404 Not Found
	validNote := `{"note":"Valid note content"}`
	req = httptest.NewRequest(http.MethodPost, "/api/tasks/non-existent/notes", strings.NewReader(validNote))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 Not Found on missing task notes, got %d", rec.Code)
	}

	// 8. POST /api/tasks/{id}/notes appends note and updates criteria
	noteWithChecklist := `{"note":"Progress report\n- [ ] Verify database indexes"}`
	req = httptest.NewRequest(http.MethodPost, "/api/tasks/"+taskID+"/notes", strings.NewReader(noteWithChecklist))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on note creation, got %d: %s", rec.Code, rec.Body.String())
	}
	var taskWithNote model.Task
	_ = json.Unmarshal(rec.Body.Bytes(), &taskWithNote)
	if taskWithNote.TotalCriteria != 2 || taskWithNote.CompletedCriteria != 1 {
		t.Errorf("expected total 2 and completed 1 criteria, got total %d, completed %d", taskWithNote.TotalCriteria, taskWithNote.CompletedCriteria)
	}
	if !strings.Contains(taskWithNote.Body, "## Notes") || !strings.Contains(taskWithNote.Body, "Verify database indexes") {
		t.Errorf("expected notes in body, got: %s", taskWithNote.Body)
	}

	// Check store directly
	inStore, err := st.GetTask(t.Context(), taskID)
	if err != nil {
		t.Fatalf("failed to get task from store: %v", err)
	}
	if !strings.Contains(inStore.Body, "Verify database indexes") {
		t.Errorf("store does not reflect appended note: %s", inStore.Body)
	}
}

func TestUpdateTaskStatus(t *testing.T) {
	srv, _, st, _ := setupTestServer(t)

	// 1. Create a task via POST /api/tasks (starts in backlog)
	createBody := `{"title":"Status Change Task","summary":"Testing PUT /api/tasks/{id}/status","body":"## Criteria\n- [ ] Ready item\n"}`
	req := httptest.NewRequest(http.MethodPost, "/api/tasks", strings.NewReader(createBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", rec.Code, rec.Body.String())
	}
	var createdTask model.Task
	_ = json.Unmarshal(rec.Body.Bytes(), &createdTask)
	taskID := createdTask.ID

	// 2. PUT /api/tasks/{id}/status to "in_progress" (200 OK)
	statusBody := `{"status":"in_progress"}`
	req = httptest.NewRequest(http.MethodPut, "/api/tasks/"+taskID+"/status", strings.NewReader(statusBody))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK updating status, got %d: %s", rec.Code, rec.Body.String())
	}
	var updatedTask model.Task
	_ = json.Unmarshal(rec.Body.Bytes(), &updatedTask)
	if updatedTask.Status != "in_progress" {
		t.Errorf("expected status in_progress, got %q", updatedTask.Status)
	}

	// Verify in store
	taskInStore, err := st.GetTask(t.Context(), taskID)
	if err != nil {
		t.Fatalf("failed to get task from store: %v", err)
	}
	if taskInStore.Status != "in_progress" {
		t.Errorf("expected status in store to be in_progress, got %q", taskInStore.Status)
	}

	// 3. PUT with invalid column status returns 400 Bad Request
	badBody := `{"status":"non_existent_column"}`
	req = httptest.NewRequest(http.MethodPut, "/api/tasks/"+taskID+"/status", strings.NewReader(badBody))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request on invalid column status, got %d", rec.Code)
	}

	// 4. PUT for non-existent task returns 404 Not Found
	req = httptest.NewRequest(http.MethodPut, "/api/tasks/non-existent-task/status", strings.NewReader(statusBody))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 Not Found on missing task, got %d", rec.Code)
	}
}



