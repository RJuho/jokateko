package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/RJuho/jokateko/internal/model"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type SearchInput struct {
	Query string `json:"query" jsonschema:"required,Keywords or phrase to search"`
	Tag   string `json:"tag,omitempty" jsonschema:"Restrict search results to entities with this tag"`
	Limit int    `json:"limit,omitempty" jsonschema:"Maximum number of results to return (default: 20)"`
}

func (s *Server) registerSearchTools() {
	// 1. search_tasks
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "search_tasks",
		Description: "Searches task titles, summaries, acceptance criteria, and completion notes using SQLite FTS5.",
	}, s.toolSearchTasks)

	// 2. search_milestones
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "search_milestones",
		Description: "Searches milestone titles, goals, and summaries using SQLite FTS5.",
	}, s.toolSearchMilestones)

	// 3. search_strategies
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "search_strategies",
		Description: "Searches architectural guidelines and engineering rules using SQLite FTS5.",
	}, s.toolSearchStrategies)

	// 4. search_glossary
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "search_glossary",
		Description: "Searches standardized project terminology and definitions using SQLite FTS5.",
	}, s.toolSearchGlossary)

	// 5. search_all
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "search_all",
		Description: "Universal search across all document types (tasks, milestones, strategies, glossary) using SQLite FTS5.",
	}, s.toolSearchAll)
}

func resolveLimit(l int) int {
	if l <= 0 {
		return 20
	}
	if l > 100 {
		return 100
	}
	return l
}

func (s *Server) toolSearchTasks(ctx context.Context, _ *mcp.CallToolRequest, in SearchInput) (*mcp.CallToolResult, []model.SearchResult, error) {
	q := strings.TrimSpace(in.Query)
	res, err := s.store.SearchTasks(ctx, q, in.Tag, resolveLimit(in.Limit))
	if err != nil {
		return nil, nil, fmt.Errorf("task search failed: %w", err)
	}
	if res == nil {
		res = []model.SearchResult{}
	}
	return nil, res, nil
}

func (s *Server) toolSearchMilestones(ctx context.Context, _ *mcp.CallToolRequest, in SearchInput) (*mcp.CallToolResult, []model.SearchResult, error) {
	q := strings.TrimSpace(in.Query)
	res, err := s.store.SearchMilestones(ctx, q, in.Tag, resolveLimit(in.Limit))
	if err != nil {
		return nil, nil, fmt.Errorf("milestone search failed: %w", err)
	}
	if res == nil {
		res = []model.SearchResult{}
	}
	return nil, res, nil
}

func (s *Server) toolSearchStrategies(ctx context.Context, _ *mcp.CallToolRequest, in SearchInput) (*mcp.CallToolResult, []model.SearchResult, error) {
	q := strings.TrimSpace(in.Query)
	res, err := s.store.SearchStrategies(ctx, q, in.Tag, resolveLimit(in.Limit))
	if err != nil {
		return nil, nil, fmt.Errorf("strategy search failed: %w", err)
	}
	if res == nil {
		res = []model.SearchResult{}
	}
	return nil, res, nil
}

func (s *Server) toolSearchGlossary(ctx context.Context, _ *mcp.CallToolRequest, in SearchInput) (*mcp.CallToolResult, []model.SearchResult, error) {
	q := strings.TrimSpace(in.Query)
	res, err := s.store.SearchGlossary(ctx, q, in.Tag, resolveLimit(in.Limit))
	if err != nil {
		return nil, nil, fmt.Errorf("glossary search failed: %w", err)
	}
	if res == nil {
		res = []model.SearchResult{}
	}
	return nil, res, nil
}

func (s *Server) toolSearchAll(ctx context.Context, _ *mcp.CallToolRequest, in SearchInput) (*mcp.CallToolResult, []model.SearchResult, error) {
	q := strings.TrimSpace(in.Query)
	res, err := s.store.SearchAll(ctx, q, in.Tag, resolveLimit(in.Limit))
	if err != nil {
		return nil, nil, fmt.Errorf("universal search failed: %w", err)
	}
	if res == nil {
		res = []model.SearchResult{}
	}
	return nil, res, nil
}
