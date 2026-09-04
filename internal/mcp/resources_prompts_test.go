package mcp_test

import (
	"strings"
	"testing"

	"github.com/RJuho/jokateko/internal/config"
	internalmcp "github.com/RJuho/jokateko/internal/mcp"
	"github.com/RJuho/jokateko/internal/model"
	"github.com/RJuho/jokateko/internal/store"
	"github.com/RJuho/jokateko/internal/writer"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestMCP_Resources(t *testing.T) {
	_, _, st, session := setupTestMCP(t)
	ctx := t.Context()

	// Seed data
	_ = st.UpsertStrategy(ctx, model.Strategy{
		ID:      "zero-cgo",
		Title:   "Zero CGO Architecture",
		Tier:    model.TierCore,
		Tags:    []string{"backend"},
		Summary: "Strict zero-CGO policy",
		Body:    "All packages must compile with CGO_ENABLED=0.",
	})

	_ = st.UpsertGlossaryTerm(ctx, model.GlossaryTerm{
		ID:      "tac",
		Title:   "Tasks-as-Code",
		Tags:    []string{"feature"},
		Summary: "Managing tasks directly as Markdown files",
		Body:    "Full definition of Tasks-as-Code.",
	})

	_ = st.UpsertTask(ctx, model.Task{
		ID:       "260901-sqlite",
		Title:    "SQLite Store",
		Status:   "ready",
		Priority: model.PriorityHigh,
		Summary:  "Summary",
		Body:     "Body",
	})

	// 1. Read jokateko://board
	resBoard, err := session.ReadResource(ctx, &mcp.ReadResourceParams{
		URI: "jokateko://board",
	})
	if err != nil {
		t.Fatalf("ReadResource(jokateko://board) failed: %v", err)
	}
	if len(resBoard.Contents) == 0 {
		t.Fatal("expected contents in jokateko://board")
	}
	if !strings.Contains(resBoard.Contents[0].Text, "260901-sqlite") {
		t.Errorf("expected board resource to contain task 260901-sqlite, got:\n%s", resBoard.Contents[0].Text)
	}

	// 2. Read jokateko://strategies/tier1
	resTier1, err := session.ReadResource(ctx, &mcp.ReadResourceParams{
		URI: "jokateko://strategies/tier1",
	})
	if err != nil {
		t.Fatalf("ReadResource(jokateko://strategies/tier1) failed: %v", err)
	}
	if len(resTier1.Contents) == 0 {
		t.Fatal("expected contents in jokateko://strategies/tier1")
	}
	if !strings.Contains(resTier1.Contents[0].Text, "Zero CGO Architecture") {
		t.Errorf("expected tier1 resource to contain Zero CGO Architecture, got:\n%s", resTier1.Contents[0].Text)
	}

	// 3. Read jokateko://glossary
	resGlossary, err := session.ReadResource(ctx, &mcp.ReadResourceParams{
		URI: "jokateko://glossary",
	})
	if err != nil {
		t.Fatalf("ReadResource(jokateko://glossary) failed: %v", err)
	}
	if len(resGlossary.Contents) == 0 {
		t.Fatal("expected contents in jokateko://glossary")
	}
	if !strings.Contains(resGlossary.Contents[0].Text, "Tasks-as-Code") {
		t.Errorf("expected glossary resource to contain Tasks-as-Code, got:\n%s", resGlossary.Contents[0].Text)
	}
}

func TestMCP_Prompts(t *testing.T) {
	_, _, st, session := setupTestMCP(t)
	ctx := t.Context()

	// Seed Tier-1 strategy
	_ = st.UpsertStrategy(ctx, model.Strategy{
		ID:      "zero-cgo",
		Title:   "Zero CGO Architecture",
		Tier:    model.TierCore,
		Summary: "All Go code must compile without CGO",
	})

	// Seed a blocker task (status: ready)
	_ = st.UpsertTask(ctx, model.Task{
		ID:       "task-blocker",
		Title:    "Blocker Task",
		Status:   "ready",
		Priority: model.PriorityMedium,
		Summary:  "Medium priority blocker",
		Body:     "## Description\nBlocker description.",
	})

	// Seed a blocked high priority task
	_ = st.UpsertTask(ctx, model.Task{
		ID:           "task-blocked-high",
		Title:        "High Priority But Blocked",
		Status:       "ready",
		Priority:     model.PriorityHigh,
		Dependencies: []string{"task-blocker"},
		Summary:      "Blocked task",
		Body:         "## Description\nBlocked description.",
	})

	// Get prompt next_task: blocker task should be recommended because high priority task is blocked!
	promptRes, err := session.GetPrompt(ctx, &mcp.GetPromptParams{
		Name: "next_task",
	})
	if err != nil {
		t.Fatalf("GetPrompt(next_task) failed: %v", err)
	}
	if len(promptRes.Messages) == 0 {
		t.Fatal("expected prompt messages")
	}
	text := promptRes.Messages[0].Content.(*mcp.TextContent).Text
	if !strings.Contains(text, "task-blocker") {
		t.Errorf("expected prompt to recommend task-blocker (unblocked), got:\n%s", text)
	}
	if strings.Contains(text, "task-blocked-high") {
		t.Errorf("prompt should not recommend task-blocked-high because it is blocked")
	}
	if !strings.Contains(text, "Zero CGO Architecture") {
		t.Errorf("expected prompt to include Tier-1 strategy guideline")
	}

	// Now complete task-blocker
	_ = st.UpsertTask(ctx, model.Task{
		ID:       "task-blocker",
		Title:    "Blocker Task",
		Status:   "done",
		Priority: model.PriorityMedium,
		Summary:  "Completed blocker",
	})

	// Now next_task should recommend task-blocked-high because its blocker is done!
	promptRes2, err := session.GetPrompt(ctx, &mcp.GetPromptParams{
		Name: "next_task",
	})
	if err != nil {
		t.Fatalf("GetPrompt(next_task) after unblocking failed: %v", err)
	}
	text2 := promptRes2.Messages[0].Content.(*mcp.TextContent).Text
	if !strings.Contains(text2, "task-blocked-high") {
		t.Errorf("expected prompt to now recommend task-blocked-high, got:\n%s", text2)
	}
}

func TestMCP_Instructions(t *testing.T) {
	server, _, _, _ := setupTestMCP(t)

	// 1. Default instructions should match DefaultInstructions
	if server.Instructions() != internalmcp.DefaultInstructions {
		t.Errorf("expected default instructions, got:\n%s", server.Instructions())
	}
	if !strings.Contains(server.Instructions(), "All modifications to .jokateko/ files must be performed via Jokateko MCP tools") {
		t.Errorf("expected default instructions to contain MCP mutation guidance")
	}

	// 2. Custom instructions override default
	dir := t.TempDir()
	cfg := config.Default(dir)
	cfg.MCP.Instructions = "Custom project guidance for AI agents."
	st, err := store.OpenMemory()
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	wr := writer.New(nil)
	customServer := internalmcp.New(cfg, dir, st, wr)
	if customServer.Instructions() != "Custom project guidance for AI agents." {
		t.Errorf("expected custom instructions, got: %q", customServer.Instructions())
	}
}

