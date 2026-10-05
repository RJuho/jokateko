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
)

// writeEntity formats frontmatter and body and writes them atomically to path.
func (s *Service) writeEntity(kind, path string, fm any, body string) error {
	data, err := parser.Format(fm, body)
	if err != nil {
		return fmt.Errorf("failed to format %s markdown: %w", kind, err)
	}
	if err := s.writer.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("failed to save %s file: %w", kind, err)
	}
	return nil
}

// removeEntityFile deletes an entity's Markdown file, if it has one.
func (s *Service) removeEntityFile(kind, path string) error {
	if path == "" {
		return nil
	}
	if err := s.writer.RemoveFile(path); err != nil {
		return fmt.Errorf("failed to remove %s file: %w", kind, err)
	}
	return nil
}

func wrapGet[T any](v T, err error, kind, id string) (T, error) {
	if err != nil {
		var zero T
		if errors.Is(err, store.ErrNotFound) {
			return zero, notFoundf("%s %q not found", kind, id)
		}
		return zero, fmt.Errorf("failed to get %s %q: %w", kind, id, err)
	}
	return v, nil
}

// --- Milestones ---

// GetMilestone returns the milestone with the given ID.
func (s *Service) GetMilestone(ctx context.Context, id string) (model.Milestone, error) {
	ms, err := s.store.GetMilestone(ctx, id)
	return wrapGet(ms, err, "milestone", id)
}

// NewMilestone holds the fields for creating a milestone.
type NewMilestone struct {
	// ID is optional; when empty a "YYMMDD-title-slug" ID is generated.
	ID         string
	Title      string
	Status     model.MilestoneStatus
	TargetDate string
	Tags       []string
	Summary    string
	Body       string
}

func (s *Service) saveMilestoneLocked(ctx context.Context, ms *model.Milestone, event string) error {
	ms.FilePath = entityPath(s.dirs.Milestones, ms.FilePath, ms.ID)
	ms.Tags = nonNil(ms.Tags)
	if err := s.writeEntity("milestone", ms.FilePath, ms.Frontmatter(), ms.Body); err != nil {
		return err
	}
	ms.ModTime = s.now()
	if err := s.store.UpsertMilestone(ctx, *ms); err != nil {
		return fmt.Errorf("failed to index milestone %q: %w", ms.ID, err)
	}
	// Reload to pick up progress fields derived from assigned tasks.
	if fresh, err := s.store.GetMilestone(ctx, ms.ID); err == nil {
		*ms = fresh
	}
	ms.BodyHTML, _ = parser.RenderHTML([]byte(ms.Body))
	s.notify(event, *ms)
	return nil
}

