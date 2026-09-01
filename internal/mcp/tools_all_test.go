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

	// 3. lookup_glossary (all)
	allTerms, err := callToolJSON[[]internalmcp.GlossaryEntry](t, session, "lookup_glossary", internalmcp.LookupGlossaryInput{})
	if err != nil {
		t.Fatalf("lookup_glossary failed: %v", err)
	}
	if len(allTerms) != 1 || allTerms[0].ID != "tac" {
		t.Errorf("unexpected glossary terms: %+v", allTerms)
	}

	// 4. lookup_glossary (specific term)
	singleTerm, err := callToolJSON[[]internalmcp.GlossaryEntry](t, session, "lookup_glossary", internalmcp.LookupGlossaryInput{Term: "tac"})
	if err != nil {
		t.Fatalf("lookup_glossary for 'tac' failed: %v", err)
	}
	if len(singleTerm) != 1 || singleTerm[0].Title != "Tasks-as-Code" {
		t.Errorf("expected term 'tac', got %+v", singleTerm)
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
