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