// CreateMilestone validates in, allocates a unique ID, and writes the new milestone.
func (s *Service) CreateMilestone(ctx context.Context, in NewMilestone) (model.Milestone, error) {
	title := strings.TrimSpace(in.Title)
	if title == "" {
		return model.Milestone{}, invalidf("title is required")
	}
	if err := s.CheckTags(in.Tags); err != nil {
		return model.Milestone{}, err
	}
	status := in.Status
	switch status {
	case "":
		status = model.MilestoneStatusOpen
	case model.MilestoneStatusOpen, model.MilestoneStatusClosed:
	default:
		return model.Milestone{}, invalidf("invalid status %q; expected 'open' or 'closed'", status)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	id, err := allocateID(s.dirs.Milestones, strings.TrimSpace(in.ID), s.datedSlug(title, "milestone"), func(id string) bool {
		_, err := s.store.GetMilestone(ctx, id)
		return err == nil
	})
	if err != nil {
		return model.Milestone{}, err
	}

	ms := model.Milestone{
		ID:         id,
		Title:      title,
		Status:     status,
		TargetDate: strings.TrimSpace(in.TargetDate),
		Tags:       in.Tags,
		Summary:    strings.TrimSpace(in.Summary),
		Body:       in.Body,
	}
	if err := s.saveMilestoneLocked(ctx, &ms, "milestone.created"); err != nil {
		return model.Milestone{}, err
	}
	return ms, nil
}

// UpdateMilestone loads a milestone, applies fn, and persists the result.
func (s *Service) UpdateMilestone(ctx context.Context, id string, fn func(ms *model.Milestone) error) (model.Milestone, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ms, err := s.GetMilestone(ctx, id)
	if err != nil {
		return model.Milestone{}, err
	}
	prevTags := slices.Clone(ms.Tags)
	if err := fn(&ms); err != nil {
		return model.Milestone{}, err
	}
	if err := s.checkAddedTags(prevTags, ms.Tags); err != nil {
		return model.Milestone{}, err
	}
	if err := s.saveMilestoneLocked(ctx, &ms, "milestone.updated"); err != nil {
		return model.Milestone{}, err
	}
	return ms, nil
}

// ParseMilestoneStatus validates a user-supplied milestone status.
func ParseMilestoneStatus(raw string) (model.MilestoneStatus, error) {
	switch st := model.MilestoneStatus(strings.ToLower(strings.TrimSpace(raw))); st {
	case model.MilestoneStatusOpen, model.MilestoneStatusClosed:
		return st, nil
	default:
		return "", invalidf("invalid status %q; expected 'open' or 'closed'", raw)
	}
}

// DeleteMilestone removes a milestone. Unless force is set, it refuses to delete
// a milestone that still has tasks assigned; with force, those tasks lose their
// milestone first so no dangling reference remains.
func (s *Service) DeleteMilestone(ctx context.Context, id string, force bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ms, err := s.GetMilestone(ctx, id)
	if err != nil {
		return err
	}
	if !force && ms.TotalTasks > 0 {
		return conflictf("cannot delete milestone %q: %d task(s) are assigned to it. Set force=true to delete anyway", id, ms.TotalTasks)
	}
	assigned, err := s.store.ListTasks(ctx, model.FilterCriteria{Milestone: id})
	if err != nil {
		return fmt.Errorf("failed to list tasks of milestone %q: %w", id, err)
	}
	for _, a := range assigned {
		if _, err := s.updateTaskLocked(ctx, a.ID, false, func(t *model.Task) error {
			t.Milestone = ""
			return nil
		}); err != nil {
			return fmt.Errorf("failed to detach task %q from milestone %q: %w", a.ID, id, err)
		}
	}
	if err := s.removeEntityFile("milestone", ms.FilePath); err != nil {
		return err
	}
	if err := s.store.DeleteMilestone(ctx, id); err != nil {
		return fmt.Errorf("failed to delete milestone from store: %w", err)
	}
	s.notify("milestone.deleted", map[string]string{"id": id})
	return nil
}

// --- Strategies ---

// GetStrategy returns the strategy with the given ID.
func (s *Service) GetStrategy(ctx context.Context, id string) (model.Strategy, error) {
	st, err := s.store.GetStrategy(ctx, id)
	return wrapGet(st, err, "strategy", id)
}

// NewStrategy holds the fields for creating a strategy.
type NewStrategy struct {
	// ID is optional; when empty a slug of the title is used.
	ID    string
	Title string
	// Tier defaults to TierCore when zero.
	Tier    model.Tier
	Tags    []string
	Summary string
	Body    string
}

// CheckTier validates a strategy tier.
func CheckTier(tier model.Tier) error {
	if !tier.IsValid() {
		return invalidf("invalid tier %d: tier must be 1 (Core Invariants), 2 (Domain Patterns), or 3 (Implementation Specs)", tier)
	}
	return nil
}

func (s *Service) saveStrategyLocked(ctx context.Context, st *model.Strategy, event string) error {
	st.FilePath = entityPath(s.dirs.Strategies, st.FilePath, st.ID)
	st.Tags = nonNil(st.Tags)
	if err := s.writeEntity("strategy", st.FilePath, st.Frontmatter(), st.Body); err != nil {
		return err
	}
	st.ModTime = s.now()
	st.BodyHTML, _ = parser.RenderHTML([]byte(st.Body))
	if err := s.store.UpsertStrategy(ctx, *st); err != nil {
		return fmt.Errorf("failed to index strategy %q: %w", st.ID, err)
	}
	s.notify(event, *st)
	return nil
}

// CreateStrategy validates in, allocates a unique ID, and writes the new strategy.
func (s *Service) CreateStrategy(ctx context.Context, in NewStrategy) (model.Strategy, error) {
	title := strings.TrimSpace(in.Title)
	if title == "" {
		return model.Strategy{}, invalidf("title is required")
	}
	if err := s.CheckTags(in.Tags); err != nil {
		return model.Strategy{}, err
	}
	tier := in.Tier
	if tier == 0 {
		tier = model.TierCore
	}
	if err := CheckTier(tier); err != nil {
		return model.Strategy{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	id, err := allocateID(s.dirs.Strategies, strings.TrimSpace(in.ID), Slugify(title, "strategy"), func(id string) bool {
		_, err := s.store.GetStrategy(ctx, id)
		return err == nil
	})
	if err != nil {
		return model.Strategy{}, err
	}

	st := model.Strategy{
		ID:      id,
		Title:   title,
		Tier:    tier,
		Tags:    in.Tags,
		Summary: strings.TrimSpace(in.Summary),
		Body:    in.Body,
	}
	if err := s.saveStrategyLocked(ctx, &st, "strategy.created"); err != nil {
		return model.Strategy{}, err
	}
	return st, nil
}

// UpdateStrategy loads a strategy, applies fn, and persists the result.
func (s *Service) UpdateStrategy(ctx context.Context, id string, fn func(st *model.Strategy) error) (model.Strategy, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	st, err := s.GetStrategy(ctx, id)
	if err != nil {
		return model.Strategy{}, err
	}
	prevTags := slices.Clone(st.Tags)
	if err := fn(&st); err != nil {
		return model.Strategy{}, err
	}
	if err := s.checkAddedTags(prevTags, st.Tags); err != nil {
		return model.Strategy{}, err
	}
	if err := s.saveStrategyLocked(ctx, &st, "strategy.updated"); err != nil {
		return model.Strategy{}, err
	}
	return st, nil
}

// DeleteStrategy removes a strategy file and its index entry.
func (s *Service) DeleteStrategy(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	st, err := s.GetStrategy(ctx, id)
	if err != nil {
		return err
	}
	if err := s.removeEntityFile("strategy", st.FilePath); err != nil {
		return err
	}
	if err := s.store.DeleteStrategy(ctx, id); err != nil {
		return fmt.Errorf("failed to delete strategy from store: %w", err)
	}
	s.notify("strategy.deleted", map[string]string{"id": id})
	return nil
}

// --- Glossary ---

// GetGlossaryTerm returns the glossary term with the given ID.
func (s *Service) GetGlossaryTerm(ctx context.Context, id string) (model.GlossaryTerm, error) {
	term, err := s.store.GetGlossaryTerm(ctx, id)
	return wrapGet(term, err, "glossary term", id)
}

// NewGlossaryTerm holds the fields for creating a glossary term.
type NewGlossaryTerm struct {
	// ID is optional; when empty a slug of the title is used.
	ID      string
	Title   string
	Tags    []string
	Summary string
	Body    string
}

func (s *Service) saveGlossaryTermLocked(ctx context.Context, term *model.GlossaryTerm, event string) error {
	term.FilePath = entityPath(s.dirs.Glossary, term.FilePath, term.ID)
	term.Tags = nonNil(term.Tags)
	if err := s.writeEntity("glossary term", term.FilePath, term.Frontmatter(), term.Body); err != nil {
		return err
	}
	term.ModTime = s.now()
	term.BodyHTML, _ = parser.RenderHTML([]byte(term.Body))
	if err := s.store.UpsertGlossaryTerm(ctx, *term); err != nil {
		return fmt.Errorf("failed to index glossary term %q: %w", term.ID, err)
	}
	s.notify(event, *term)
	return nil
}

// CreateGlossaryTerm validates in, allocates a unique ID, and writes the new term.
func (s *Service) CreateGlossaryTerm(ctx context.Context, in NewGlossaryTerm) (model.GlossaryTerm, error) {
	title := strings.TrimSpace(in.Title)
	if title == "" {
		return model.GlossaryTerm{}, invalidf("title is required")
	}
	if err := s.CheckTags(in.Tags); err != nil {
		return model.GlossaryTerm{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	id, err := allocateID(s.dirs.Glossary, strings.TrimSpace(in.ID), Slugify(title, "term"), func(id string) bool {
		_, err := s.store.GetGlossaryTerm(ctx, id)
		return err == nil
	})
	if err != nil {
		return model.GlossaryTerm{}, err
	}

	term := model.GlossaryTerm{
		ID:      id,
		Title:   title,
		Tags:    in.Tags,
		Summary: strings.TrimSpace(in.Summary),
		Body:    in.Body,
	}
	if err := s.saveGlossaryTermLocked(ctx, &term, "glossary.created"); err != nil {
		return model.GlossaryTerm{}, err
	}
	return term, nil
}

// UpdateGlossaryTerm loads a glossary term, applies fn, and persists the result.
func (s *Service) UpdateGlossaryTerm(ctx context.Context, id string, fn func(term *model.GlossaryTerm) error) (model.GlossaryTerm, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	term, err := s.GetGlossaryTerm(ctx, id)
	if err != nil {
		return model.GlossaryTerm{}, err
	}
	prevTags := slices.Clone(term.Tags)
	if err := fn(&term); err != nil {
		return model.GlossaryTerm{}, err
	}
	if err := s.checkAddedTags(prevTags, term.Tags); err != nil {
		return model.GlossaryTerm{}, err
	}
	if err := s.saveGlossaryTermLocked(ctx, &term, "glossary.updated"); err != nil {
		return model.GlossaryTerm{}, err
	}
	return term, nil
}

// DeleteGlossaryTerm removes a glossary term file and its index entry.
func (s *Service) DeleteGlossaryTerm(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	term, err := s.GetGlossaryTerm(ctx, id)
	if err != nil {
		return err
	}
	if err := s.removeEntityFile("glossary term", term.FilePath); err != nil {
		return err
	}
	if err := s.store.DeleteGlossaryTerm(ctx, id); err != nil {
		return fmt.Errorf("failed to delete glossary term from store: %w", err)
	}
	s.notify("glossary.deleted", map[string]string{"id": id})
	return nil
}
