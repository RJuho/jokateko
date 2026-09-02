package mcp

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/RJuho/jokateko/internal/model"
	"github.com/RJuho/jokateko/internal/store"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type ListStrategiesInput struct {
	Tier int    `json:"tier,omitempty" jsonschema:"Filter by architectural tier (1: Core, 2: Domain, 3: Implementation)"`
	Tag  string `json:"tag,omitempty" jsonschema:"Filter by topic tag"`
}

type StrategySummary struct {
	ID      string     `json:"id"`
	Title   string     `json:"title"`
	Tier    model.Tier `json:"tier"`
	Tags    []string   `json:"tags"`
	Summary string     `json:"summary"`
}

type GetStrategyInput struct {
	ID string `json:"id" jsonschema:"required,Strategy slug (e.g. architecture, zero-cgo)"`
}

type StrategyDetail struct {
	ID       string     `json:"id"`
	Title    string     `json:"title"`
	Tier     model.Tier `json:"tier"`
	Tags     []string   `json:"tags"`
	Summary  string     `json:"summary"`
	Body     string     `json:"body"`
	FilePath string     `json:"file_path,omitempty"`
}

func (s *Server) registerStrategyTools() {
	// 1. list_strategies
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "list_strategies",
		Description: "Returns high-level summaries and tiers of architectural guidelines for progressive disclosure. See jokateko://strategies/tiers for tier specifications.",
	}, s.toolListStrategies)

	// 2. get_strategy
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "get_strategy",
		Description: "Retrieves the full markdown document of a specific architectural strategy.",
	}, s.toolGetStrategy)
}

func (s *Server) toolListStrategies(ctx context.Context, _ *mcp.CallToolRequest, in ListStrategiesInput) (*mcp.CallToolResult, []StrategySummary, error) {
	strats, err := s.store.ListStrategies(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list strategies: %w", err)
	}

	filterTag := strings.TrimSpace(in.Tag)
	summaries := make([]StrategySummary, 0, len(strats))
	for _, st := range strats {
		if in.Tier > 0 && int(st.Tier) != in.Tier {
			continue
		}
		if filterTag != "" && !slices.Contains(st.Tags, filterTag) {
			continue
		}

		summaries = append(summaries, StrategySummary{
			ID:      st.ID,
			Title:   st.Title,
			Tier:    st.Tier,
			Tags:    st.Tags,
			Summary: st.Summary,
		})
	}

	return nil, summaries, nil
}

func (s *Server) toolGetStrategy(ctx context.Context, _ *mcp.CallToolRequest, in GetStrategyInput) (*mcp.CallToolResult, *StrategyDetail, error) {
	id := strings.TrimSpace(in.ID)
	if id == "" {
		return nil, nil, errors.New("strategy id is required")
	}

	strat, err := s.store.GetStrategy(ctx, id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, nil, fmt.Errorf("strategy %q not found", id)
		}
		return nil, nil, fmt.Errorf("failed to get strategy %q: %w", id, err)
	}

	return nil, &StrategyDetail{
		ID:       strat.ID,
		Title:    strat.Title,
		Tier:     strat.Tier,
		Tags:     strat.Tags,
		Summary:  strat.Summary,
		Body:     strat.Body,
		FilePath: strat.FilePath,
	}, nil
}
