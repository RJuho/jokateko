package mcp

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/RJuho/jokateko/internal/model"
	"github.com/RJuho/jokateko/internal/service"
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

type CreateStrategyInput struct {
	Title   string   `json:"title" jsonschema:"required,Descriptive title for the architectural guideline"`
	Tier    int      `json:"tier" jsonschema:"required,Architectural tier (1: Core Invariants, 2: Domain Patterns, 3: Implementation Specs)"`
	Summary string   `json:"summary" jsonschema:"required,1-2 sentence high-level summary"`
	Tags    []string `json:"tags,omitempty" jsonschema:"Categorization tags"`
	Body    string   `json:"body,omitempty" jsonschema:"Full markdown body detailing architectural guidelines, rules, and invariants"`
}

type UpdateStrategyInput struct {
	ID      string   `json:"id" jsonschema:"required,Strategy slug (e.g. architecture, pure-go-dependencies)"`
	Title   string   `json:"title,omitempty" jsonschema:"Updated strategy title"`
	Tier    int      `json:"tier,omitempty" jsonschema:"Updated architectural tier (1, 2, or 3)"`
	Summary string   `json:"summary,omitempty" jsonschema:"Updated 1-2 sentence summary"`
	Tags    []string `json:"tags,omitempty" jsonschema:"Updated categorization tags"`
	Body    string   `json:"body,omitempty" jsonschema:"Updated markdown body"`
}

type DeleteStrategyInput struct {
	ID string `json:"id" jsonschema:"required,Strategy ID or slug (e.g. architecture, pure-go-dependencies)"`
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

	// 3. create_strategy
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "create_strategy",
		Description: "Creates a new architectural strategy markdown file inside .jokateko/strategies/.",
	}, s.toolCreateStrategy)

	// 4. update_strategy
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "update_strategy",
		Description: "Updates an existing architectural strategy's metadata or markdown body.",
	}, s.toolUpdateStrategy)

	// 5. delete_strategy
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "delete_strategy",
		Description: "Deletes an architectural strategy file and removes it from the store.",
	}, s.toolDeleteStrategy)
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

func toStrategyDetail(st model.Strategy) *StrategyDetail {
	return &StrategyDetail{
		ID:       st.ID,
		Title:    st.Title,
		Tier:     st.Tier,
		Tags:     st.Tags,
		Summary:  st.Summary,
		Body:     st.Body,
		FilePath: st.FilePath,
	}
}

func (s *Server) toolGetStrategy(ctx context.Context, _ *mcp.CallToolRequest, in GetStrategyInput) (*mcp.CallToolResult, *StrategyDetail, error) {
	id, err := requireID(in.ID, "strategy")
	if err != nil {
		return nil, nil, err
	}
	strat, err := s.svc.GetStrategy(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	return nil, toStrategyDetail(strat), nil
}

func (s *Server) toolCreateStrategy(ctx context.Context, _ *mcp.CallToolRequest, in CreateStrategyInput) (*mcp.CallToolResult, *StrategyDetail, error) {
	if err := s.checkMutations(); err != nil {
		return nil, nil, err
	}
	if strings.TrimSpace(in.Summary) == "" {
		return nil, nil, errors.New("summary is required")
	}
	// The tier is mandatory for agents: reject 0 instead of defaulting to Core.
	if err := service.CheckTier(model.Tier(in.Tier)); err != nil {
		return nil, nil, err
	}
	strat, err := s.svc.CreateStrategy(ctx, service.NewStrategy{
		Title:   in.Title,
		Tier:    model.Tier(in.Tier),
		Tags:    in.Tags,
		Summary: in.Summary,
		Body:    in.Body,
	})
	if err != nil {
		return nil, nil, err
	}
	return nil, toStrategyDetail(strat), nil
}

func (s *Server) toolUpdateStrategy(ctx context.Context, _ *mcp.CallToolRequest, in UpdateStrategyInput) (*mcp.CallToolResult, *StrategyDetail, error) {
	if err := s.checkMutations(); err != nil {
		return nil, nil, err
	}
	id, err := requireID(in.ID, "strategy")
	if err != nil {
		return nil, nil, err
	}
	if in.Tier > 0 {
		if err := service.CheckTier(model.Tier(in.Tier)); err != nil {
			return nil, nil, err
		}
	}

	strat, err := s.svc.UpdateStrategy(ctx, id, func(st *model.Strategy) error {
		if v := strings.TrimSpace(in.Title); v != "" {
			st.Title = v
		}
		if in.Tier > 0 {
			st.Tier = model.Tier(in.Tier)
		}
		if v := strings.TrimSpace(in.Summary); v != "" {
			st.Summary = v
		}
		if in.Tags != nil {
			st.Tags = in.Tags
		}
		if in.Body != "" {
			st.Body = in.Body
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return nil, toStrategyDetail(strat), nil
}

func (s *Server) toolDeleteStrategy(ctx context.Context, _ *mcp.CallToolRequest, in DeleteStrategyInput) (*mcp.CallToolResult, *DeleteEntityOutput, error) {
	if err := s.checkMutations(); err != nil {
		return nil, nil, err
	}
	id, err := requireID(in.ID, "strategy")
	if err != nil {
		return nil, nil, err
	}
	if err := s.svc.DeleteStrategy(ctx, id); err != nil {
		return nil, nil, err
	}
	return nil, &DeleteEntityOutput{
		Success: true,
		ID:      id,
		Message: fmt.Sprintf("Strategy %q deleted successfully", id),
	}, nil
}
