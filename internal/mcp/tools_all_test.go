package mcp_test

import (
	"os"
	"path/filepath"
	"testing"

	internalmcp "github.com/RJuho/jokateko/internal/mcp"
	"github.com/RJuho/jokateko/internal/model"
)

func TestMCP_MilestoneTools(t *testing.T) {
	_, dir, _, session := setupTestMCP(t)

	// 1. create_milestone
	createInput := internalmcp.CreateMilestoneInput{
		Title:      "V1 Release",
		TargetDate: "2026-10-01",
		Tags:       []string{"feature"},
		Summary:    "Initial production release",
		Body:       "## Roadmap\nDeliver core features.",
	}
	ms, err := callToolJSON[internalmcp.MilestoneDetail](t, session, "create_milestone", createInput)
	if err != nil {
		t.Fatalf("create_milestone failed: %v", err)
	}
	if ms.Title != "V1 Release" || ms.TargetDate != "2026-10-01" {
		t.Errorf("unexpected created milestone: %+v", ms)
	}

	// Verify file on disk
	msFile := filepath.Join(dir, ".jokateko", "milestones", ms.ID+".md")
	if _, err := os.Stat(msFile); os.IsNotExist(err) {
		t.Fatalf("milestone file not found at %s", msFile)
	}

	// 2. list_milestones
	list, err := callToolJSON[[]internalmcp.MilestoneSummary](t, session, "list_milestones", internalmcp.ListMilestonesInput{})
	if err != nil {
		t.Fatalf("list_milestones failed: %v", err)
	}
	if len(list) != 1 || list[0].ID != ms.ID {
		t.Errorf("expected 1 milestone in list, got %+v", list)
	}

	// 3. get_milestone
	fetched, err := callToolJSON[internalmcp.MilestoneDetail](t, session, "get_milestone", internalmcp.GetMilestoneInput{ID: ms.ID})
	if err != nil {
		t.Fatalf("get_milestone failed: %v", err)
	}
	if fetched.ID != ms.ID || fetched.Summary != ms.Summary {
		t.Errorf("fetched milestone mismatch: %+v", fetched)
	}

	// 4. update_milestone
	newSummary := "Updated milestone deliverable"
	updated, err := callToolJSON[internalmcp.MilestoneDetail](t, session, "update_milestone", internalmcp.UpdateMilestoneInput{
		ID:      ms.ID,
		Summary: &newSummary,
	})
	if err != nil {
		t.Fatalf("update_milestone failed: %v", err)
	}
	if updated.Summary != newSummary {
		t.Errorf("expected updated summary %q, got %q", newSummary, updated.Summary)
	}
}

