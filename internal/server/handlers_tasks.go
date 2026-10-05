package server

import (
	"cmp"
	"net/http"
	"strings"

	"github.com/RJuho/jokateko/internal/model"
	"github.com/RJuho/jokateko/internal/service"
)

func (s *Server) handleListTasks(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filter := model.FilterCriteria{
		Status:      q.Get("status"),
		Milestone:   q.Get("milestone"),
		Tag:         q.Get("tag"),
		Priority:    model.Priority(q.Get("priority")),
		SearchQuery: q.Get("q"),
	}

	tasks, err := s.store.ListTasks(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list tasks: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, tasks)
}

func (s *Server) handleGetTask(w http.ResponseWriter, r *http.Request) {
	task, err := s.svc.GetTask(r.Context(), r.PathValue("id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, task)
}

func (s *Server) handleCreateTask(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID              string         `json:"id"`
		Title           string         `json:"title"`
		Status          string         `json:"status"`
		Priority        model.Priority `json:"priority"`
		Milestone       string         `json:"milestone"`
		Tags            []string       `json:"tags"`
		Summary         string         `json:"summary"`
		Dependencies    []string       `json:"dependencies"`
		TargetAt        string         `json:"target_at"`
		Body            string         `json:"body"`
		ReopenMilestone bool           `json:"reopen_milestone"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}

	task, err := s.svc.CreateTask(r.Context(), service.NewTask{
		ID:              req.ID,
		Title:           req.Title,
		Status:          req.Status,
		Priority:        req.Priority,
		Milestone:       req.Milestone,
		Tags:            req.Tags,
		Summary:         req.Summary,
		Dependencies:    req.Dependencies,
		TargetAt:        req.TargetAt,
		Body:            req.Body,
		ReopenMilestone: req.ReopenMilestone,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, task)
}

func (s *Server) handleUpdateTask(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title           *string         `json:"title"`
		Status          *string         `json:"status"`
		Priority        *model.Priority `json:"priority"`
		Milestone       *string         `json:"milestone"`
		Tags            *[]string       `json:"tags"`
		Summary         *string         `json:"summary"`
		Dependencies    *[]string       `json:"dependencies"`
		TargetAt        *string         `json:"target_at"`
		Body            *string         `json:"body"`
		ReopenMilestone bool            `json:"reopen_milestone"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}

	task, err := s.svc.PatchTask(r.Context(), r.PathValue("id"), service.TaskPatch{
		Title:           req.Title,
		Status:          req.Status,
		Priority:        req.Priority,
		Milestone:       req.Milestone,
		Tags:            req.Tags,
		Summary:         req.Summary,
		Dependencies:    req.Dependencies,
		TargetAt:        req.TargetAt,
		Body:            req.Body,
		ReopenMilestone: req.ReopenMilestone,
		BodyHint:        "use POST /api/tasks/{id}/notes to append notes",
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, task)
}

func (s *Server) handleDeleteTask(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	force := r.URL.Query().Get("force") == "true"
	if err := s.svc.DeleteTask(r.Context(), id, force); err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted", "id": id})
}

func (s *Server) handleAddTaskDependency(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DependencyID string `json:"dependency_id"`
		ID           string `json:"id"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}

	depID := strings.TrimSpace(cmp.Or(req.DependencyID, req.ID))
	if depID == "" {
		writeError(w, http.StatusBadRequest, "dependency_id is required")
		return
	}

	task, _, err := s.svc.AddTaskDependency(r.Context(), r.PathValue("id"), depID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, task)
}

func (s *Server) handleRemoveTaskDependency(w http.ResponseWriter, r *http.Request) {
	task, err := s.svc.RemoveTaskDependency(r.Context(), r.PathValue("id"), r.PathValue("depId"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, task)
}

func (s *Server) handleAddTaskNote(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Note string `json:"note"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}

	task, err := s.svc.AddTaskNote(r.Context(), r.PathValue("id"), req.Note)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, task)
}

func (s *Server) handleUpdateTaskStatus(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Status string `json:"status"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}

	task, err := s.svc.UpdateTaskStatus(r.Context(), r.PathValue("id"), req.Status)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, task)
}
