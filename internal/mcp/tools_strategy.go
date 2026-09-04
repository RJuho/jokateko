package mcp

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/RJuho/jokateko/internal/model"
	"github.com/RJuho/jokateko/internal/parser"
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

func (s *Server) toolCreateStrategy(ctx context.Context, _ *mcp.CallToolRequest, in CreateStrategyInput) (*mcp.CallToolResult, *StrategyDetail, error) {
	title := strings.TrimSpace(in.Title)
	if title == "" {
		return nil, nil, errors.New("title is required")
	}

	summary := strings.TrimSpace(in.Summary)
	if summary == "" {
		return nil, nil, errors.New("summary is required")
	}

	tier := model.Tier(in.Tier)
	if !tier.IsValid() {
		return nil, nil, fmt.Errorf("invalid tier %d: tier must be 1 (Core Invariants), 2 (Domain Patterns), or 3 (Implementation Specs)", in.Tier)
	}

	if s.cfg.Tags.EnforceAllowed && len(in.Tags) > 0 {
		for _, tag := range in.Tags {
			if !slices.Contains(s.cfg.Tags.Allowed, tag) {
				return nil, nil, fmt.Errorf("tag %q is not permitted. Allowed tags: %v", tag, s.cfg.Tags.Allowed)
			}
		}
	}

	tags := in.Tags
	if tags == nil {
		tags = []string{}
	}

	id := slugify(title)
	fm := model.StrategyFrontmatter{
		Title:   title,
		Tier:    tier,
		Summary: summary,
		Tags:    tags,
	}

	fileBytes, err := parser.Format(fm, in.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to format strategy markdown: %w", err)
	}

	filePath := filepath.Join(s.StrategiesDir(), fmt.Sprintf("%s.md", id))
	if err := s.writer.WriteFile(filePath, fileBytes, 0644); err != nil {
		return nil, nil, fmt.Errorf("failed to save strategy file: %w", err)
	}

	strat := model.Strategy{
		ID:       id,
		Title:    title,
		Tier:     tier,
		Summary:  summary,
		Tags:     tags,
		Body:     in.Body,
		FilePath: filePath,
		ModTime:  time.Now(),
	}

	_ = s.store.UpsertStrategy(ctx, strat)

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

func (s *Server) toolUpdateStrategy(ctx context.Context, _ *mcp.CallToolRequest, in UpdateStrategyInput) (*mcp.CallToolResult, *StrategyDetail, error) {
	id := strings.TrimSpace(in.ID)
	if id == "" {
		return nil, nil, errors.New("strategy id is required")
	}

	existing, err := s.store.GetStrategy(ctx, id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, nil, fmt.Errorf("strategy %q not found", id)
		}
		return nil, nil, fmt.Errorf("failed to get strategy %q: %w", id, err)
	}

	title := existing.Title
	if strings.TrimSpace(in.Title) != "" {
		title = strings.TrimSpace(in.Title)
	}

	tier := existing.Tier
	if in.Tier > 0 {
		t := model.Tier(in.Tier)
		if !t.IsValid() {
			return nil, nil, fmt.Errorf("invalid tier %d: tier must be 1, 2, or 3", in.Tier)
		}
		tier = t
	}

	summary := existing.Summary
	if strings.TrimSpace(in.Summary) != "" {
		summary = strings.TrimSpace(in.Summary)
	}

	tags := existing.Tags
	if in.Tags != nil {
		if s.cfg.Tags.EnforceAllowed && len(in.Tags) > 0 {
			for _, tag := range in.Tags {
				if !slices.Contains(s.cfg.Tags.Allowed, tag) {
					return nil, nil, fmt.Errorf("tag %q is not permitted. Allowed tags: %v", tag, s.cfg.Tags.Allowed)
				}
			}
		}
		tags = in.Tags
	}

	body := existing.Body
	if in.Body != "" {
		body = in.Body
	}

	fm := model.StrategyFrontmatter{
		Title:   title,
		Tier:    tier,
		Summary: summary,
		Tags:    tags,
	}

	fileBytes, err := parser.Format(fm, body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to format strategy markdown: %w", err)
	}

	filePath := existing.FilePath
	if filePath == "" {
		filePath = filepath.Join(s.StrategiesDir(), fmt.Sprintf("%s.md", id))
	}

	if err := s.writer.WriteFile(filePath, fileBytes, 0644); err != nil {
		return nil, nil, fmt.Errorf("failed to save strategy file: %w", err)
	}

	updated := model.Strategy{
		ID:       id,
		Title:    title,
		Tier:     tier,
		Summary:  summary,
		Tags:     tags,
		Body:     body,
		FilePath: filePath,
		ModTime:  time.Now(),
	}

	_ = s.store.UpsertStrategy(ctx, updated)

	return nil, &StrategyDetail{
		ID:       updated.ID,
		Title:    updated.Title,
		Tier:     updated.Tier,
		Tags:     updated.Tags,
		Summary:  updated.Summary,
		Body:     updated.Body,
		FilePath: updated.FilePath,
	}, nil
}

func (s *Server) toolDeleteStrategy(ctx context.Context, _ *mcp.CallToolRequest, in DeleteStrategyInput) (*mcp.CallToolResult, *DeleteEntityOutput, error) {
	if !s.cfg.MCP.AllowMutations {
		return nil, nil, errors.New("mutations are disabled in configuration")
	}

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

	if strat.FilePath != "" {
		if err := s.writer.RemoveFile(strat.FilePath); err != nil {
			return nil, nil, fmt.Errorf("failed to remove strategy file: %w", err)
		}
	}

	if err := s.store.DeleteStrategy(ctx, id); err != nil {
		return nil, nil, fmt.Errorf("failed to delete strategy from store: %w", err)
	}

	return nil, &DeleteEntityOutput{
		Success: true,
		ID:      id,
		Message: fmt.Sprintf("Strategy %q deleted successfully", id),
	}, nil
}

