package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/RJuho/jokateko/internal/model"
	"github.com/RJuho/jokateko/internal/parser"
)

// UpsertMilestone inserts or updates a milestone, its tags, and its FTS index.
func (s *Store) UpsertMilestone(ctx context.Context, m model.Milestone) error {
	status := m.Status
	if status == "" {
		status = model.MilestoneStatusOpen
	}

	var mtime int64
	if !m.ModTime.IsZero() {
		mtime = m.ModTime.Unix()
	} else {
		mtime = time.Now().Unix()
	}

	return s.WithTx(ctx, func(q *Queries) error {
		// 1. Upsert milestone
		err := q.UpsertMilestone(ctx, UpsertMilestoneParams{
			ID:         m.ID,
			Title:      m.Title,
			Status:     string(status),
			TargetDate: m.TargetDate,
			Summary:    m.Summary,
			Body:       m.Body,
			Filepath:   m.FilePath,
			Mtime:      mtime,
		})
		if err != nil {
			return fmt.Errorf("failed to upsert milestone %q: %w", m.ID, err)
		}

		// 2. Sync tags
		if err := q.ClearEntityTags(ctx, ClearEntityTagsParams{
			EntityType: "milestone",
			EntityID:   m.ID,
		}); err != nil {
			return fmt.Errorf("failed to clear tags for milestone %q: %w", m.ID, err)
		}
		for _, tag := range m.Tags {
			if tag == "" {
				continue
			}
			if err := q.AddEntityTag(ctx, AddEntityTagParams{
				EntityType: "milestone",
				EntityID:   m.ID,
				Tag:        tag,
			}); err != nil {
				return fmt.Errorf("failed to add tag %q to milestone %q: %w", tag, m.ID, err)
			}
		}

		// 3. Sync FTS index
		if err := q.DeleteFTSEntity(ctx, DeleteFTSEntityParams{
			EntityType: "milestone",
			EntityID:   m.ID,
		}); err != nil {
			return fmt.Errorf("failed to delete FTS entry for milestone %q: %w", m.ID, err)
		}
		if err := q.InsertFTSEntity(ctx, InsertFTSEntityParams{
			EntityType: "milestone",
			EntityID:   m.ID,
			Title:      m.Title,
			Summary:    m.Summary,
			Body:       m.Body,
		}); err != nil {
			return fmt.Errorf("failed to insert FTS entry for milestone %q: %w", m.ID, err)
		}

		return nil
	})
}

// GetMilestone retrieves a milestone by its ID along with its computed task progress metrics.
func (s *Store) GetMilestone(ctx context.Context, id string) (model.Milestone, error) {
	s.RLock()
	defer s.RUnlock()

	row, err := s.queries.GetMilestone(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.Milestone{}, fmt.Errorf("%w: milestone %q", ErrNotFound, id)
		}
		return model.Milestone{}, fmt.Errorf("failed to get milestone %q: %w", id, err)
	}

	tags, err := s.queries.GetEntityTags(ctx, GetEntityTagsParams{
		EntityType: "milestone",
		EntityID:   id,
	})
	if err != nil {
		return model.Milestone{}, fmt.Errorf("failed to get tags for milestone %q: %w", id, err)
	}

	metrics, err := s.queries.GetMilestoneTaskMetrics(ctx, id)
	if err != nil {
		return model.Milestone{}, fmt.Errorf("failed to get metrics for milestone %q: %w", id, err)
	}

	var modTime time.Time
	if row.Mtime > 0 {
		modTime = time.Unix(row.Mtime, 0)
	}

	total := int(metrics.TotalTasks)
	completed := int(metrics.CompletedTasks)
	var progressPct float64
	var isArchived bool

	if total > 0 {
		pct := (float64(completed) / float64(total)) * 100.0
		progressPct = math.Round(pct*10) / 10
		if completed == total {
			isArchived = true
		}
	}

	var bodyHTML string
	if row.Body != "" {
		bodyHTML, _ = parser.RenderHTML([]byte(row.Body))
	}

	return model.Milestone{
		ID:                 row.ID,
		Title:              row.Title,
		Status:             model.MilestoneStatus(row.Status),
		IsArchived:         isArchived,
		TargetDate:         row.TargetDate,
		Tags:               tags,
		Summary:            row.Summary,
		Body:               row.Body,
		BodyHTML:           bodyHTML,
		TotalTasks:         total,
		CompletedTasks:     completed,
		ProgressPercentage: progressPct,
		FilePath:           row.Filepath,
		ModTime:            modTime,
	}, nil
}

// DeleteMilestone removes a milestone along with its tags and FTS index.
func (s *Store) DeleteMilestone(ctx context.Context, id string) error {
	return s.WithTx(ctx, func(q *Queries) error {
		if err := q.DeleteMilestone(ctx, id); err != nil {
			return fmt.Errorf("failed to delete milestone %q: %w", id, err)
		}
		if err := q.ClearEntityTags(ctx, ClearEntityTagsParams{
			EntityType: "milestone",
			EntityID:   id,
		}); err != nil {
			return fmt.Errorf("failed to clear tags for milestone %q: %w", id, err)
		}
		if err := q.DeleteFTSEntity(ctx, DeleteFTSEntityParams{
			EntityType: "milestone",
			EntityID:   id,
		}); err != nil {
			return fmt.Errorf("failed to delete FTS entry for milestone %q: %w", id, err)
		}
		return nil
	})
}

// ListMilestones returns all milestones along with their computed task progress metrics.
func (s *Store) ListMilestones(ctx context.Context) ([]model.Milestone, error) {
	s.RLock()
	defer s.RUnlock()

	rows, err := s.queries.ListMilestones(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list milestones: %w", err)
	}

	milestones := make([]model.Milestone, 0, len(rows))
	for _, row := range rows {
		tags, err := s.queries.GetEntityTags(ctx, GetEntityTagsParams{
			EntityType: "milestone",
			EntityID:   row.ID,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get tags for milestone %q: %w", row.ID, err)
		}

		metrics, err := s.queries.GetMilestoneTaskMetrics(ctx, row.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get metrics for milestone %q: %w", row.ID, err)
		}

		var modTime time.Time
		if row.Mtime > 0 {
			modTime = time.Unix(row.Mtime, 0)
		}

		total := int(metrics.TotalTasks)
		completed := int(metrics.CompletedTasks)
		var progressPct float64
		var isArchived bool

		if total > 0 {
			pct := (float64(completed) / float64(total)) * 100.0
			progressPct = math.Round(pct*10) / 10
			if completed == total {
				isArchived = true
			}
		}

		var bodyHTML string
		if row.Body != "" {
			bodyHTML, _ = parser.RenderHTML([]byte(row.Body))
		}

		milestones = append(milestones, model.Milestone{
			ID:                 row.ID,
			Title:              row.Title,
			Status:             model.MilestoneStatus(row.Status),
			IsArchived:         isArchived,
			TargetDate:         row.TargetDate,
			Tags:               tags,
			Summary:            row.Summary,
			Body:               row.Body,
			BodyHTML:           bodyHTML,
			TotalTasks:         total,
			CompletedTasks:     completed,
			ProgressPercentage: progressPct,
			FilePath:           row.Filepath,
			ModTime:            modTime,
		})
	}

	return milestones, nil
}
