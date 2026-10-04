package service

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/RJuho/jokateko/internal/model"
	"github.com/RJuho/jokateko/internal/parser"
	"github.com/RJuho/jokateko/internal/store"
	"github.com/RJuho/jokateko/internal/validator"
)

// GetTask returns the task with the given ID.
func (s *Service) GetTask(ctx context.Context, id string) (model.Task, error) {
	t, err := s.store.GetTask(ctx, id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return model.Task{}, notFoundf("task %q not found", id)
		}
		return model.Task{}, fmt.Errorf("failed to get task %q: %w", id, err)
	}
	return t, nil
}

// saveTaskLocked stamps, persists, indexes and announces t. Callers hold s.mu.
func (s *Service) saveTaskLocked(ctx context.Context, t *model.Task, event string) error {
	t.FilePath = entityPath(s.dirs.Tasks, t.FilePath, t.ID)
	if t.CreatedAt == "" {
		t.CreatedAt = parser.DeriveFallbackCreatedAt(t.ID, t.ModTime)
	}
	t.ChangedAt = s.timestamp()
	t.Tags = nonNil(t.Tags)
	t.Dependencies = nonNil(t.Dependencies)

	data, err := parser.Format(t.Frontmatter(), t.Body)
	if err != nil {
		return fmt.Errorf("failed to format task markdown: %w", err)
	}
	if err := s.writer.WriteFile(t.FilePath, data, 0o644); err != nil {
		return fmt.Errorf("failed to save task file: %w", err)
	}

	t.TotalCriteria, t.CompletedCriteria, _ = parser.ExtractAcceptanceCriteria([]byte(t.Body))
	t.BodyHTML, _ = parser.RenderHTML([]byte(t.Body))
	t.ModTime = s.now()

	if err := s.store.UpsertTask(ctx, *t); err != nil {
		return fmt.Errorf("failed to index task %q: %w", t.ID, err)
	}
	s.notify(event, *t)
	return nil
}

// NewTask holds the fields for creating a task.
type NewTask struct {
	// ID is optional; when empty a "YYMMDD-title-slug" ID is generated.
	ID           string
	Title        string
	Status       string
	Priority     model.Priority
	Milestone    string
	Tags         []string
	Summary      string
	Dependencies []string
	// TargetAt accepts any format understood by parser.NormalizeTimestamp.
	TargetAt string
	Body     string
}

