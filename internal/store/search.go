package store

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/RJuho/jokateko/internal/model"
)

// formatFTS5Query prepares and sanitizes raw query strings for SQLite FTS5 matching.
func formatFTS5Query(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	// If the caller already provided an exact quoted phrase, preserve it.
	if strings.HasPrefix(raw, "\"") && strings.HasSuffix(raw, "\"") {
		return raw
	}

	words := strings.Fields(raw)
	if len(words) == 0 {
		return ""
	}

	var b strings.Builder
	for _, w := range words {
		// Strip special syntax runes that would break FTS5 parser
		clean := strings.Map(func(r rune) rune {
			if strings.ContainsRune(`*^":()`, r) {
				return -1
			}
			return r
		}, w)
		clean = strings.TrimSpace(clean)
		if clean == "" {
			continue
		}
		if b.Len() > 0 {
			b.WriteString(" OR ")
		}
		fmt.Fprintf(&b, "%q*", clean)
	}

	return b.String()
}

// normalizeRank maps SQLite FTS5 BM25 rank to a positive relevance score.
func normalizeRank(rank float64) float64 {
	abs := math.Abs(rank)
	score := abs / (1.0 + abs)
	return math.Round(score*100) / 100
}

// SearchAll executes a universal full-text search across all entity types.
func (s *Store) SearchAll(ctx context.Context, query, tag string, limit int) ([]model.SearchResult, error) {
	s.RLock()
	defer s.RUnlock()

	matchExpr := formatFTS5Query(query)
	if matchExpr == "" {
		return []model.SearchResult{}, nil
	}

	if limit <= 0 {
		limit = 20
	}

	var (
		sqlQuery string
		args     []any
	)

	if tag != "" {
		sqlQuery = `
			SELECT
				f.entity_type,
				f.entity_id,
				f.title,
				snippet(fts_entities, -1, '<mark>', '</mark>', '...', 16) AS snippet,
				f.rank
			FROM fts_entities f
			JOIN entity_tags et ON et.entity_type = f.entity_type AND et.entity_id = f.entity_id
			WHERE f.fts_entities MATCH ? AND et.tag = ?
			ORDER BY f.rank ASC
			LIMIT ?;`
		args = []any{matchExpr, tag, limit}
	} else {
		sqlQuery = `
			SELECT
				f.entity_type,
				f.entity_id,
				f.title,
				snippet(fts_entities, -1, '<mark>', '</mark>', '...', 16) AS snippet,
				f.rank
			FROM fts_entities f
			WHERE f.fts_entities MATCH ?
			ORDER BY f.rank ASC
			LIMIT ?;`
		args = []any{matchExpr, limit}
	}

	rows, err := s.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("search all failed: %w", err)
	}

	type rawItem struct {
		entityType string
		entityID   string
		title      string
		snippet    string
		rank       float64
	}

	var items []rawItem
	for rows.Next() {
		var item rawItem
		if err := rows.Scan(&item.entityType, &item.entityID, &item.title, &item.snippet, &item.rank); err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("failed to scan search result: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, fmt.Errorf("error during row iteration: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("error closing rows: %w", err)
	}

	results := make([]model.SearchResult, 0, len(items))
	for _, item := range items {
		tags, err := s.queries.GetEntityTags(ctx, GetEntityTagsParams{
			EntityType: item.entityType,
			EntityID:   item.entityID,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get tags for %s %q: %w", item.entityType, item.entityID, err)
		}

		results = append(results, model.SearchResult{
			ID:      item.entityID,
			Type:    item.entityType,
			Title:   item.title,
			Tags:    tags,
			Snippet: item.snippet,
			Score:   normalizeRank(item.rank),
		})
	}

	return results, nil
}

// SearchTasks searches task titles, summaries, and Markdown body.
func (s *Store) SearchTasks(ctx context.Context, query, tag string, limit int) ([]model.SearchResult, error) {
	s.RLock()
	defer s.RUnlock()

	matchExpr := formatFTS5Query(query)
	if matchExpr == "" {
		return []model.SearchResult{}, nil
	}

	if limit <= 0 {
		limit = 20
	}

	var (
		sqlQuery string
		args     []any
	)

	if tag != "" {
		sqlQuery = `
			SELECT
				f.entity_id,
				f.title,
				t.status,
				t.priority,
				snippet(fts_entities, -1, '<mark>', '</mark>', '...', 16) AS snippet,
				f.rank
			FROM fts_entities f
			JOIN tasks t ON f.entity_id = t.id
			JOIN entity_tags et ON et.entity_type = 'task' AND et.entity_id = f.entity_id
			WHERE f.entity_type = 'task' AND f.fts_entities MATCH ? AND et.tag = ?
			ORDER BY f.rank ASC
			LIMIT ?;`
		args = []any{matchExpr, tag, limit}
	} else {
		sqlQuery = `
			SELECT
				f.entity_id,
				f.title,
				t.status,
				t.priority,
				snippet(fts_entities, -1, '<mark>', '</mark>', '...', 16) AS snippet,
				f.rank
			FROM fts_entities f
			JOIN tasks t ON f.entity_id = t.id
			WHERE f.entity_type = 'task' AND f.fts_entities MATCH ?
			ORDER BY f.rank ASC
			LIMIT ?;`
		args = []any{matchExpr, limit}
	}

	rows, err := s.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("search tasks failed: %w", err)
	}

	type rawTask struct {
		entityID string
		title    string
		status   string
		priority string
		snippet  string
		rank     float64
	}

	var items []rawTask
	for rows.Next() {
		var item rawTask
		if err := rows.Scan(&item.entityID, &item.title, &item.status, &item.priority, &item.snippet, &item.rank); err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("failed to scan task search result: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, fmt.Errorf("error during row iteration: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("error closing rows: %w", err)
	}

	results := make([]model.SearchResult, 0, len(items))
	for _, item := range items {
		tags, err := s.queries.GetEntityTags(ctx, GetEntityTagsParams{
			EntityType: "task",
			EntityID:   item.entityID,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get tags for task %q: %w", item.entityID, err)
		}

		results = append(results, model.SearchResult{
			ID:       item.entityID,
			Type:     "task",
			Title:    item.title,
			Status:   item.status,
			Priority: model.Priority(item.priority),
			Tags:     tags,
			Snippet:  item.snippet,
			Score:    normalizeRank(item.rank),
		})
	}

	return results, nil
}

// SearchMilestones searches milestone titles, summaries, and Markdown body.
func (s *Store) SearchMilestones(ctx context.Context, query, tag string, limit int) ([]model.SearchResult, error) {
	s.RLock()
	defer s.RUnlock()

	matchExpr := formatFTS5Query(query)
	if matchExpr == "" {
		return []model.SearchResult{}, nil
	}

	if limit <= 0 {
		limit = 20
	}

	var (
		sqlQuery string
		args     []any
	)

	if tag != "" {
		sqlQuery = `
			SELECT
				f.entity_id,
				f.title,
				m.status,
				m.target_date,
				snippet(fts_entities, -1, '<mark>', '</mark>', '...', 16) AS snippet,
				f.rank
			FROM fts_entities f
			JOIN milestones m ON f.entity_id = m.id
			JOIN entity_tags et ON et.entity_type = 'milestone' AND et.entity_id = f.entity_id
			WHERE f.entity_type = 'milestone' AND f.fts_entities MATCH ? AND et.tag = ?
			ORDER BY f.rank ASC
			LIMIT ?;`
		args = []any{matchExpr, tag, limit}
	} else {
		sqlQuery = `
			SELECT
				f.entity_id,
				f.title,
				m.status,
				m.target_date,
				snippet(fts_entities, -1, '<mark>', '</mark>', '...', 16) AS snippet,
				f.rank
			FROM fts_entities f
			JOIN milestones m ON f.entity_id = m.id
			WHERE f.entity_type = 'milestone' AND f.fts_entities MATCH ?
			ORDER BY f.rank ASC
			LIMIT ?;`
		args = []any{matchExpr, limit}
	}

	rows, err := s.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("search milestones failed: %w", err)
	}

	type rawMilestone struct {
		entityID   string
		title      string
		status     string
		targetDate string
		snippet    string
		rank       float64
	}

	var items []rawMilestone
	for rows.Next() {
		var item rawMilestone
		if err := rows.Scan(&item.entityID, &item.title, &item.status, &item.targetDate, &item.snippet, &item.rank); err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("failed to scan milestone search result: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, fmt.Errorf("error during row iteration: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("error closing rows: %w", err)
	}

	results := make([]model.SearchResult, 0, len(items))
	for _, item := range items {
		tags, err := s.queries.GetEntityTags(ctx, GetEntityTagsParams{
			EntityType: "milestone",
			EntityID:   item.entityID,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get tags for milestone %q: %w", item.entityID, err)
		}

		results = append(results, model.SearchResult{
			ID:         item.entityID,
			Type:       "milestone",
			Title:      item.title,
			Status:     item.status,
			TargetDate: item.targetDate,
			Tags:       tags,
			Snippet:    item.snippet,
			Score:      normalizeRank(item.rank),
		})
	}

	return results, nil
}

// SearchStrategies searches architectural guidelines and rules.
func (s *Store) SearchStrategies(ctx context.Context, query, tag string, limit int) ([]model.SearchResult, error) {
	s.RLock()
	defer s.RUnlock()

	matchExpr := formatFTS5Query(query)
	if matchExpr == "" {
		return []model.SearchResult{}, nil
	}

	if limit <= 0 {
		limit = 20
	}

	var (
		sqlQuery string
		args     []any
	)

	if tag != "" {
		sqlQuery = `
			SELECT
				f.entity_id,
				f.title,
				strat.tier,
				snippet(fts_entities, -1, '<mark>', '</mark>', '...', 16) AS snippet,
				f.rank
			FROM fts_entities f
			JOIN strategies strat ON f.entity_id = strat.id
			JOIN entity_tags et ON et.entity_type = 'strategy' AND et.entity_id = f.entity_id
			WHERE f.entity_type = 'strategy' AND f.fts_entities MATCH ? AND et.tag = ?
			ORDER BY f.rank ASC
			LIMIT ?;`
		args = []any{matchExpr, tag, limit}
	} else {
		sqlQuery = `
			SELECT
				f.entity_id,
				f.title,
				strat.tier,
				snippet(fts_entities, -1, '<mark>', '</mark>', '...', 16) AS snippet,
				f.rank
			FROM fts_entities f
			JOIN strategies strat ON f.entity_id = strat.id
			WHERE f.entity_type = 'strategy' AND f.fts_entities MATCH ?
			ORDER BY f.rank ASC
			LIMIT ?;`
		args = []any{matchExpr, limit}
	}

	rows, err := s.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("search strategies failed: %w", err)
	}

	type rawStrat struct {
		entityID string
		title    string
		tier     int64
		snippet  string
		rank     float64
	}

	var items []rawStrat
	for rows.Next() {
		var item rawStrat
		if err := rows.Scan(&item.entityID, &item.title, &item.tier, &item.snippet, &item.rank); err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("failed to scan strategy search result: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, fmt.Errorf("error during row iteration: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("error closing rows: %w", err)
	}

	results := make([]model.SearchResult, 0, len(items))
	for _, item := range items {
		tags, err := s.queries.GetEntityTags(ctx, GetEntityTagsParams{
			EntityType: "strategy",
			EntityID:   item.entityID,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get tags for strategy %q: %w", item.entityID, err)
		}

		results = append(results, model.SearchResult{
			ID:      item.entityID,
			Type:    "strategy",
			Title:   item.title,
			Tier:    model.Tier(item.tier),
			Tags:    tags,
			Snippet: item.snippet,
			Score:   normalizeRank(item.rank),
		})
	}

	return results, nil
}

// SearchGlossary searches standardized terminology and glossary definitions.
func (s *Store) SearchGlossary(ctx context.Context, query, tag string, limit int) ([]model.SearchResult, error) {
	s.RLock()
	defer s.RUnlock()

	matchExpr := formatFTS5Query(query)
	if matchExpr == "" {
		return []model.SearchResult{}, nil
	}

	if limit <= 0 {
		limit = 20
	}

	var (
		sqlQuery string
		args     []any
	)

	if tag != "" {
		sqlQuery = `
			SELECT
				f.entity_id,
				f.title,
				snippet(fts_entities, -1, '<mark>', '</mark>', '...', 16) AS snippet,
				f.rank
			FROM fts_entities f
			JOIN glossary g ON f.entity_id = g.id
			JOIN entity_tags et ON et.entity_type = 'glossary' AND et.entity_id = f.entity_id
			WHERE f.entity_type = 'glossary' AND f.fts_entities MATCH ? AND et.tag = ?
			ORDER BY f.rank ASC
			LIMIT ?;`
		args = []any{matchExpr, tag, limit}
	} else {
		sqlQuery = `
			SELECT
				f.entity_id,
				f.title,
				snippet(fts_entities, -1, '<mark>', '</mark>', '...', 16) AS snippet,
				f.rank
			FROM fts_entities f
			JOIN glossary g ON f.entity_id = g.id
			WHERE f.entity_type = 'glossary' AND f.fts_entities MATCH ?
			ORDER BY f.rank ASC
			LIMIT ?;`
		args = []any{matchExpr, limit}
	}

	rows, err := s.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("search glossary failed: %w", err)
	}

	type rawGlossary struct {
		entityID string
		title    string
		snippet  string
		rank     float64
	}

	var items []rawGlossary
	for rows.Next() {
		var item rawGlossary
		if err := rows.Scan(&item.entityID, &item.title, &item.snippet, &item.rank); err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("failed to scan glossary search result: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, fmt.Errorf("error during row iteration: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("error closing rows: %w", err)
	}

	results := make([]model.SearchResult, 0, len(items))
	for _, item := range items {
		tags, err := s.queries.GetEntityTags(ctx, GetEntityTagsParams{
			EntityType: "glossary",
			EntityID:   item.entityID,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get tags for glossary %q: %w", item.entityID, err)
		}

		results = append(results, model.SearchResult{
			ID:      item.entityID,
			Type:    "glossary",
			Title:   item.title,
			Tags:    tags,
			Snippet: item.snippet,
			Score:   normalizeRank(item.rank),
		})
	}

	return results, nil
}