func TestMCP_StrategyAndGlossaryTools(t *testing.T) {
	_, _, st, session := setupTestMCP(t)
	ctx := t.Context()

	// Seed strategy
	_ = st.UpsertStrategy(ctx, model.Strategy{
		ID:      "zero-cgo",
		Title:   "Zero CGO Architecture",
		Tier:    model.TierCore,
		Tags:    []string{"backend"},
		Summary: "Strict zero-CGO policy for deterministic cross-compilation",
		Body:    "All packages must compile with CGO_ENABLED=0.",
	})

	// Seed glossary
	_ = st.UpsertGlossaryTerm(ctx, model.GlossaryTerm{
		ID:      "tac",
		Title:   "Tasks-as-Code",
		Tags:    []string{"feature"},
		Summary: "Managing tasks directly as versioned Markdown files",
		Body:    "Full definition of Tasks-as-Code.",
	})

	// 1. list_strategies
	strats, err := callToolJSON[[]internalmcp.StrategySummary](t, session, "list_strategies", internalmcp.ListStrategiesInput{Tier: 1})
	if err != nil {
		t.Fatalf("list_strategies failed: %v", err)
	}
	if len(strats) != 1 || strats[0].ID != "zero-cgo" {
		t.Errorf("unexpected strategies list: %+v", strats)
	}

	// 2. get_strategy
	stratDetail, err := callToolJSON[internalmcp.StrategyDetail](t, session, "get_strategy", internalmcp.GetStrategyInput{ID: "zero-cgo"})
	if err != nil {
		t.Fatalf("get_strategy failed: %v", err)
	}
	if stratDetail.Title != "Zero CGO Architecture" || stratDetail.Tier != 1 {
		t.Errorf("unexpected strategy detail: %+v", stratDetail)
	}

	// 3. create_strategy
	createdStrat, err := callToolJSON[internalmcp.StrategyDetail](t, session, "create_strategy", internalmcp.CreateStrategyInput{
		Title:   "Progressive Disclosure Guidelines",
		Tier:    2,
		Summary: "Architecture guidelines for tiered information exposure",
		Tags:    []string{"backend"},
		Body:    "## Progressive Disclosure\nTiers expose information gradually.",
	})
	if err != nil {
		t.Fatalf("create_strategy failed: %v", err)
	}
	if createdStrat.ID != "progressive-disclosure-guidelines" || createdStrat.Tier != 2 {
		t.Errorf("unexpected created strategy: %+v", createdStrat)
	}

	// 4. update_strategy
	updatedStrat, err := callToolJSON[internalmcp.StrategyDetail](t, session, "update_strategy", internalmcp.UpdateStrategyInput{
		ID:      "progressive-disclosure-guidelines",
		Tier:    3,
		Summary: "Updated summary for progressive disclosure",
	})
	if err != nil {
		t.Fatalf("update_strategy failed: %v", err)
	}
	if updatedStrat.Tier != 3 || updatedStrat.Summary != "Updated summary for progressive disclosure" {
		t.Errorf("unexpected updated strategy: %+v", updatedStrat)
	}

	// 5. lookup_glossary (all)
	allTerms, err := callToolJSON[[]internalmcp.GlossaryEntry](t, session, "lookup_glossary", internalmcp.LookupGlossaryInput{})
	if err != nil {
		t.Fatalf("lookup_glossary failed: %v", err)
	}
	if len(allTerms) != 1 || allTerms[0].ID != "tac" {
		t.Errorf("unexpected glossary terms: %+v", allTerms)
	}

	// 6. lookup_glossary (specific term)
	singleTerm, err := callToolJSON[[]internalmcp.GlossaryEntry](t, session, "lookup_glossary", internalmcp.LookupGlossaryInput{Term: "tac"})
	if err != nil {
		t.Fatalf("lookup_glossary for 'tac' failed: %v", err)
	}
	if len(singleTerm) != 1 || singleTerm[0].Title != "Tasks-as-Code" {
		t.Errorf("expected term 'tac', got %+v", singleTerm)
	}

	// 7. create_glossary_term
	createdTerm, err := callToolJSON[internalmcp.GlossaryEntry](t, session, "create_glossary_term", internalmcp.CreateGlossaryTermInput{
		Title:   "Single Executable",
		Summary: "Packaging frontend and backend into a zero-dependency binary",
		Tags:    []string{"feature"},
		Body:    "Go embed embeds the web UI directly.",
	})
	if err != nil {
		t.Fatalf("create_glossary_term failed: %v", err)
	}
	if createdTerm.ID != "single-executable" || createdTerm.Title != "Single Executable" {
		t.Errorf("unexpected created glossary term: %+v", createdTerm)
	}

	// 8. update_glossary_term
	updatedTerm, err := callToolJSON[internalmcp.GlossaryEntry](t, session, "update_glossary_term", internalmcp.UpdateGlossaryTermInput{
		ID:      "single-executable",
		Summary: "Updated single executable definition",
	})
	if err != nil {
		t.Fatalf("update_glossary_term failed: %v", err)
	}
	if updatedTerm.Summary != "Updated single executable definition" {
		t.Errorf("unexpected updated glossary term: %+v", updatedTerm)
	}
}

