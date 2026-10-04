package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/RJuho/jokateko/internal/model"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func (s *Server) registerResources() {
	// 1. jokateko://board
	s.mcpServer.AddResource(&mcp.Resource{
		URI:         "jokateko://board",
		Name:        "Kanban Board State",
		Description: "Live snapshot of Kanban board columns and tasks",
		MIMEType:    "application/json",
	}, s.resourceBoard)

	// 2. jokateko://strategies/tier1
	s.mcpServer.AddResource(&mcp.Resource{
		URI:         "jokateko://strategies/tier1",
		Name:        "Tier-1 Core Architectural Guidelines",
		Description: "Concatenation of all Tier-1 core architectural rules and constraints",
		MIMEType:    "text/markdown",
	}, s.resourceTier1Strategies)

	// 3. jokateko://strategies/tiers
	s.mcpServer.AddResource(&mcp.Resource{
		URI:         "jokateko://strategies/tiers",
		Name:        "Architectural Strategy Tiers Specification",
		Description: "Configured progressive disclosure tiers, titles, summaries, and scopes for architectural guidelines",
		MIMEType:    "text/markdown",
	}, s.resourceStrategyTiers)

	// 4. jokateko://glossary
	s.mcpServer.AddResource(&mcp.Resource{
		URI:         "jokateko://glossary",
		Name:        "Project Glossary Dictionary",
		Description: "Standardized project terminology definitions and domain concepts",
		MIMEType:    "text/markdown",
	}, s.resourceGlossary)
}

func (s *Server) resourceBoard(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	cols := s.cfg.Columns()

	board, err := s.store.GetBoardState(ctx, s.cfg.Project.Name, cols, model.FilterCriteria{})
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve board state: %w", err)
	}

	bytes, err := json.MarshalIndent(board, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal board state: %w", err)
	}

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{
				URI:      req.Params.URI,
				MIMEType: "application/json",
				Text:     string(bytes),
			},
		},
	}, nil
}

func (s *Server) resourceTier1Strategies(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	strats, err := s.store.ListStrategies(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list strategies: %w", err)
	}

	var sb strings.Builder
	sb.WriteString("# Tier-1 Core Architectural Guidelines\n\n")

	count := 0
	for _, st := range strats {
		if st.Tier == model.TierCore {
			count++
			fmt.Fprintf(&sb, "## %s (`%s`)\n\n", st.Title, st.ID)
			if st.Summary != "" {
				fmt.Fprintf(&sb, "> %s\n\n", st.Summary)
			}
			if st.Body != "" {
				sb.WriteString(st.Body)
				sb.WriteString("\n\n")
			}
		}
	}

	if count == 0 {
		sb.WriteString("No Tier-1 strategies defined.\n")
	}

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{
				URI:      req.Params.URI,
				MIMEType: "text/markdown",
				Text:     sb.String(),
			},
		},
	}, nil
}

func (s *Server) resourceStrategyTiers(_ context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	var sb strings.Builder
	sb.WriteString("# Architectural Strategy Tiers (Progressive Disclosure)\n\n")
	sb.WriteString("Architectural strategies are structured into progressive disclosure tiers to prevent agent token exhaustion.\n\n")

	for _, tr := range s.cfg.Strategies.Tiers {
		fmt.Fprintf(&sb, "## %s: %s (`tier = %s`)\n", tr.Name, tr.Title, tr.ID)
		if tr.Summary != "" {
			fmt.Fprintf(&sb, "%s\n", tr.Summary)
		}
		sb.WriteString("\n")
	}

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{
				URI:      req.Params.URI,
				MIMEType: "text/markdown",
				Text:     sb.String(),
			},
		},
	}, nil
}

func (s *Server) resourceGlossary(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	terms, err := s.store.ListGlossaryTerms(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list glossary terms: %w", err)
	}

	var sb strings.Builder
	sb.WriteString("# Project Glossary\n\n")

	if len(terms) == 0 {
		sb.WriteString("No glossary terms defined.\n")
	} else {
		for _, t := range terms {
			fmt.Fprintf(&sb, "### %s (`%s`)\n\n", t.Title, t.ID)
			if t.Summary != "" {
				fmt.Fprintf(&sb, "%s\n\n", t.Summary)
			}
			if t.Body != "" {
				sb.WriteString(t.Body)
				sb.WriteString("\n\n")
			}
		}
	}

	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{
				URI:      req.Params.URI,
				MIMEType: "text/markdown",
				Text:     sb.String(),
			},
		},
	}, nil
}
