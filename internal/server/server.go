package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"mime"
	"net"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/RJuho/jokateko/internal/config"
	"github.com/RJuho/jokateko/internal/csp"
	"github.com/RJuho/jokateko/internal/model"
	"github.com/RJuho/jokateko/internal/service"
	"github.com/RJuho/jokateko/internal/store"
	"github.com/RJuho/jokateko/web"
)

// Server encapsulates the HTTP daemon, REST API endpoints, and live SSE event hub.
type Server struct {
	cfg        *config.Config
	svc        *service.Service
	store      *store.Store
	sseHub     *SSEHub
	columns    []model.Column
	startTime  time.Time
	httpServer *http.Server
	listener   net.Listener
	handler    http.Handler
	mcpHandler http.Handler
}

// SetMCPHandler configures an optional Model Context Protocol HTTP/SSE handler mounted at /api/mcp.
func (s *Server) SetMCPHandler(h http.Handler) {
	s.mcpHandler = h
}

// New creates and configures a new Server instance. Mutations go through svc;
// sse may be nil when live updates are not needed.
func New(svc *service.Service, sse *SSEHub) *Server {
	cfg := svc.Config()

	s := &Server{
		cfg:       cfg,
		svc:       svc,
		store:     svc.Store(),
		sseHub:    sse,
		columns:   cfg.Columns(),
		startTime: time.Now(),
	}

	mux := http.NewServeMux()

	// Health & System
	mux.HandleFunc("GET /api/health", s.handleHealth)
	mux.HandleFunc("GET /api/version", s.handleVersion)
	mux.HandleFunc("GET /api/about", s.handleAbout)
	mux.HandleFunc("GET /api/licenses", s.handleLicenses)

	// Model Context Protocol (MCP) Endpoint
	mcpHandlerFunc := func(w http.ResponseWriter, r *http.Request) {
		if s.mcpHandler != nil {
			s.mcpHandler.ServeHTTP(w, r)
			return
		}
		http.Error(w, "MCP endpoint not configured", http.StatusNotFound)
	}
	mux.HandleFunc("GET /api/mcp", mcpHandlerFunc)
	mux.HandleFunc("POST /api/mcp", mcpHandlerFunc)
	mux.HandleFunc("DELETE /api/mcp", mcpHandlerFunc)

	// Server-Sent Events
	mux.HandleFunc("GET /api/events", s.sseHub.ServeHTTP)

	// Board & Tasks
	mux.HandleFunc("GET /api/board", s.handleGetBoard)
	mux.HandleFunc("GET /api/tasks", s.handleListTasks)
	mux.HandleFunc("GET /api/tasks/{id}", s.handleGetTask)
	mux.HandleFunc("POST /api/tasks", s.handleCreateTask)
	mux.HandleFunc("PUT /api/tasks/{id}", s.handleUpdateTask)
	mux.HandleFunc("PUT /api/tasks/{id}/status", s.handleUpdateTaskStatus)
	mux.HandleFunc("DELETE /api/tasks/{id}", s.handleDeleteTask)
	mux.HandleFunc("POST /api/tasks/{id}/dependencies", s.handleAddTaskDependency)
	mux.HandleFunc("DELETE /api/tasks/{id}/dependencies/{depId}", s.handleRemoveTaskDependency)
	mux.HandleFunc("POST /api/tasks/{id}/notes", s.handleAddTaskNote)

	// Milestones
	mux.HandleFunc("GET /api/milestones", s.handleListMilestones)
	mux.HandleFunc("GET /api/milestones/{id}", s.handleGetMilestone)
	mux.HandleFunc("POST /api/milestones", s.handleCreateMilestone)
	mux.HandleFunc("DELETE /api/milestones/{id}", s.handleDeleteMilestone)

	// Strategies
	mux.HandleFunc("GET /api/strategies", s.handleListStrategies)
	mux.HandleFunc("GET /api/strategies/{id}", s.handleGetStrategy)
	mux.HandleFunc("POST /api/strategies", s.handleCreateStrategy)
	mux.HandleFunc("DELETE /api/strategies/{id}", s.handleDeleteStrategy)

	// Glossary
	mux.HandleFunc("GET /api/glossary", s.handleListGlossary)
	mux.HandleFunc("GET /api/glossary/{id}", s.handleGetGlossaryTerm)
	mux.HandleFunc("POST /api/glossary", s.handleCreateGlossaryTerm)
	mux.HandleFunc("DELETE /api/glossary/{id}", s.handleDeleteGlossaryTerm)

	// Tags & Search
	mux.HandleFunc("GET /api/tags", s.handleGetTags)
	mux.HandleFunc("GET /api/search", s.handleSearch)

	// Static assets: lazily loaded Mermaid runtime and robots.txt
	if mermaid, err := web.GetMermaidRuntime(); err == nil {
		mux.HandleFunc("GET "+mermaid.AssetPath(), s.handleMermaidJS)
	} else {
		log.Printf("[WARN] Mermaid runtime unavailable, diagrams will not render: %v", err)
	}
	mux.HandleFunc("GET /robots.txt", s.handleRobots)

	// Embedded Web UI Handler
	mux.HandleFunc("GET /", s.handleStaticUI)

	s.handler = s.wrapMiddleware(mux)

	return s
}