func TestMCP_SearchTagBoardTools(t *testing.T) {
	_, _, st, session := setupTestMCP(t)
	ctx := t.Context()

	// Seed task
	_ = st.UpsertTask(ctx, model.Task{
		ID:       "260901-sqlite",
		Title:    "SQLite In-Memory Store",
		Status:   "ready",
		Priority: model.PriorityHigh,
		Tags:     []string{"backend", "database"},
		Summary:  "Set up SQLite in memory using modernc.org/sqlite",
		Body:     "Details about SQLite in-memory store.",
	})

	// 1. search_tasks
	taskResults, err := callToolJSON[[]model.SearchResult](t, session, "search_tasks", internalmcp.SearchInput{Query: "SQLite"})
	if err != nil {
		t.Fatalf("search_tasks failed: %v", err)
	}
	if len(taskResults) == 0 || taskResults[0].ID != "260901-sqlite" {
		t.Errorf("expected search_tasks to match 260901-sqlite, got %+v", taskResults)
	}

	// 2. search_all
	allResults, err := callToolJSON[[]model.SearchResult](t, session, "search_all", internalmcp.SearchInput{Query: "SQLite"})
	if err != nil {
		t.Fatalf("search_all failed: %v", err)
	}
	if len(allResults) == 0 {
		t.Errorf("expected search_all to return results, got %+v", allResults)
	}

	// 3. list_tags
	tagList, err := callToolJSON[model.TagList](t, session, "list_tags", internalmcp.ListTagsInput{})
	if err != nil {
		t.Fatalf("list_tags failed: %v", err)
	}
	if !tagList.Enforced {
		t.Errorf("expected tags.enforced = true")
	}

	// 4. get_board_state
	board, err := callToolJSON[internalmcp.BoardSummaryOutput](t, session, "get_board_state", internalmcp.GetBoardStateInput{})
	if err != nil {
		t.Fatalf("get_board_state failed: %v", err)
	}
	if len(board.Columns) < 2 {
		t.Errorf("expected at least 2 columns, got %d", len(board.Columns))
	}
}

