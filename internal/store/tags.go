package store

import (
	"context"
	"fmt"

	"github.com/RJuho/jokateko/internal/model"
)

// GetTagCounts aggregates usage metrics for all tags across tasks, milestones, and strategies.
func (s *Store) GetTagCounts(ctx context.Context) ([]model.TagCount, error) {
	s.RLock()
	defer s.RUnlock()

	rows, err := s.queries.GetTagCounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get tag counts: %w", err)
	}

	tags := make([]model.TagCount, 0, len(rows))
	for _, row := range rows {
		tags = append(tags, model.TagCount{
			Tag:            row.Tag,
			TaskCount:      int(row.TaskCount),
			MilestoneCount: int(row.MilestoneCount),
			StrategyCount:  int(row.StrategyCount),
		})
	}

	return tags, nil
}

// GetEntityTags returns all tags attached to a specific entity.
func (s *Store) GetEntityTags(ctx context.Context, entityType, entityID string) ([]string, error) {
	s.RLock()
	defer s.RUnlock()

	return s.queries.GetEntityTags(ctx, GetEntityTagsParams{
		EntityType: entityType,
		EntityID:   entityID,
	})
}
