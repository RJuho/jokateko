package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/RJuho/jokateko/internal/model"
	"github.com/RJuho/jokateko/internal/parser"
	"github.com/RJuho/jokateko/internal/store"
)

// --- Milestones ---

func (s *Server) handleListMilestones(w http.ResponseWriter, r *http.Request) {
	ms, err := s.store.ListMilestones(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list milestones: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, ms)
}

func (s *Server) handleGetMilestone(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ms, err := s.store.GetMilestone(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "milestone not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get milestone: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, ms)
}

func (s *Server) handleCreateMilestone(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID         string               `json:"id"`
		Title      string               `json:"title"`
		Status     model.MilestoneStatus `json:"status"`
		TargetDate string               `json:"target_date"`
		Tags       []string             `json:"tags"`
		Summary    string               `json:"summary"`
		Body       string               `json:"body"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	title := strings.TrimSpace(req.Title)
	if title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}

	id := strings.TrimSpace(req.ID)
	if id == "" {
		id = fmt.Sprintf("%s-%s", time.Now().Format("060102"), slugify(title))
	}

	status := req.Status
	if status == "" {
		status = model.MilestoneStatusOpen
	}

	tags := req.Tags
	if tags == nil {
		tags = []string{}
	}

	fm := model.MilestoneFrontmatter{
		Title:      title,
		Status:     status,
		TargetDate: req.TargetDate,
		Tags:       tags,
		Summary:    req.Summary,
	}

	fileBytes, err := parser.Format(fm, req.Body)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to format milestone: "+err.Error())
		return
	}

	msDir := s.cfg.Paths.Milestones
	if !filepath.IsAbs(msDir) {
		msDir = filepath.Join(s.workspaceDir, msDir)
	}
	filePath := filepath.Join(msDir, fmt.Sprintf("%s.md", id))

	if err := s.writer.WriteFile(filePath, fileBytes, 0644); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to write milestone file: "+err.Error())
		return
	}

	msBodyHTML, _ := parser.RenderHTML([]byte(req.Body))
	ms := model.Milestone{
		ID:         id,
		Title:      title,
		Status:     status,
		TargetDate: req.TargetDate,
		Tags:       tags,
		Summary:    req.Summary,
		Body:       req.Body,
		BodyHTML:   msBodyHTML,
		FilePath:   filePath,
		ModTime:    time.Now(),
	}

	if err := s.store.UpsertMilestone(r.Context(), ms); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to index milestone: "+err.Error())
		return
	}

	s.sseHub.Broadcast("milestone.created", ms)
	writeJSON(w, http.StatusCreated, ms)
}

func (s *Server) handleDeleteMilestone(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ms, err := s.store.GetMilestone(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "milestone not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get milestone: "+err.Error())
		return
	}

	force := r.URL.Query().Get("force") == "true"
	if !force && ms.TotalTasks > 0 {
		writeError(w, http.StatusConflict, fmt.Sprintf("cannot delete milestone %q: %d task(s) are assigned to it. Set ?force=true to delete anyway", id, ms.TotalTasks))
		return
	}

	if ms.FilePath != "" {
		_ = s.writer.RemoveFile(ms.FilePath)
	}

	if err := s.store.DeleteMilestone(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete milestone: "+err.Error())
		return
	}

	s.sseHub.Broadcast("milestone.deleted", map[string]string{"id": id})
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted", "id": id})
}

// --- Strategies ---

func (s *Server) handleListStrategies(w http.ResponseWriter, r *http.Request) {
	strats, err := s.store.ListStrategies(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list strategies: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, strats)
}

func (s *Server) handleGetStrategy(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	strat, err := s.store.GetStrategy(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "strategy not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get strategy: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, strat)
}

func (s *Server) handleCreateStrategy(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID      string     `json:"id"`
		Title   string     `json:"title"`
		Tier    model.Tier `json:"tier"`
		Tags    []string   `json:"tags"`
		Summary string     `json:"summary"`
		Body    string     `json:"body"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	title := strings.TrimSpace(req.Title)
	if title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}

	id := strings.TrimSpace(req.ID)
	if id == "" {
		id = slugify(title)
	}

	tier := req.Tier
	if !tier.IsValid() {
		tier = model.TierCore
	}

	tags := req.Tags
	if tags == nil {
		tags = []string{}
	}

	fm := model.StrategyFrontmatter{
		Title:   title,
		Tier:    tier,
		Tags:    tags,
		Summary: req.Summary,
	}

	fileBytes, err := parser.Format(fm, req.Body)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to format strategy: "+err.Error())
		return
	}

	stratDir := s.cfg.Paths.Strategies
	if !filepath.IsAbs(stratDir) {
		stratDir = filepath.Join(s.workspaceDir, stratDir)
	}
	filePath := filepath.Join(stratDir, fmt.Sprintf("%s.md", id))

	if err := s.writer.WriteFile(filePath, fileBytes, 0644); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to write strategy file: "+err.Error())
		return
	}

	stratBodyHTML, _ := parser.RenderHTML([]byte(req.Body))
	strat := model.Strategy{
		ID:       id,
		Title:    title,
		Tier:     tier,
		Tags:     tags,
		Summary:  req.Summary,
		Body:     req.Body,
		BodyHTML: stratBodyHTML,
		FilePath: filePath,
		ModTime:  time.Now(),
	}

	if err := s.store.UpsertStrategy(r.Context(), strat); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to index strategy: "+err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, strat)
}