// maxRequestBody caps JSON request bodies accepted by the REST API.
const maxRequestBody = 1 << 20

func (s *Server) wrapMiddleware(next http.Handler) http.Handler {
	cspHeader := s.buildCSP()
	security := s.cfg.Server.Security

	// Reject cross-site state-changing requests (CSRF). Browsers always send
	// Sec-Fetch-Site or Origin; configured CORS origins remain trusted.
	cop := http.NewCrossOriginProtection()
	for _, o := range security.CORSAllowedOrigins {
		if o == "*" {
			continue
		}
		if err := cop.AddTrustedOrigin(o); err != nil {
			log.Printf("[WARN] ignoring invalid CORS origin %q: %v", o, err)
		}
	}
	cop.SetDenyHandler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeError(w, http.StatusForbidden, "cross-origin request rejected")
	}))
	protected := cop.Handler(next)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Panic recovery
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("[PANIC] %s %s: %v", r.Method, r.URL.Path, rec)
				writeError(w, http.StatusInternalServerError, "internal server error")
			}
		}()

		// Security headers
		if cspHeader != "" {
			w.Header().Set("Content-Security-Policy", cspHeader)
		}
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")

		// DNS rebinding protection: a connection to a loopback address must
		// carry a loopback (or explicitly configured) Host header.
		if !s.isAllowedHost(r) {
			writeError(w, http.StatusForbidden, "host not allowed")
			return
		}

		// CORS Handling
		if security.CORSEnabled {
			origin := r.Header.Get("Origin")
			if origin != "" {
				allowed := len(security.CORSAllowedOrigins) == 0 ||
					slices.Contains(security.CORSAllowedOrigins, "*") ||
					slices.Contains(security.CORSAllowedOrigins, origin)
				if allowed {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
					w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
					w.Header().Add("Vary", "Origin")
				}
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
		}

		// The REST API only accepts JSON bodies. Rejecting other declared content
		// types blocks HTML-form and text/plain "simple request" CSRF vectors.
		if (r.Method == http.MethodPost || r.Method == http.MethodPut) &&
			strings.HasPrefix(r.URL.Path, "/api/") && r.URL.Path != "/api/mcp" {
			if ct := r.Header.Get("Content-Type"); ct != "" {
				if mt, _, err := mime.ParseMediaType(ct); err != nil || mt != "application/json" {
					writeError(w, http.StatusUnsupportedMediaType, "Content-Type must be application/json")
					return
				}
			}
			r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)
		}

		protected.ServeHTTP(w, r)
	})
}

