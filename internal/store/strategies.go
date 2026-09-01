package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/RJuho/jokateko/internal/model"
)

// UpsertStrategy inserts or updates an architectural strategy, its tags, and FTS index.
func (s *Store) UpsertStrategy(ctx context.Context, strat model.Strategy) error {
	tier := strat.Tier
	if !tier.IsValid() {
		tier = model.TierCore
	}

	var mtime int64
	if !strat.ModTime.IsZero() {
		mtime = strat.ModTime.Unix()
	} else {
		mtime = time.Now().Unix()
	}

	return s.WithTx(ctx, func(q *Queries) error {
		// 1. Upsert strategy
		err := q.UpsertStrategy(ctx, UpsertStrategyParams{
			ID:       strat.ID,
			Title:    strat.Title,
			Tier:     int64(tier),
			Summary:  strat.Summary,
			Body:     strat.Body,
			Filepath: strat.FilePath,
			Mtime:    mtime,
		})
		if err != nil {
			return fmt.Errorf("failed to upsert strategy %q: %w", strat.ID, err)
		}

		// 2. Sync tags
		if err := q.ClearEntityTags(ctx, ClearEntityTagsParams{
			EntityType: "strategy",
			EntityID:   strat.ID,
		}); err != nil {
			return fmt.Errorf("failed to clear tags for strategy %q: %w", strat.ID, err)
		}
		for _, tag := range strat.Tags {
			if tag == "" {
				continue
			}
			if err := q.AddEntityTag(ctx, AddEntityTagParams{
				EntityType: "strategy",
				EntityID:   strat.ID,
				Tag:        tag,
			}); err != nil {
				return fmt.Errorf("failed to add tag %q to strategy %q: %w", tag, strat.ID, err)
			}
		}

		// 3. Sync FTS index
		if err := q.DeleteFTSEntity(ctx, DeleteFTSEntityParams{
			EntityType: "strategy",
			EntityID:   strat.ID,
		}); err != nil {
			return fmt.Errorf("failed to delete FTS entry for strategy %q: %w", strat.ID, err)
		}
		if err := q.InsertFTSEntity(ctx, InsertFTSEntityParams{
			EntityType: "strategy",
			EntityID:   strat.ID,
			Title:      strat.Title,
			Summary:    strat.Summary,
			Body:       strat.Body,
		}); err != nil {
			return fmt.Errorf("failed to insert FTS entry for strategy %q: %w", strat.ID, err)
		}

		return nil
	})
}

// GetStrategy retrieves a strategy by ID with its tags.
func (s *Store) GetStrategy(ctx context.Context, id string) (model.Strategy, error) {
	s.RLock()
	defer s.RUnlock()

	row, err := s.queries.GetStrategy(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.Strategy{}, fmt.Errorf("%w: strategy %q", ErrNotFound, id)
		}
		return model.Strategy{}, fmt.Errorf("failed to get strategy %q: %w", id, err)
	}

	tags, err := s.queries.GetEntityTags(ctx, GetEntityTagsParams{
		EntityType: "strategy",
		EntityID:   id,
	})
	if err != nil {
		return model.Strategy{}, fmt.Errorf("failed to get tags for strategy %q: %w", id, err)
	}

	var modTime time.Time
	if row.Mtime > 0 {
		modTime = time.Unix(row.Mtime, 0)
	}

	return model.Strategy{
		ID:       row.ID,
		Title:    row.Title,
		Tier:     model.Tier(row.Tier),
		Tags:     tags,
		Summary:  row.Summary,
		Body:     row.Body,
		FilePath: row.Filepath,
		ModTime:  modTime,
	}, nil
}

// DeleteStrategy removes a strategy, its tags, and its FTS index.
func (s *Store) DeleteStrategy(ctx context.Context, id string) error {
	return s.WithTx(ctx, func(q *Queries) error {
		if err := q.DeleteStrategy(ctx, id); err != nil {
			return fmt.Errorf("failed to delete strategy %q: %w", id, err)
		}
		if err := q.ClearEntityTags(ctx, ClearEntityTagsParams{
			EntityType: "strategy",
			EntityID:   id,
		}); err != nil {
			return fmt.Errorf("failed to clear tags for strategy %q: %w", id, err)
		}
		if err := q.DeleteFTSEntity(ctx, DeleteFTSEntityParams{
			EntityType: "strategy",
			EntityID:   id,
		}); err != nil {
			return fmt.Errorf("failed to delete FTS entry for strategy %q: %w", id, err)
		}
		return nil
	})
}

// ListStrategies returns strategies, optionally filtered by tier.
func (s *Store) ListStrategies(ctx context.Context, tier ...model.Tier) ([]model.Strategy, error) {
	s.RLock()
	defer s.RUnlock()

	var (
		rows []Strategy
		err  error
	)

	if len(tier) > 0 && tier[0].IsValid() {
		rows, err = s.queries.ListStrategiesByTier(ctx, int64(tier[0]))
	} else {
		rows, err = s.queries.ListStrategies(ctx)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to list strategies: %w", err)
	}

	strategies := make([]model.Strategy, 0, len(rows))
	for _, row := range rows {
		tags, err := s.queries.GetEntityTags(ctx, GetEntityTagsParams{
			EntityType: "strategy",
			EntityID:   row.ID,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get tags for strategy %q: %w", row.ID, err)
		}

		var modTime time.Time
		if row.Mtime > 0 {
			modTime = time.Unix(row.Mtime, 0)
		}

		strategies = append(strategies, model.Strategy{
			ID:       row.ID,
			Title:    row.Title,
			Tier:     model.Tier(row.Tier),
			Tags:     tags,
			Summary:  row.Summary,
			Body:     row.Body,
			FilePath: row.Filepath,
			ModTime:  modTime,
		})
	}

	return strategies, nil
}
