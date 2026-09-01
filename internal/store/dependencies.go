package store

import (
	"context"
	"fmt"
)

// TaskStub represents lightweight summary information for a task in the dependency graph.
type TaskStub struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Status string `json:"status"`
}

// AreDependenciesDone checks whether all tasks that taskID depends on have status "done".
// If taskID has zero dependencies, it returns true.
func (s *Store) AreDependenciesDone(ctx context.Context, taskID string) (bool, error) {
	s.RLock()
	defer s.RUnlock()

	count, err := s.queries.CountUnfinishedDependencies(ctx, taskID)
	if err != nil {
		return false, fmt.Errorf("failed to check dependencies for task %q: %w", taskID, err)
	}

	return count == 0, nil
}

// GetBlockingTasks returns all upstream dependency tasks that are not yet marked as "done".
func (s *Store) GetBlockingTasks(ctx context.Context, taskID string) ([]TaskStub, error) {
	s.RLock()
	defer s.RUnlock()

	rows, err := s.queries.GetUnfinishedDependencies(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get unfinished dependencies for task %q: %w", taskID, err)
	}

	tasks := make([]TaskStub, 0, len(rows))
	for _, r := range rows {
		tasks = append(tasks, TaskStub{
			ID:     r.ID,
			Title:  r.Title,
			Status: r.Status,
		})
	}

	return tasks, nil
}

// GetDownstreamTasks returns all tasks that directly declare a dependency on the given task ID.
func (s *Store) GetDownstreamTasks(ctx context.Context, taskID string) ([]TaskStub, error) {
	s.RLock()
	defer s.RUnlock()

	rows, err := s.queries.GetDownstreamTasks(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get downstream tasks for %q: %w", taskID, err)
	}

	tasks := make([]TaskStub, 0, len(rows))
	for _, r := range rows {
		tasks = append(tasks, TaskStub{
			ID:     r.ID,
			Title:  r.Title,
			Status: r.Status,
		})
	}

	return tasks, nil
}

// FindUnblockedTasks finds all downstream tasks that were depending on completedTaskID
// and now have ALL their remaining dependencies satisfied (count == 0).
func (s *Store) FindUnblockedTasks(ctx context.Context, completedTaskID string) ([]TaskStub, error) {
	s.RLock()
	defer s.RUnlock()

	downstream, err := s.queries.GetDownstreamTasks(ctx, completedTaskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get downstream tasks for %q: %w", completedTaskID, err)
	}

	unblocked := make([]TaskStub, 0, len(downstream))
	for _, down := range downstream {
		unfinishedCount, err := s.queries.CountUnfinishedDependencies(ctx, down.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to check unfinished dependencies for %q: %w", down.ID, err)
		}

		if unfinishedCount == 0 {
			unblocked = append(unblocked, TaskStub{
				ID:     down.ID,
				Title:  down.Title,
				Status: down.Status,
			})
		}
	}

	return unblocked, nil
}
