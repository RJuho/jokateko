package store

import (
	"context"
	"fmt"

	"github.com/RJuho/jokateko/internal/model"
)

// GetBoardState aggregates tasks into Kanban columns according to configured column definitions and filters.
func (s *Store) GetBoardState(
	ctx context.Context,
	projectName string,
	columns []model.Column,
	filter ...model.FilterCriteria,
) (model.BoardState, error) {
	tasks, err := s.ListTasks(ctx, filter...)
	if err != nil {
		return model.BoardState{}, fmt.Errorf("failed to list tasks for board: %w", err)
	}

	tasksByColumn := make(map[string][]model.Task, len(columns))
	for _, t := range tasks {
		tasksByColumn[t.Status] = append(tasksByColumn[t.Status], t)
	}

	colStates := make([]model.ColumnState, len(columns))
	for i, col := range columns {
		colTasks := tasksByColumn[col.ID]
		if colTasks == nil {
			colTasks = []model.Task{}
		} else {
			model.SortTasksForColumn(colTasks, col)
		}

		colStates[i] = model.ColumnState{
			ID:            col.ID,
			Name:          col.Name,
			Color:         col.Color,
			HandledBy:     col.HandledBy,
			Instructions:  col.Instructions,
			SortBy:        col.SortBy,
			SortDirection: col.SortDirection,
			Tasks:         colTasks,
			Count:         len(colTasks),
		}
	}

	return model.BoardState{
		ProjectName: projectName,
		Columns:     colStates,
	}, nil
}
