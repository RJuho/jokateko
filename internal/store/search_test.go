package store_test

import (
	"context"
	"strings"
	"testing"

	"github.com/RJuho/jokateko/internal/model"
	"github.com/RJuho/jokateko/internal/store"
)

func TestFullTextSearch(t *testing.T) {
	st, err := store.OpenMemory()
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer func() {
		_ = st.Close()
	}()

	ctx := context.Background()

	// Seed entities
	task := model.Task{
		ID:       "260901-fts-task",
		Title:    "Implement SQLite In-Memory Database Store",
		Status:   "done",
		Priority: model.PriorityHigh,
		Tags:     []string{"backend", "database", "sqlite"},
		Summary:  "Set up pure Go in-memory SQLite tables using modernc.org/sqlite and sqlc.",
		Body:     "The SQLite in-memory store uses modernc pure Go driver without CGO.",
	}
	if err := st.UpsertTask(ctx, task); err != nil {
		t.Fatalf("UpsertTask failed: %v", err)
	}

	milestone := model.Milestone{
		ID:         "260915-fts-milestone",
		Title:      "MVP Release Delivery",
		Status:     model.MilestoneStatusOpen,
		TargetDate: "2026-09-15",
		Tags:       []string{"release", "mvp"},
		Summary:    "Deliver core local daemon with file watcher, in-memory SQLite, and live Preact Kanban.",
		Body:       "The MVP delivery milestone encapsulates all foundational capabilities.",
	}
	if err := st.UpsertMilestone(ctx, milestone); err != nil {
		t.Fatalf("UpsertMilestone failed: %v", err)
	}

	strategy := model.Strategy{
		ID:      "strat-pure-go",
		Title:   "Zero CGO and Single Executable Architecture",
		Tier:    model.TierCore,
		Tags:    []string{"architecture", "go"},
		Summary: "All Go code must cross-compile cleanly without CGO or external system libraries.",
		Body:    "Strictly enforce pure Go implementation for portability across platforms.",
	}
	if err := st.UpsertStrategy(ctx, strategy); err != nil {
		t.Fatalf("UpsertStrategy failed: %v", err)
	}

	term := model.GlossaryTerm{
		ID:      "term-sqlite-index",
		Title:   "In-Memory Query Index",
		Tags:    []string{"backend", "database"},
		Summary: "The ephemeral SQLite database running inside the Jokateko daemon.",
		Body:    "Provides fast in-memory indexing of markdown tasks and milestones.",
	}
	if err := st.UpsertGlossaryTerm(ctx, term); err != nil {
		t.Fatalf("UpsertGlossaryTerm failed: %v", err)
	}

	// 1. SearchAll (Universal Search)
	t.Run("SearchAll matches across different entity types", func(t *testing.T) {
		results, err := st.SearchAll(ctx, "SQLite", "", 10)
		if err != nil {
			t.Fatalf("SearchAll failed: %v", err)
		}
		if len(results) < 3 {
			t.Errorf("expected at least 3 matches for 'SQLite', got %d", len(results))
		}

		// Verify snippet contains <mark>
		foundSnippetMark := false
		for _, r := range results {
			if strings.Contains(r.Snippet, "<mark>") {
				foundSnippetMark = true
				break
			}
		}
		if !foundSnippetMark {
			t.Error("expected at least one snippet with <mark> highlighting")
		}
	})

	// 2. SearchAll with Tag Filter
	t.Run("SearchAll filters by tag", func(t *testing.T) {
		results, err := st.SearchAll(ctx, "SQLite", "release", 10)
		if err != nil {
			t.Fatalf("SearchAll with tag failed: %v", err)
		}
		if len(results) != 1 || results[0].ID != milestone.ID {
			t.Errorf("expected only milestone to match tag 'release', got %+v", results)
		}
	})

	// 3. SearchTasks
	t.Run("SearchTasks returns task specific fields", func(t *testing.T) {
		results, err := st.SearchTasks(ctx, "modernc", "", 10)
		if err != nil {
			t.Fatalf("SearchTasks failed: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("expected 1 task match, got %d", len(results))
		}
		res := results[0]
		if res.ID != task.ID || res.Status != "done" || res.Priority != model.PriorityHigh {
			t.Errorf("unexpected task search result: %+v", res)
		}
	})

	// 4. SearchMilestones
	t.Run("SearchMilestones returns milestone specific fields", func(t *testing.T) {
		results, err := st.SearchMilestones(ctx, "Kanban", "", 10)
		if err != nil {
			t.Fatalf("SearchMilestones failed: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("expected 1 milestone match, got %d", len(results))
		}
		res := results[0]
		if res.ID != milestone.ID || res.TargetDate != "2026-09-15" {
			t.Errorf("unexpected milestone search result: %+v", res)
		}
	})

	// 5. SearchStrategies
	t.Run("SearchStrategies returns strategy specific fields", func(t *testing.T) {
		results, err := st.SearchStrategies(ctx, "CGO", "", 10)
		if err != nil {
			t.Fatalf("SearchStrategies failed: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("expected 1 strategy match, got %d", len(results))
		}
		res := results[0]
		if res.ID != strategy.ID || res.Tier != model.TierCore {
			t.Errorf("unexpected strategy search result: %+v", res)
		}
	})

	// 6. SearchGlossary
	t.Run("SearchGlossary returns glossary terms", func(t *testing.T) {
		results, err := st.SearchGlossary(ctx, "ephemeral", "", 10)
		if err != nil {
			t.Fatalf("SearchGlossary failed: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("expected 1 glossary match, got %d", len(results))
		}
		if results[0].ID != term.ID {
			t.Errorf("unexpected glossary search result: %+v", results[0])
		}
	})

	// 7. Empty Query
	t.Run("Empty query returns empty slice", func(t *testing.T) {
		results, err := st.SearchAll(ctx, "   ", "", 10)
		if err != nil {
			t.Fatalf("expected no error on empty query: %v", err)
		}
		if len(results) != 0 {
			t.Errorf("expected 0 results, got %d", len(results))
		}
	})
}
