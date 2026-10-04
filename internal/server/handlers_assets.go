package server

import (
	"net/http"
	"strings"

	"github.com/RJuho/jokateko/web"
)

// robotsTxt allows crawling; Jokateko is a local tool, but an HTML fallback for
// /robots.txt is invalid and confuses crawlers and audits.
const robotsTxt = "User-agent: *\nAllow: /\n"

func (s *Server) handleRobots(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte(robotsTxt))
}

// handleMermaidJS serves the embedded Mermaid runtime that the UI loads lazily
// (with SRI) only when a diagram is shown. The path is versioned, so the
// response can be cached immutably.
func (s *Server) handleMermaidJS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	w.Header().Set("Vary", "Accept-Encoding")

	if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
		if gzData, err := web.GetGzipMermaidJS(); err == nil {
			w.Header().Set("Content-Encoding", "gzip")
			_, _ = w.Write(gzData)
			return
		}
	}

	data, err := web.GetMermaidJS()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to read embedded mermaid runtime")
		return
	}
	_, _ = w.Write(data)
}
