package server

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/RJuho/jokateko/internal/config"
	"github.com/RJuho/jokateko/internal/model"
	"github.com/RJuho/jokateko/internal/store"
	"github.com/RJuho/jokateko/internal/version"
	"github.com/RJuho/jokateko/internal/writer"
	"github.com/RJuho/jokateko/web"
)

// Server encapsulates the HTTP daemon, REST API endpoints, and live SSE event hub.
type Server struct {
	cfg          *config.Config
	workspaceDir string
	store        *store.Store
	writer       *writer.Writer
	sseHub       *SSEHub
	columns      []model.Column
	startTime    time.Time
	httpServer   *http.Server
	listener     net.Listener
	handler      http.Handler
	mcpHandler   http.Handler
}

// SetMCPHandler configures an optional Model Context Protocol HTTP/SSE handler mounted at /api/mcp.
func (s *Server) SetMCPHandler(h http.Handler) {
	s.mcpHandler = h
}

// New creates and configures a new Server instance.
func New(cfg *config.Config, workspaceDir string, st *store.Store, wr *writer.Writer, sse *SSEHub) *Server {
	if cfg == nil {
		cfg = config.Default(workspaceDir)
	}

	cols := make([]model.Column, 0, len(cfg.Board.Columns))
	for _, c := range cfg.Board.Columns {
		cols = append(cols, model.Column{
			ID:           c.ID,
			Name:         c.Name,
			Color:        c.Color,
			HandledBy:    c.HandledBy,
			Instructions: c.Instructions,
		})
	}

	s := &Server{
		cfg:          cfg,
		workspaceDir: workspaceDir,
		store:        st,
		writer:       wr,
		sseHub:       sse,
		columns:      cols,
		startTime:    time.Now(),
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

	// Embedded Web UI Handler
	mux.HandleFunc("GET /", s.handleStaticUI)

	s.handler = s.wrapMiddleware(mux)

	return s
}

func (s *Server) wrapMiddleware(next http.Handler) http.Handler {
	cspHeader := s.buildCSP()

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

		// CORS Handling
		if s.cfg.Server.Security.CORSEnabled {
			origin := r.Header.Get("Origin")
			if origin != "" {
				allowed := false
				if len(s.cfg.Server.Security.CORSAllowedOrigins) == 0 {
					allowed = true
				} else {
					for _, o := range s.cfg.Server.Security.CORSAllowedOrigins {
						if o == "*" || o == origin {
							allowed = true
							break
						}
					}
				}
				if allowed {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
					w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
				}
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

func extractTagSHA256(html []byte, tag string) string {
	openTag := []byte("<" + tag + ">")
	closeTag := []byte("</" + tag + ">")

	start := bytes.Index(html, openTag)
	if start == -1 {
		return ""
	}
	start += len(openTag)
	end := bytes.Index(html[start:], closeTag)
	if end == -1 {
		return ""
	}
	body := html[start : start+end]
	sum := sha256.Sum256(body)
	return fmt.Sprintf("'sha256-%s'", base64.StdEncoding.EncodeToString(sum[:]))
}

var (
	cachedScriptHash string
	cachedStyleHash  string
	assetHashesOnce  sync.Once
)

func getWebAssetHashes() (string, string) {
	assetHashesOnce.Do(func() {
		scriptH := version.ScriptHash
		styleH := version.StyleHash

		if scriptH == "" || styleH == "" {
			htmlBytes, err := web.GetHTML()
			if err == nil {
				if scriptH == "" {
					scriptH = extractTagSHA256(htmlBytes, "script")
				}
				if styleH == "" {
					styleH = extractTagSHA256(htmlBytes, "style")
				}
			}
		}

		if scriptH != "" && !strings.HasPrefix(scriptH, "'") {
			scriptH = fmt.Sprintf("'%s'", scriptH)
		}
		if styleH != "" && !strings.HasPrefix(styleH, "'") {
			styleH = fmt.Sprintf("'%s'", styleH)
		}

		cachedScriptHash = scriptH
		cachedStyleHash = styleH
	})

	return cachedScriptHash, cachedStyleHash
}

func (s *Server) buildCSP() string {
	csp := s.cfg.Server.Security.CSP
	if !csp.Enabled {
		return ""
	}

	scriptHash, styleHash := getWebAssetHashes()

	scriptSrc := append([]string(nil), csp.ScriptSrc...)
	if scriptHash != "" && !slices.Contains(scriptSrc, scriptHash) {
		scriptSrc = append(scriptSrc, scriptHash)
	}

	styleSrc := append([]string(nil), csp.StyleSrc...)
	if styleHash != "" && !slices.Contains(styleSrc, styleHash) {
		styleSrc = append(styleSrc, styleHash)
	}

	var parts []string
	addDirective := func(name string, values []string) {
		if len(values) > 0 {
			parts = append(parts, fmt.Sprintf("%s %s", name, strings.Join(values, " ")))
		}
	}

	addDirective("default-src", csp.DefaultSrc)
	addDirective("script-src", scriptSrc)
	addDirective("style-src", styleSrc)
	addDirective("style-src-elem", csp.StyleSrcElem)
	addDirective("style-src-attr", csp.StyleSrcAttr)
	addDirective("img-src", csp.ImgSrc)
	addDirective("connect-src", csp.ConnectSrc)
	addDirective("font-src", csp.FontSrc)

	return strings.Join(parts, "; ")
}

func (s *Server) handleStaticUI(w http.ResponseWriter, r *http.Request) {
	// If path starts with /api/, return 404 (not handled by any API route)
	if strings.HasPrefix(r.URL.Path, "/api/") {
		writeError(w, http.StatusNotFound, "api endpoint not found")
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

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
