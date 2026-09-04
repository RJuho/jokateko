package server

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/RJuho/jokateko/internal/model"
	"github.com/RJuho/jokateko/internal/parser"
	"github.com/RJuho/jokateko/internal/store"
	"github.com/RJuho/jokateko/internal/validator"
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
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "task id is required")
		return
	}

	task, err := s.store.GetTask(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "task not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get task: "+err.Error())
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
		Body         string         `json:"body"`
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

	summary := strings.TrimSpace(req.Summary)
	if summary == "" {
		writeError(w, http.StatusBadRequest, "summary is required")
		return
	}

	id := strings.TrimSpace(req.ID)
	if id == "" {
		// Auto-generate date-based slug from title
		slug := slugify(title)
		id = fmt.Sprintf("%s-%s", time.Now().Format("060102"), slug)
	}

	status := req.Status
	if status == "" {
		if len(s.columns) > 0 {
			status = s.columns[0].ID
		} else {
			status = "ready"
		}
	}

	priority := req.Priority
	if !priority.IsValid() {
		priority = model.PriorityMedium
	}

	tags := req.Tags
	if tags == nil {
		tags = []string{}
	}

	deps := req.Dependencies
	if deps == nil {
		deps = []string{}
	}

	fm := model.TaskFrontmatter{
		Title:        title,
		Status:       status,
		Priority:     priority,
		Milestone:    req.Milestone,
		Tags:         tags,
		Summary:      summary,
		Dependencies: deps,
	}

	fileBytes, err := parser.Format(fm, req.Body)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to format task markdown: "+err.Error())
		return
	}

	tasksDir := s.cfg.Paths.Tasks
	if !filepath.IsAbs(tasksDir) {
		tasksDir = filepath.Join(s.workspaceDir, tasksDir)
	}
	filePath := filepath.Join(tasksDir, fmt.Sprintf("%s.md", id))

	if err := s.writer.WriteFile(filePath, fileBytes, 0644); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to write task file: "+err.Error())
		return
	}

	total, completed, _ := parser.ExtractAcceptanceCriteria([]byte(req.Body))
	bodyHTML, _ := parser.RenderHTML([]byte(req.Body))
	task := model.Task{
		ID:                id,
		Title:             title,
		Status:            status,
		Priority:          priority,
		Milestone:         req.Milestone,
		Tags:              tags,
		Summary:           summary,
		Dependencies:      deps,
		Body:              req.Body,
		BodyHTML:          bodyHTML,
		TotalCriteria:     total,
		CompletedCriteria: completed,
		FilePath:          filePath,
		ModTime:           time.Now(),
	}

	if err := s.store.UpsertTask(r.Context(), task); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to index task in store: "+err.Error())
		return
	}

	s.sseHub.Broadcast("task.created", task)
	writeJSON(w, http.StatusCreated, task)
}

