package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/RJuho/jokateko/internal/model"
)

// UpsertTask inserts or updates a task and its associated tags, dependencies, and FTS index.
func (s *Store) UpsertTask(ctx context.Context, task model.Task) error {
	p := task.Priority
	if p == "" {
		p = model.PriorityMedium
	}

	var mtime int64
	if !task.ModTime.IsZero() {
		mtime = task.ModTime.Unix()
	} else {
		mtime = time.Now().Unix()
	}

	return s.WithTx(ctx, func(q *Queries) error {
		// 1. Upsert task record
		err := q.UpsertTask(ctx, UpsertTaskParams{
			ID:                task.ID,
			Title:             task.Title,
			Status:            task.Status,
			Priority:          string(p),
			MilestoneID:       task.Milestone,
			Summary:           task.Summary,
			Body:              task.Body,
			TotalCriteria:     int64(task.TotalCriteria),
			CompletedCriteria: int64(task.CompletedCriteria),
			Filepath:          task.FilePath,
			Mtime:             mtime,
		})
		if err != nil {
			return fmt.Errorf("failed to upsert task %q: %w", task.ID, err)
		}

		// 2. Sync tags
		if err := q.ClearEntityTags(ctx, ClearEntityTagsParams{
			EntityType: "task",
			EntityID:   task.ID,
		}); err != nil {
			return fmt.Errorf("failed to clear tags for task %q: %w", task.ID, err)
		}
		for _, tag := range task.Tags {
			if tag == "" {
				continue
			}
			if err := q.AddEntityTag(ctx, AddEntityTagParams{
				EntityType: "task",
				EntityID:   task.ID,
				Tag:        tag,
			}); err != nil {
				return fmt.Errorf("failed to add tag %q to task %q: %w", tag, task.ID, err)
			}
		}

		// 3. Sync dependencies
		if err := q.ClearTaskDependencies(ctx, task.ID); err != nil {
			return fmt.Errorf("failed to clear dependencies for task %q: %w", task.ID, err)
		}
		for _, dep := range task.Dependencies {
			if dep == "" {
				continue
			}
			if err := q.AddTaskDependency(ctx, AddTaskDependencyParams{
				TaskID:          task.ID,
				DependsOnTaskID: dep,
			}); err != nil {
				return fmt.Errorf("failed to add dependency %q to task %q: %w", dep, task.ID, err)
			}
		}

		// 4. Sync FTS index
		if err := q.DeleteFTSEntity(ctx, DeleteFTSEntityParams{
			EntityType: "task",
			EntityID:   task.ID,
		}); err != nil {
			return fmt.Errorf("failed to delete FTS entry for task %q: %w", task.ID, err)
		}
		if err := q.InsertFTSEntity(ctx, InsertFTSEntityParams{
			EntityType: "task",
			EntityID:   task.ID,
			Title:      task.Title,
			Summary:    task.Summary,
			Body:       task.Body,
		}); err != nil {
			return fmt.Errorf("failed to insert FTS entry for task %q: %w", task.ID, err)
		}

		return nil
	})
}

// GetTask retrieves a task by its ID along with its tags and dependencies.
func (s *Store) GetTask(ctx context.Context, id string) (model.Task, error) {
	s.RLock()
	defer s.RUnlock()

	row, err := s.queries.GetTask(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.Task{}, fmt.Errorf("%w: task %q", ErrNotFound, id)
		}
		return model.Task{}, fmt.Errorf("failed to get task %q: %w", id, err)
	}

	tags, err := s.queries.GetEntityTags(ctx, GetEntityTagsParams{
		EntityType: "task",
		EntityID:   id,
	})
	if err != nil {
		return model.Task{}, fmt.Errorf("failed to get tags for task %q: %w", id, err)
	}

	deps, err := s.queries.GetTaskDependencies(ctx, id)
	if err != nil {
		return model.Task{}, fmt.Errorf("failed to get dependencies for task %q: %w", id, err)
	}

	var modTime time.Time
	if row.Mtime > 0 {
		modTime = time.Unix(row.Mtime, 0)
	}

	return model.Task{
		ID:                row.ID,
		Title:             row.Title,
		Status:            row.Status,
		Priority:          model.Priority(row.Priority),
		Milestone:         row.MilestoneID,
		Tags:              tags,
		Summary:           row.Summary,
		Dependencies:      deps,
		Body:              row.Body,
		TotalCriteria:     int(row.TotalCriteria),
		CompletedCriteria: int(row.CompletedCriteria),
		FilePath:          row.Filepath,
		ModTime:           modTime,
	}, nil
}

