package server

import (
	"net/http"
	"time"

	"github.com/RJuho/jokateko/internal/version"
)

type HealthResponse struct {
	Status        string `json:"status"`
	Project       string `json:"project"`
	Version       string `json:"version"`
	Commit        string `json:"commit"`
	UptimeSeconds int64  `json:"uptime_seconds"`
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	info := version.Get()
	resp := HealthResponse{
		Status:        "ok",
		Project:       s.cfg.Project.Name,
		Version:       info.Version,
		Commit:        info.Commit,
		UptimeSeconds: int64(time.Since(s.startTime).Seconds()),
	}

	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleVersion(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, version.Get())
}

func (s *Server) handleAbout(w http.ResponseWriter, _ *http.Request) {
	info := version.Get()
	report := version.GetLicenses()
	resp := map[string]any{
		"name":         "Jokateko",
		"description":  "Local, Markdown-driven Kanban and task management for developers and AI agents",
		"repository":   "https://github.com/RJuho/jokateko",
		"license":      report.Project.License,
		"license_text": report.Project.Text,
		"build":        info,
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleLicenses(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, version.GetLicenses())
}
