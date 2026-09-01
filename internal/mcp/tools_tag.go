package mcp

import (
	"context"
	"fmt"

	"github.com/RJuho/jokateko/internal/model"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type ListTagsInput struct{}

func (s *Server) registerTagTools() {
	// 1. list_tags
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "list_tags",
		Description: "Lists controlled tag vocabulary and usage counts across tasks, milestones, and strategies.",
	}, s.toolListTags)
}

func (s *Server) toolListTags(ctx context.Context, _ *mcp.CallToolRequest, _ ListTagsInput) (*mcp.CallToolResult, *model.TagList, error) {
	tagCounts, err := s.store.GetTagCounts(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get tag counts: %w", err)
	}

	return nil, &model.TagList{
		Enforced: s.cfg.Tags.EnforceAllowed,
		Tags:     tagCounts,
	}, nil
}