func (s *Server) handleDeleteStrategy(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	strat, err := s.store.GetStrategy(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "strategy not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get strategy: "+err.Error())
		return
	}

	if strat.FilePath != "" {
		_ = s.writer.RemoveFile(strat.FilePath)
	}

	if err := s.store.DeleteStrategy(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete strategy: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted", "id": id})
}

// --- Glossary ---

func (s *Server) handleListGlossary(w http.ResponseWriter, r *http.Request) {
	terms, err := s.store.ListGlossaryTerms(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list glossary terms: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, terms)
}

func (s *Server) handleGetGlossaryTerm(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	term, err := s.store.GetGlossaryTerm(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "glossary term not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get glossary term: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, term)
}

func (s *Server) handleCreateGlossaryTerm(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID      string   `json:"id"`
		Title   string   `json:"title"`
		Tags    []string `json:"tags"`
		Summary string   `json:"summary"`
		Body    string   `json:"body"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	title := strings.TrimSpace(req.Title)
	if title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}

	id := strings.TrimSpace(req.ID)
	if id == "" {
		id = slugify(title)
	}

	tags := req.Tags
	if tags == nil {
		tags = []string{}
	}

	fm := model.GlossaryFrontmatter{
		Title:   title,
		Tags:    tags,
		Summary: req.Summary,
	}

	fileBytes, err := parser.Format(fm, req.Body)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to format glossary term: "+err.Error())
		return
	}

	glossDir := s.cfg.Paths.Glossary
	if !filepath.IsAbs(glossDir) {
		glossDir = filepath.Join(s.workspaceDir, glossDir)
	}
	filePath := filepath.Join(glossDir, fmt.Sprintf("%s.md", id))

	if err := s.writer.WriteFile(filePath, fileBytes, 0644); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to write glossary file: "+err.Error())
		return
	}

	termBodyHTML, _ := parser.RenderHTML([]byte(req.Body))
	term := model.GlossaryTerm{
		ID:       id,
		Title:    title,
		Tags:     tags,
		Summary:  req.Summary,
		Body:     req.Body,
		BodyHTML: termBodyHTML,
		FilePath: filePath,
		ModTime:  time.Now(),
	}

	if err := s.store.UpsertGlossaryTerm(r.Context(), term); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to index glossary term: "+err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, term)
}

func (s *Server) handleDeleteGlossaryTerm(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	term, err := s.store.GetGlossaryTerm(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "glossary term not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get glossary term: "+err.Error())
		return
	}

	if term.FilePath != "" {
		_ = s.writer.RemoveFile(term.FilePath)
	}

	if err := s.store.DeleteGlossaryTerm(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete glossary term: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted", "id": id})
}