// isAllowedHost implements DNS rebinding protection. Requests that arrived on a
// loopback interface must name a loopback host or the configured server host.
func (s *Server) isAllowedHost(r *http.Request) bool {
	local, ok := r.Context().Value(http.LocalAddrContextKey).(net.Addr)
	if !ok {
		return true
	}
	tcp, ok := local.(*net.TCPAddr)
	if !ok || !tcp.IP.IsLoopback() {
		return true
	}

	host := r.Host
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	host = strings.Trim(host, "[]")
	if strings.EqualFold(host, "localhost") || strings.HasSuffix(strings.ToLower(host), ".localhost") {
		return true
	}
	if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
		return true
	}
	return host != "" && strings.EqualFold(host, s.cfg.Server.Host)
}

// decodeJSON decodes the request body into v, writing a 400 response on failure.
func decodeJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		status := http.StatusBadRequest
		if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
			status = http.StatusRequestEntityTooLarge
		}
		writeError(w, status, "invalid request body: "+err.Error())
		return false
	}
	return true
}

// writeServiceError maps service error kinds to HTTP status codes. Errors that
// carry a machine-readable code (service.ErrorCode) also return it as "code".
func writeServiceError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, service.ErrInvalid):
		status = http.StatusBadRequest
	case errors.Is(err, service.ErrNotFound):
		status = http.StatusNotFound
	case errors.Is(err, service.ErrConflict):
		status = http.StatusConflict
	}
	if code := service.ErrorCode(err); code != "" {
		writeJSON(w, status, map[string]string{"error": err.Error(), "code": code})
		return
	}
	writeError(w, status, err.Error())
}

// buildCSP renders the response policy. The lazily loaded Mermaid runtime carries an
// SRI integrity attribute; listing its hash lets CSP3 browsers allow exactly that file
// even without 'self'.
func (s *Server) buildCSP() string {
	var extra []string
	if mermaid, err := web.GetMermaidRuntime(); err == nil {
		extra = append(extra, "'"+mermaid.Integrity+"'")
	}
	return csp.Build(s.cfg.Server.Security.CSP, extra...)
}

func (s *Server) handleStaticUI(w http.ResponseWriter, r *http.Request) {
	// If path starts with /api/, return 404 (not handled by any API route)
	if strings.HasPrefix(r.URL.Path, "/api/") {
		writeError(w, http.StatusNotFound, "api endpoint not found")
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Add("Vary", "Accept-Encoding")

	// If client accepts gzip encoding, serve pre-compressed asset directly
	if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
		gzData, err := web.GetGzipHTML()
		if err == nil {
			w.Header().Set("Content-Encoding", "gzip")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(gzData)
			return
		}
	}

	// Fallback to uncompressed HTML
	data, err := web.GetHTML()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to read embedded web UI")
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

// Start launches the HTTP server listening on the configured address.
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.cfg.Server.Host, s.cfg.Server.Port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to bind server on %s: %w", addr, err)
	}

	s.listener = ln
	s.httpServer = &http.Server{
		Handler:      s.handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 0, // 0 for streaming SSE support
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		_ = s.httpServer.Serve(ln)
	}()

	return nil
}

// Shutdown gracefully stops the HTTP server and SSE hub.
func (s *Server) Shutdown(ctx context.Context) error {
	if s.sseHub != nil {
		s.sseHub.Stop()
	}
	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}

// Close forcibly terminates the HTTP server and SSE hub immediately.
func (s *Server) Close() error {
	if s.sseHub != nil {
		s.sseHub.Stop()
	}
	if s.listener != nil {
		_ = s.listener.Close()
	}
	if s.httpServer != nil {
		return s.httpServer.Close()
	}
	return nil
}

// Addr returns the bound network address of the running server.
func (s *Server) Addr() string {
	if s.listener != nil {
		return s.listener.Addr().String()
	}
	return fmt.Sprintf("%s:%d", s.cfg.Server.Host, s.cfg.Server.Port)
}

// Port returns the bound TCP port of the running server.
func (s *Server) Port() int {
	if s.listener != nil {
		if tcpAddr, ok := s.listener.Addr().(*net.TCPAddr); ok {
			return tcpAddr.Port
		}
	}
	return s.cfg.Server.Port
}

// Handler returns the HTTP request handler (useful for testing).
func (s *Server) Handler() http.Handler {
	return s.handler
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{
		"error": message,
	})
}
