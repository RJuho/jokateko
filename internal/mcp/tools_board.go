package mcp

import (
	"context"
	"fmt"

	"github.com/RJuho/jokateko/internal/model"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type GetBoardStateInput struct{}

type BoardColumnSummary struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
	Count int    `json:"count"`
}

type BoardSummaryOutput struct {
	ProjectName string               `json:"project_name"`
	Columns     []BoardColumnSummary `json:"columns"`
}

func (s *Server) registerBoardTools() {
	// 1. get_board_state
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "get_board_state",
		Description: "Returns Kanban workflow column hierarchy and aggregated task counts.",
	}, s.toolGetBoardState)
}

func (s *Server) toolGetBoardState(ctx context.Context, _ *mcp.CallToolRequest, _ GetBoardStateInput) (*mcp.CallToolResult, *BoardSummaryOutput, error) {
	cols := make([]model.Column, 0, len(s.cfg.Board.Columns))
	for _, c := range s.cfg.Board.Columns {
		cols = append(cols, model.Column{
			ID:    c.ID,
			Name:  c.Name,
			Color: c.Color,
		})
	}

	board, err := s.store.GetBoardState(ctx, s.cfg.Project.Name, cols, model.FilterCriteria{})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get board state: %w", err)
	}

	colSummaries := make([]BoardColumnSummary, 0, len(board.Columns))
	for _, c := range board.Columns {
		colSummaries = append(colSummaries, BoardColumnSummary{
			ID:    c.ID,
			Name:  c.Name,
			Color: c.Color,
			Count: c.Count,
		})
	}

	return nil, &BoardSummaryOutput{
		ProjectName: board.ProjectName,
		Columns:     colSummaries,
	}, nil
}
