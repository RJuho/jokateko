package server

import (
	"net/http"

	"github.com/RJuho/jokateko/internal/model"
	"github.com/RJuho/jokateko/internal/service"
)

func writeDeleted(w http.ResponseWriter, id string) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted", "id": id})
}

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
	ms, err := s.svc.GetMilestone(r.Context(), r.PathValue("id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, ms)
}

func (s *Server) handleCreateMilestone(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID         string                `json:"id"`
		Title      string                `json:"title"`
		Status     model.MilestoneStatus `json:"status"`
		TargetDate string                `json:"target_date"`
		Tags       []string              `json:"tags"`
		Summary    string                `json:"summary"`
		Body       string                `json:"body"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}

	ms, err := s.svc.CreateMilestone(r.Context(), service.NewMilestone{
		ID:         req.ID,
		Title:      req.Title,
		Status:     req.Status,
		TargetDate: req.TargetDate,
		Tags:       req.Tags,
		Summary:    req.Summary,
		Body:       req.Body,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, ms)
}

func (s *Server) handleDeleteMilestone(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.svc.DeleteMilestone(r.Context(), id, r.URL.Query().Get("force") == "true"); err != nil {
		writeServiceError(w, err)
		return
	}
	writeDeleted(w, id)
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
	strat, err := s.svc.GetStrategy(r.Context(), r.PathValue("id"))
	if err != nil {
		writeServiceError(w, err)
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
	if !decodeJSON(w, r, &req) {
		return
	}

	strat, err := s.svc.CreateStrategy(r.Context(), service.NewStrategy{
		ID:      req.ID,
		Title:   req.Title,
		Tier:    req.Tier,
		Tags:    req.Tags,
		Summary: req.Summary,
		Body:    req.Body,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, strat)
}

func (s *Server) handleDeleteStrategy(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.svc.DeleteStrategy(r.Context(), id); err != nil {
		writeServiceError(w, err)
		return
	}
	writeDeleted(w, id)
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
	term, err := s.svc.GetGlossaryTerm(r.Context(), r.PathValue("id"))
	if err != nil {
		writeServiceError(w, err)
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
	if !decodeJSON(w, r, &req) {
		return
	}

	term, err := s.svc.CreateGlossaryTerm(r.Context(), service.NewGlossaryTerm{
		ID:      req.ID,
		Title:   req.Title,
		Tags:    req.Tags,
		Summary: req.Summary,
		Body:    req.Body,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, term)
}

func (s *Server) handleDeleteGlossaryTerm(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.svc.DeleteGlossaryTerm(r.Context(), id); err != nil {
		writeServiceError(w, err)
		return
	}
	writeDeleted(w, id)
}