// CreateTask validates in, allocates a unique ID, and writes the new task.
func (s *Service) CreateTask(ctx context.Context, in NewTask) (model.Task, error) {
	title := strings.TrimSpace(in.Title)
	if title == "" {
		return model.Task{}, invalidf("title is required")
	}
	summary := strings.TrimSpace(in.Summary)
	if summary == "" {
		return model.Task{}, invalidf("summary is required")
	}
	targetAt, err := parser.NormalizeTimestamp(in.TargetAt)
	if err != nil {
		return model.Task{}, invalidf("invalid target_at format: %v", err)
	}

	status := strings.TrimSpace(in.Status)
	if status == "" {
		status = s.defaultTaskStatus()
	} else if err := s.CheckStatus(status); err != nil {
		return model.Task{}, err
	}

	priority := in.Priority
	if !priority.IsValid() {
		priority = model.PriorityMedium
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	id, err := allocateID(s.dirs.Tasks, strings.TrimSpace(in.ID), s.datedSlug(title, "task"), func(id string) bool {
		_, err := s.store.GetTask(ctx, id)
		return err == nil
	})
	if err != nil {
		return model.Task{}, err
	}

	now := s.timestamp()
	t := model.Task{
		ID:           id,
		Title:        title,
		Status:       status,
		Priority:     priority,
		Milestone:    strings.TrimSpace(in.Milestone),
		Tags:         in.Tags,
		Summary:      summary,
		Dependencies: in.Dependencies,
		Body:         in.Body,
		CreatedAt:    now,
		TargetAt:     targetAt,
	}
	if err := s.saveTaskLocked(ctx, &t, "task.created"); err != nil {
		return model.Task{}, err
	}
	return t, nil
}

func (s *Service) defaultTaskStatus() string {
	if st := strings.TrimSpace(s.cfg.Board.DefaultCreateState); st != "" {
		return st
	}
	if len(s.cfg.Board.Columns) > 0 {
		return s.cfg.Board.Columns[0].ID
	}
	return "backlog"
}

// CheckStatus rejects statuses that are not configured board columns.
// When no columns are configured, any status is accepted.
func (s *Service) CheckStatus(status string) error {
	if len(s.cfg.Board.Columns) > 0 && !s.cfg.HasColumn(status) {
		return invalidf("status %q is not a valid board column", status)
	}
	return nil
}

// UpdateTask loads a task, applies fn, and persists the result as "task.updated".
// If fn returns an error, nothing is written.
func (s *Service) UpdateTask(ctx context.Context, id string, fn func(t *model.Task) error) (model.Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.updateTaskLocked(ctx, id, fn)
}

func (s *Service) updateTaskLocked(ctx context.Context, id string, fn func(t *model.Task) error) (model.Task, error) {
	t, err := s.GetTask(ctx, id)
	if err != nil {
		return model.Task{}, err
	}
	if err := fn(&t); err != nil {
		return model.Task{}, err
	}
	if err := s.saveTaskLocked(ctx, &t, "task.updated"); err != nil {
		return model.Task{}, err
	}
	return t, nil
}

// CheckBodyEdit enforces the editable_states rule for replacing a task body.
// Pure checkbox toggles are always allowed. hint tells the caller how to append notes instead.
func (s *Service) CheckBodyEdit(status, oldBody, newBody, hint string) error {
	if oldBody == newBody || s.cfg.IsTaskEditable(status) || parser.IsOnlyCheckboxToggle(oldBody, newBody) {
		return nil
	}
	editable := s.cfg.Board.EditableStates
	if len(editable) == 0 {
		editable = []string{"backlog"}
	}
	return conflictf("cannot edit task body while task is in %q status; task body is only editable in [%s]; move task to an editable status to revise specification, or %s",
		status, strings.Join(editable, ", "), hint)
}

// NormalizeTargetAt validates and normalizes a target date.
func NormalizeTargetAt(raw string) (string, error) {
	v, err := parser.NormalizeTimestamp(raw)
	if err != nil {
		return "", invalidf("invalid target_at format: %v", err)
	}
	return v, nil
}

// UpdateTaskStatus moves a task to another board column.
func (s *Service) UpdateTaskStatus(ctx context.Context, id, status string) (model.Task, error) {
	status = strings.TrimSpace(status)
	if status == "" {
		return model.Task{}, invalidf("status is required")
	}
	if err := s.CheckStatus(status); err != nil {
		return model.Task{}, err
	}
	return s.UpdateTask(ctx, id, func(t *model.Task) error {
		t.Status = status
		return nil
	})
}

// UpdateTaskItem toggles the checklist item at 1-based index.
func (s *Service) UpdateTaskItem(ctx context.Context, id string, index int, completed bool) (model.Task, error) {
	return s.UpdateTask(ctx, id, func(t *model.Task) error {
		body, err := parser.UpdateCheckboxByIndex(t.Body, index, completed)
		if err != nil {
			return invalidf("failed to update checkbox item: %v", err)
		}
		t.Body = body
		return nil
	})
}

// AddTaskNote appends a timestamped note under the task's "## Notes" section.
func (s *Service) AddTaskNote(ctx context.Context, id, note string) (model.Task, error) {
	note = strings.TrimSpace(note)
	if note == "" {
		return model.Task{}, invalidf("note content is required")
	}
	return s.UpdateTask(ctx, id, func(t *model.Task) error {
		body, err := parser.AppendTaskNote(t.Body, s.now(), note)
		if err != nil {
			return fmt.Errorf("failed to append note: %w", err)
		}
		t.Body = body
		return nil
	})
}

// SetTaskTarget sets or clears (empty raw) the task's target date.
func (s *Service) SetTaskTarget(ctx context.Context, id, raw string) (model.Task, error) {
	targetAt, err := NormalizeTargetAt(raw)
	if err != nil {
		return model.Task{}, err
	}
	return s.UpdateTask(ctx, id, func(t *model.Task) error {
		t.TargetAt = targetAt
		return nil
	})
}

// AddTaskDependency makes id depend on depID after checking both exist and that
// no cycle is introduced. added is false when the dependency was already present.
func (s *Service) AddTaskDependency(ctx context.Context, id, depID string) (task model.Task, added bool, err error) {
	if id == depID {
		return model.Task{}, false, invalidf("task %q cannot depend on itself", id)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	t, err := s.GetTask(ctx, id)
	if err != nil {
		return model.Task{}, false, err
	}
	if _, err := s.store.GetTask(ctx, depID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return model.Task{}, false, notFoundf("dependency task %q not found", depID)
		}
		return model.Task{}, false, fmt.Errorf("failed to get dependency task %q: %w", depID, err)
	}
	if slices.Contains(t.Dependencies, depID) {
		return t, false, nil
	}
	if err := s.checkCycles(ctx, id, append(slices.Clone(t.Dependencies), depID)); err != nil {
		return model.Task{}, false, err
	}

	t.Dependencies = append(t.Dependencies, depID)
	if err := s.saveTaskLocked(ctx, &t, "task.updated"); err != nil {
		return model.Task{}, false, err
	}
	return t, true, nil
}

// checkCycles reports ErrConflict if giving task id the dependency list deps creates a cycle.
func (s *Service) checkCycles(ctx context.Context, id string, deps []string) error {
	tasks, err := s.store.ListTasks(ctx, model.FilterCriteria{})
	if err != nil {
		return fmt.Errorf("failed to list tasks for cycle check: %w", err)
	}
	graph := make(map[string][]string, len(tasks)+1)
	files := make(map[string]string, len(tasks)+1)
	for _, t := range tasks {
		graph[t.ID] = t.Dependencies
		files[t.ID] = t.FilePath
	}
	graph[id] = deps

	diags := validator.DetectCycles(graph, files)
	if len(diags) == 0 {
		return nil
	}
	var msgs []string
	for _, d := range diags {
		if len(d.Context) > 0 {
			msgs = append(msgs, d.Context...)
		} else {
			msgs = append(msgs, d.Message)
		}
	}
	return conflictf("circular dependency detected: %s", strings.Join(msgs, "; "))
}

// RemoveTaskDependency removes depID from the dependencies of id.
func (s *Service) RemoveTaskDependency(ctx context.Context, id, depID string) (model.Task, error) {
	return s.UpdateTask(ctx, id, func(t *model.Task) error {
		idx := slices.Index(t.Dependencies, depID)
		if idx == -1 {
			return notFoundf("dependency %q not found on task %q", depID, id)
		}
		t.Dependencies = slices.Delete(t.Dependencies, idx, idx+1)
		return nil
	})
}

// CompleteTaskInput holds the documentation required to complete a task.
type CompleteTaskInput struct {
	ID                 string
	Summary            string
	WhatDone           string
	WhyDone            string
	IgnoreDependencies bool
}

// CompleteTask marks a task done, recording a completion summary in its body.
// It returns the updated task and the downstream tasks that became unblocked.
func (s *Service) CompleteTask(ctx context.Context, in CompleteTaskInput) (model.Task, []store.TaskStub, error) {
	summary := strings.TrimSpace(in.Summary)
	if summary == "" {
		return model.Task{}, nil, invalidf("summary is required when completing a task")
	}
	whatDone := strings.TrimSpace(in.WhatDone)
	if whatDone == "" {
		return model.Task{}, nil, invalidf("what_done documentation is required when completing a task")
	}
	whyDone := strings.TrimSpace(in.WhyDone)
	if whyDone == "" {
		return model.Task{}, nil, invalidf("why_done rationale is required when completing a task")
	}

	t, err := s.UpdateTask(ctx, in.ID, func(t *model.Task) error {
		if !in.IgnoreDependencies && len(t.Dependencies) > 0 {
			blockers, err := s.store.GetBlockingTasks(ctx, t.ID)
			if err != nil {
				return fmt.Errorf("failed to check task dependencies: %w", err)
			}
			if len(blockers) > 0 {
				ids := make([]string, len(blockers))
				for i, b := range blockers {
					ids[i] = b.ID
				}
				return conflictf("cannot complete task: blocking dependencies remain unfinished: %v. Use ignore_dependencies=true to override", ids)
			}
		}
		body, err := parser.CompleteTaskBody(t.Body, s.now(), whatDone, whyDone)
		if err != nil {
			return &kindError{kind: ErrConflict, msg: err.Error()}
		}
		t.Status = "done"
		t.Summary = summary
		t.Body = body
		return nil
	})
	if err != nil {
		return model.Task{}, nil, err
	}

	unblocked, _ := s.store.FindUnblockedTasks(ctx, t.ID)
	return t, unblocked, nil
}

// DeleteTask removes a task file and its index entry. Unless force is set,
// it refuses to delete a task that other tasks depend on.
func (s *Service) DeleteTask(ctx context.Context, id string, force bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	t, err := s.GetTask(ctx, id)
	if err != nil {
		return err
	}
	if !force {
		downstream, err := s.store.GetDownstreamTasks(ctx, id)
		if err != nil {
			return fmt.Errorf("failed to check downstream tasks: %w", err)
		}
		if len(downstream) > 0 {
			ids := make([]string, len(downstream))
			for i, d := range downstream {
				ids[i] = d.ID
			}
			return conflictf("cannot delete task %q: %d task(s) depend on it (%s). Set force=true to delete anyway", id, len(downstream), strings.Join(ids, ", "))
		}
	}

	if t.FilePath != "" {
		if err := s.writer.RemoveFile(t.FilePath); err != nil {
			return fmt.Errorf("failed to remove task file: %w", err)
		}
	}
	if err := s.store.DeleteTask(ctx, id); err != nil {
		return fmt.Errorf("failed to delete task from store: %w", err)
	}
	s.notify("task.deleted", map[string]string{"id": id})
	return nil
}
