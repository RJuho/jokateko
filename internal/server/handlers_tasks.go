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
		ID           string         `json:"id"`
		Title        string         `json:"title"`
		Status       string         `json:"status"`
		Priority     model.Priority `json:"priority"`
		Milestone    string         `json:"milestone"`
		Tags         []string       `json:"tags"`
		Summary      string         `json:"summary"`
		Dependencies []string       `json:"dependencies"`
		TargetAt     string         `json:"target_at"`
		Body         string         `json:"body"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}

	task, err := s.svc.CreateTask(r.Context(), service.NewTask{
		ID:           req.ID,
		Title:        req.Title,
		Status:       req.Status,
		Priority:     req.Priority,
		Milestone:    req.Milestone,
		Tags:         req.Tags,
		Summary:      req.Summary,
		Dependencies: req.Dependencies,
		TargetAt:     req.TargetAt,
		Body:         req.Body,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, task)
}

func (s *Server) handleUpdateTask(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title        *string         `json:"title"`
		Status       *string         `json:"status"`
		Priority     *model.Priority `json:"priority"`
		Milestone    *string         `json:"milestone"`
		Tags         *[]string       `json:"tags"`
		Summary      *string         `json:"summary"`
		Dependencies *[]string       `json:"dependencies"`
		TargetAt     *string         `json:"target_at"`
		Body         *string         `json:"body"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}

	task, err := s.svc.UpdateTask(r.Context(), r.PathValue("id"), func(t *model.Task) error {
		if req.Title != nil {
			t.Title = strings.TrimSpace(*req.Title)
		}
		if req.Status != nil {
			status := strings.TrimSpace(*req.Status)
			if err := s.svc.CheckStatus(status); err != nil {
				return err
			}
			t.Status = status
		}
		if req.Priority != nil && req.Priority.IsValid() {
			t.Priority = *req.Priority
		}
		if req.Milestone != nil {
			t.Milestone = strings.TrimSpace(*req.Milestone)
		}
		if req.Tags != nil {
			t.Tags = *req.Tags
		}
		if req.Summary != nil {
			t.Summary = strings.TrimSpace(*req.Summary)
		}
		if req.Dependencies != nil {
			t.Dependencies = *req.Dependencies
		}
		if req.TargetAt != nil {
			targetAt, err := service.NormalizeTargetAt(*req.TargetAt)
			if err != nil {
				return err
			}
			t.TargetAt = targetAt
		}
		if req.Body != nil {
			if err := s.svc.CheckBodyEdit(t.Status, t.Body, *req.Body, "use POST /api/tasks/{id}/notes to append notes"); err != nil {
				return err
			}
			t.Body = *req.Body
		}
		return nil
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