func (s *Server) handleUpdateTask(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "task id is required")
		return
	}

	existing, err := s.store.GetTask(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "task not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get task: "+err.Error())
		return
	}

	var req struct {
		Title        *string         `json:"title"`
		Status       *string         `json:"status"`
		Priority     *model.Priority `json:"priority"`
		Milestone    *string         `json:"milestone"`
		Tags         *[]string       `json:"tags"`
		Summary      *string         `json:"summary"`
		Dependencies *[]string       `json:"dependencies"`
		Body         *string         `json:"body"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	if req.Title != nil {
		existing.Title = strings.TrimSpace(*req.Title)
	}
	if req.Status != nil {
		existing.Status = strings.TrimSpace(*req.Status)
	}
	if req.Priority != nil && req.Priority.IsValid() {
		existing.Priority = *req.Priority
	}
	if req.Milestone != nil {
		existing.Milestone = strings.TrimSpace(*req.Milestone)
	}
	if req.Tags != nil {
		existing.Tags = *req.Tags
	}
	if req.Summary != nil {
		existing.Summary = strings.TrimSpace(*req.Summary)
	}
	if req.Dependencies != nil {
		existing.Dependencies = *req.Dependencies
	}
	if req.Body != nil {
		existing.Body = *req.Body
	}

	fm := model.TaskFrontmatter{
		Title:        existing.Title,
		Status:       existing.Status,
		Priority:     existing.Priority,
		Milestone:    existing.Milestone,
		Tags:         existing.Tags,
		Summary:      existing.Summary,
		Dependencies: existing.Dependencies,
	}

	fileBytes, err := parser.Format(fm, existing.Body)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to format task markdown: "+err.Error())
		return
	}

	filePath := existing.FilePath
	if filePath == "" {
		tasksDir := s.cfg.Paths.Tasks
		if !filepath.IsAbs(tasksDir) {
			tasksDir = filepath.Join(s.workspaceDir, tasksDir)
		}
		filePath = filepath.Join(tasksDir, fmt.Sprintf("%s.md", id))
		existing.FilePath = filePath
	}

	if err := s.writer.WriteFile(filePath, fileBytes, 0644); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to write task file: "+err.Error())
		return
	}

	total, completed, _ := parser.ExtractAcceptanceCriteria([]byte(existing.Body))
	existing.TotalCriteria = total
	existing.CompletedCriteria = completed
	existing.ModTime = time.Now()
	bodyHTML, _ := parser.RenderHTML([]byte(existing.Body))
	existing.BodyHTML = bodyHTML

	if err := s.store.UpsertTask(r.Context(), existing); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update task in store: "+err.Error())
		return
	}

	s.sseHub.Broadcast("task.updated", existing)
	writeJSON(w, http.StatusOK, existing)
}

func (s *Server) handleDeleteTask(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "task id is required")
		return
	}

	existing, err := s.store.GetTask(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "task not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get task: "+err.Error())
		return
	}

	force := r.URL.Query().Get("force") == "true"
	if !force {
		downstream, err := s.store.GetDownstreamTasks(r.Context(), id)
		if err == nil && len(downstream) > 0 {
			downstreamIDs := make([]string, len(downstream))
			for i, d := range downstream {
				downstreamIDs[i] = d.ID
			}
			writeError(w, http.StatusConflict, fmt.Sprintf("cannot delete task %q: %d task(s) depend on it (%s). Set ?force=true to delete anyway", id, len(downstream), strings.Join(downstreamIDs, ", ")))
			return
		}
	}

	if existing.FilePath != "" {
		if err := s.writer.RemoveFile(existing.FilePath); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to remove task file: "+err.Error())
			return
		}
	}

	if err := s.store.DeleteTask(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete task from store: "+err.Error())
		return
	}

	s.sseHub.Broadcast("task.deleted", map[string]string{"id": id})
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted", "id": id})
}

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var sb strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			sb.WriteRune(r)
		} else if r == ' ' || r == '-' || r == '_' {
			if sb.Len() > 0 && sb.String()[sb.Len()-1] != '-' {
				sb.WriteByte('-')
			}
		}
	}
	res := strings.Trim(sb.String(), "-")
	if res == "" {
		return "task"
	}
	if len(res) > 40 {
		return res[:40]
	}
	return res
}

func (s *Server) handleAddTaskDependency(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "task id is required")
		return
	}

	var req struct {
		DependencyID string `json:"dependency_id"`
		ID           string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	depID := strings.TrimSpace(cmp.Or(req.DependencyID, req.ID))
	if depID == "" {
		writeError(w, http.StatusBadRequest, "dependency_id is required")
		return
	}

	if id == depID {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("task %q cannot depend on itself", id))
		return
	}

	existing, err := s.store.GetTask(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, fmt.Sprintf("task %q not found", id))
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get task: "+err.Error())
		return
	}

	if _, err := s.store.GetTask(r.Context(), depID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, fmt.Sprintf("dependency task %q not found", depID))
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get dependency task: "+err.Error())
		return
	}

	if slices.Contains(existing.Dependencies, depID) {
		writeJSON(w, http.StatusOK, existing)
		return
	}

	tasks, err := s.store.ListTasks(r.Context(), model.FilterCriteria{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list tasks for cycle check: "+err.Error())
		return
	}

	depsGraph := make(map[string][]string, len(tasks)+1)
	taskFiles := make(map[string]string, len(tasks)+1)
	for _, t := range tasks {
		depsGraph[t.ID] = t.Dependencies
		taskFiles[t.ID] = t.FilePath
	}
	depsGraph[id] = append(slices.Clone(existing.Dependencies), depID)

	cycleDiags := validator.DetectCycles(depsGraph, taskFiles)
	if len(cycleDiags) > 0 {
		var msgs []string
		for _, d := range cycleDiags {
			if len(d.Context) > 0 {
				msgs = append(msgs, d.Context...)
			} else {
				msgs = append(msgs, d.Message)
			}
		}
		writeError(w, http.StatusConflict, fmt.Sprintf("circular dependency detected: %s", strings.Join(msgs, "; ")))
		return
	}

	existing.Dependencies = append(existing.Dependencies, depID)
	existing.ModTime = time.Now()

	fm := model.TaskFrontmatter{
		Title:        existing.Title,
		Status:       existing.Status,
		Priority:     existing.Priority,
		Milestone:    existing.Milestone,
		Tags:         existing.Tags,
		Summary:      existing.Summary,
		Dependencies: existing.Dependencies,
	}

	fileBytes, err := parser.Format(fm, existing.Body)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to format task markdown: "+err.Error())
		return
	}

	filePath := existing.FilePath
	if filePath == "" {
		tasksDir := s.cfg.Paths.Tasks
		if !filepath.IsAbs(tasksDir) {
			tasksDir = filepath.Join(s.workspaceDir, tasksDir)
		}
		filePath = filepath.Join(tasksDir, fmt.Sprintf("%s.md", id))
		existing.FilePath = filePath
	}

	if err := s.writer.WriteFile(filePath, fileBytes, 0644); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to write task file: "+err.Error())
		return
	}

	if err := s.store.UpsertTask(r.Context(), existing); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update task in store: "+err.Error())
		return
	}

	s.sseHub.Broadcast("task.updated", existing)
	writeJSON(w, http.StatusOK, existing)
}

func (s *Server) handleRemoveTaskDependency(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "task id is required")
		return
	}

	depID := r.PathValue("depId")
	if depID == "" {
		writeError(w, http.StatusBadRequest, "depId is required")
		return
	}

	existing, err := s.store.GetTask(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, fmt.Sprintf("task %q not found", id))
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get task: "+err.Error())
		return
	}

	idx := slices.Index(existing.Dependencies, depID)
	if idx == -1 {
		writeError(w, http.StatusNotFound, fmt.Sprintf("dependency %q not found on task %q", depID, id))
		return
	}

	existing.Dependencies = slices.Delete(existing.Dependencies, idx, idx+1)
	existing.ModTime = time.Now()

	fm := model.TaskFrontmatter{
		Title:        existing.Title,
		Status:       existing.Status,
		Priority:     existing.Priority,
		Milestone:    existing.Milestone,
		Tags:         existing.Tags,
		Summary:      existing.Summary,
		Dependencies: existing.Dependencies,
	}

	fileBytes, err := parser.Format(fm, existing.Body)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to format task markdown: "+err.Error())
		return
	}

	filePath := existing.FilePath
	if filePath == "" {
		tasksDir := s.cfg.Paths.Tasks
		if !filepath.IsAbs(tasksDir) {
			tasksDir = filepath.Join(s.workspaceDir, tasksDir)
		}
		filePath = filepath.Join(tasksDir, fmt.Sprintf("%s.md", id))
		existing.FilePath = filePath
	}

	if err := s.writer.WriteFile(filePath, fileBytes, 0644); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to write task file: "+err.Error())
		return
	}

	if err := s.store.UpsertTask(r.Context(), existing); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update task in store: "+err.Error())
		return
	}

	s.sseHub.Broadcast("task.updated", existing)
	writeJSON(w, http.StatusOK, existing)
}

