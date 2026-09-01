package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/RJuho/jokateko/internal/model"
)

// UpsertGlossaryTerm inserts or updates a glossary definition, its tags, and FTS index.
func (s *Store) UpsertGlossaryTerm(ctx context.Context, term model.GlossaryTerm) error {
	var mtime int64
	if !term.ModTime.IsZero() {
		mtime = term.ModTime.Unix()
	} else {
		mtime = time.Now().Unix()
	}

	return s.WithTx(ctx, func(q *Queries) error {
		// 1. Upsert glossary term
		err := q.UpsertGlossaryTerm(ctx, UpsertGlossaryTermParams{
			ID:       term.ID,
			Title:    term.Title,
			Summary:  term.Summary,
			Body:     term.Body,
			Filepath: term.FilePath,
			Mtime:    mtime,
		})
		if err != nil {
			return fmt.Errorf("failed to upsert glossary term %q: %w", term.ID, err)
		}

		// 2. Sync tags
		if err := q.ClearEntityTags(ctx, ClearEntityTagsParams{
			EntityType: "glossary",
			EntityID:   term.ID,
		}); err != nil {
			return fmt.Errorf("failed to clear tags for glossary term %q: %w", term.ID, err)
		}
		for _, tag := range term.Tags {
			if tag == "" {
				continue
			}
			if err := q.AddEntityTag(ctx, AddEntityTagParams{
				EntityType: "glossary",
				EntityID:   term.ID,
				Tag:        tag,
			}); err != nil {
				return fmt.Errorf("failed to add tag %q to glossary term %q: %w", tag, term.ID, err)
			}
		}

		// 3. Sync FTS index
		if err := q.DeleteFTSEntity(ctx, DeleteFTSEntityParams{
			EntityType: "glossary",
			EntityID:   term.ID,
		}); err != nil {
			return fmt.Errorf("failed to delete FTS entry for glossary term %q: %w", term.ID, err)
		}
		if err := q.InsertFTSEntity(ctx, InsertFTSEntityParams{
			EntityType: "glossary",
			EntityID:   term.ID,
			Title:      term.Title,
			Summary:    term.Summary,
			Body:       term.Body,
		}); err != nil {
			return fmt.Errorf("failed to insert FTS entry for glossary term %q: %w", term.ID, err)
		}

		return nil
	})
}

// GetGlossaryTerm retrieves a glossary term by ID with its tags.
func (s *Store) GetGlossaryTerm(ctx context.Context, id string) (model.GlossaryTerm, error) {
	s.RLock()
	defer s.RUnlock()

	row, err := s.queries.GetGlossaryTerm(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.GlossaryTerm{}, fmt.Errorf("%w: glossary term %q", ErrNotFound, id)
		}
		return model.GlossaryTerm{}, fmt.Errorf("failed to get glossary term %q: %w", id, err)
	}

	tags, err := s.queries.GetEntityTags(ctx, GetEntityTagsParams{
		EntityType: "glossary",
		EntityID:   id,
	})
	if err != nil {
		return model.GlossaryTerm{}, fmt.Errorf("failed to get tags for glossary term %q: %w", id, err)
	}

	var modTime time.Time
	if row.Mtime > 0 {
		modTime = time.Unix(row.Mtime, 0)
	}

	return model.GlossaryTerm{
		ID:       row.ID,
		Title:    row.Title,
		Tags:     tags,
		Summary:  row.Summary,
		Body:     row.Body,
		FilePath: row.Filepath,
		ModTime:  modTime,
	}, nil
}

// DeleteGlossaryTerm removes a glossary term, its tags, and its FTS index.
func (s *Store) DeleteGlossaryTerm(ctx context.Context, id string) error {
	return s.WithTx(ctx, func(q *Queries) error {
		if err := q.DeleteGlossaryTerm(ctx, id); err != nil {
			return fmt.Errorf("failed to delete glossary term %q: %w", id, err)
		}
		if err := q.ClearEntityTags(ctx, ClearEntityTagsParams{
			EntityType: "glossary",
			EntityID:   id,
		}); err != nil {
			return fmt.Errorf("failed to clear tags for glossary term %q: %w", id, err)
		}
		if err := q.DeleteFTSEntity(ctx, DeleteFTSEntityParams{
			EntityType: "glossary",
			EntityID:   id,
		}); err != nil {
			return fmt.Errorf("failed to delete FTS entry for glossary term %q: %w", id, err)
		}
		return nil
	})
}

// ListGlossaryTerms returns all glossary terms with tags.
func (s *Store) ListGlossaryTerms(ctx context.Context) ([]model.GlossaryTerm, error) {
	s.RLock()
	defer s.RUnlock()

	rows, err := s.queries.ListGlossaryTerms(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list glossary terms: %w", err)
	}

	terms := make([]model.GlossaryTerm, 0, len(rows))
	for _, row := range rows {
		tags, err := s.queries.GetEntityTags(ctx, GetEntityTagsParams{
			EntityType: "glossary",
			EntityID:   row.ID,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get tags for glossary term %q: %w", row.ID, err)
		}

		var modTime time.Time
		if row.Mtime > 0 {
			modTime = time.Unix(row.Mtime, 0)
		}

		terms = append(terms, model.GlossaryTerm{
			ID:       row.ID,
			Title:    row.Title,
			Tags:     tags,
			Summary:  row.Summary,
			Body:     row.Body,
			FilePath: row.Filepath,
			ModTime:  modTime,
		})
	}

	return terms, nil
}