// DeleteTask removes a task along with its tags, dependencies, and FTS index.
func (s *Store) DeleteTask(ctx context.Context, id string) error {
	return s.WithTx(ctx, func(q *Queries) error {
		if err := q.DeleteTask(ctx, id); err != nil {
			return fmt.Errorf("failed to delete task %q: %w", id, err)
		}
		if err := q.ClearEntityTags(ctx, ClearEntityTagsParams{
			EntityType: "task",
			EntityID:   id,
		}); err != nil {
			return fmt.Errorf("failed to clear tags for task %q: %w", id, err)
		}
		if err := q.ClearTaskDependencies(ctx, id); err != nil {
			return fmt.Errorf("failed to clear dependencies for task %q: %w", id, err)
		}
		if err := q.DeleteFTSEntity(ctx, DeleteFTSEntityParams{
			EntityType: "task",
			EntityID:   id,
		}); err != nil {
			return fmt.Errorf("failed to delete FTS entry for task %q: %w", id, err)
		}
		return nil
	})
}

// ListTasks returns all tasks, optionally filtered by criteria.
func (s *Store) ListTasks(ctx context.Context, filter ...model.FilterCriteria) ([]model.Task, error) {
	s.RLock()
	defer s.RUnlock()

	rows, err := s.queries.ListTasks(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}

	hasFilter := len(filter) > 0
	tasks := make([]model.Task, 0, len(rows))

	for _, row := range rows {
		tags, err := s.queries.GetEntityTags(ctx, GetEntityTagsParams{
			EntityType: "task",
			EntityID:   row.ID,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get tags for task %q: %w", row.ID, err)
		}

		deps, err := s.queries.GetTaskDependencies(ctx, row.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get dependencies for task %q: %w", row.ID, err)
		}

		var modTime time.Time
		if row.Mtime > 0 {
			modTime = time.Unix(row.Mtime, 0)
		}

		task := model.Task{
			ID:                row.ID,
			Title:             row.Title,
			Status:            row.Status,
			Priority:          model.Priority(row.Priority),
			Milestone:         row.MilestoneID,
			Tags:              tags,
			Summary:           row.Summary,
			Dependencies:      deps,
			Body:              row.Body,
			TotalCriteria:     int(row.TotalCriteria),
			CompletedCriteria: int(row.CompletedCriteria),
			FilePath:          row.Filepath,
			ModTime:           modTime,
		}

		if hasFilter && !filter[0].Matches(task) {
			continue
		}

		tasks = append(tasks, task)
	}

	return tasks, nil
}

// UpdateTaskStatus updates a task's status and modification timestamp.
func (s *Store) UpdateTaskStatus(ctx context.Context, id, status string) error {
	return s.WithTx(ctx, func(q *Queries) error {
		return q.UpdateTaskStatus(ctx, UpdateTaskStatusParams{
			ID:     id,
			Status: status,
			Mtime:  time.Now().Unix(),
		})
	})
}

// UpdateTaskCriteria updates a task's criteria counters, body content, and FTS index.
func (s *Store) UpdateTaskCriteria(ctx context.Context, id string, total, completed int, body string) error {
	return s.WithTx(ctx, func(q *Queries) error {
		task, err := q.GetTask(ctx, id)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("%w: task %q", ErrNotFound, id)
			}
			return err
		}

		if err := q.UpdateTaskCriteria(ctx, UpdateTaskCriteriaParams{
			ID:                id,
			TotalCriteria:     int64(total),
			CompletedCriteria: int64(completed),
			Body:              body,
			Mtime:             time.Now().Unix(),
		}); err != nil {
			return err
		}

		// Update FTS
		_ = q.DeleteFTSEntity(ctx, DeleteFTSEntityParams{
			EntityType: "task",
			EntityID:   id,
		})
		return q.InsertFTSEntity(ctx, InsertFTSEntityParams{
			EntityType: "task",
			EntityID:   id,
			Title:      task.Title,
			Summary:    task.Summary,
			Body:       body,
		})
	})
}