func TestMCP_StrategyAndGlossaryGuards(t *testing.T) {
	_, _, _, session := setupTestMCP(t)

	// 1. create_strategy guards
	t.Run("create_strategy validates title, summary, tier, and tags", func(t *testing.T) {
		// Empty title
		_, err := callToolJSON[internalmcp.StrategyDetail](t, session, "create_strategy", internalmcp.CreateStrategyInput{
			Summary: "Some summary",
			Tier:    1,
		})
		if err == nil {
			t.Error("expected error for empty title")
		}

		// Empty summary
		_, err = callToolJSON[internalmcp.StrategyDetail](t, session, "create_strategy", internalmcp.CreateStrategyInput{
			Title: "Some Title",
			Tier:  1,
		})
		if err == nil {
			t.Error("expected error for empty summary")
		}

		// Invalid tier (0)
		_, err = callToolJSON[internalmcp.StrategyDetail](t, session, "create_strategy", internalmcp.CreateStrategyInput{
			Title:   "Some Title",
			Summary: "Some summary",
			Tier:    0,
		})
		if err == nil {
			t.Error("expected error for tier 0")
		}

		// Invalid tier (4)
		_, err = callToolJSON[internalmcp.StrategyDetail](t, session, "create_strategy", internalmcp.CreateStrategyInput{
			Title:   "Some Title",
			Summary: "Some summary",
			Tier:    4,
		})
		if err == nil {
			t.Error("expected error for tier 4")
		}

		// Invalid tag when enforcement is enabled
		_, err = callToolJSON[internalmcp.StrategyDetail](t, session, "create_strategy", internalmcp.CreateStrategyInput{
			Title:   "Valid Title",
			Summary: "Valid summary",
			Tier:    1,
			Tags:    []string{"disallowed-tag"},
		})
		if err == nil {
			t.Error("expected error for disallowed tag")
		}
	})

	// 2. update_strategy guards
	t.Run("update_strategy validates non-existent id, tier, and tags", func(t *testing.T) {
		// Non-existent ID
		_, err := callToolJSON[internalmcp.StrategyDetail](t, session, "update_strategy", internalmcp.UpdateStrategyInput{
			ID:      "non-existent-strategy-id",
			Summary: "Updated summary",
		})
		if err == nil {
			t.Error("expected error for non-existent strategy id")
		}

		// First create a valid strategy
		created, err := callToolJSON[internalmcp.StrategyDetail](t, session, "create_strategy", internalmcp.CreateStrategyInput{
			Title:   "Test Guard Strategy",
			Summary: "Guard strategy summary",
			Tier:    2,
			Tags:    []string{"backend"},
		})
		if err != nil {
			t.Fatalf("failed to create strategy: %v", err)
		}

		// Update with invalid tier
		_, err = callToolJSON[internalmcp.StrategyDetail](t, session, "update_strategy", internalmcp.UpdateStrategyInput{
			ID:   created.ID,
			Tier: 99,
		})
		if err == nil {
			t.Error("expected error for updating with tier 99")
		}

		// Update with disallowed tag
		_, err = callToolJSON[internalmcp.StrategyDetail](t, session, "update_strategy", internalmcp.UpdateStrategyInput{
			ID:   created.ID,
			Tags: []string{"unauthorized-tag"},
		})
		if err == nil {
			t.Error("expected error for disallowed tag on update")
		}
	})

	// 3. create_glossary_term guards
	t.Run("create_glossary_term validates title, summary, and tags", func(t *testing.T) {
		// Empty title
		_, err := callToolJSON[internalmcp.GlossaryEntry](t, session, "create_glossary_term", internalmcp.CreateGlossaryTermInput{
			Summary: "Term summary",
		})
		if err == nil {
			t.Error("expected error for empty title")
		}

		// Empty summary
		_, err = callToolJSON[internalmcp.GlossaryEntry](t, session, "create_glossary_term", internalmcp.CreateGlossaryTermInput{
			Title: "Term Title",
		})
		if err == nil {
			t.Error("expected error for empty summary")
		}

		// Disallowed tag
		_, err = callToolJSON[internalmcp.GlossaryEntry](t, session, "create_glossary_term", internalmcp.CreateGlossaryTermInput{
			Title:   "Valid Term",
			Summary: "Valid summary",
			Tags:    []string{"disallowed-tag"},
		})
		if err == nil {
			t.Error("expected error for disallowed tag")
		}
	})

	// 4. update_glossary_term guards
	t.Run("update_glossary_term validates non-existent id and tags", func(t *testing.T) {
		// Non-existent ID
		_, err := callToolJSON[internalmcp.GlossaryEntry](t, session, "update_glossary_term", internalmcp.UpdateGlossaryTermInput{
			ID:      "non-existent-term-id",
			Summary: "Updated definition",
		})
		if err == nil {
			t.Error("expected error for non-existent glossary term id")
		}

		// First create a valid term
		created, err := callToolJSON[internalmcp.GlossaryEntry](t, session, "create_glossary_term", internalmcp.CreateGlossaryTermInput{
			Title:   "Guard Term",
			Summary: "Guard summary",
			Tags:    []string{"feature"},
		})
		if err != nil {
			t.Fatalf("failed to create glossary term: %v", err)
		}

		// Update with disallowed tag
		_, err = callToolJSON[internalmcp.GlossaryEntry](t, session, "update_glossary_term", internalmcp.UpdateGlossaryTermInput{
			ID:   created.ID,
			Tags: []string{"bad-tag"},
		})
		if err == nil {
			t.Error("expected error for disallowed tag on update")
		}
	})
}
