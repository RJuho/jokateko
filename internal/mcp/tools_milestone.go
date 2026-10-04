package mcp

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/RJuho/jokateko/internal/model"
	"github.com/RJuho/jokateko/internal/service"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// --- Milestone Tool Input / Output Models ---

type ListMilestonesInput struct {
	IncludeArchived bool `json:"include_archived,omitempty" jsonschema:"Set true to include closed or 100% completed milestones"`
}

type MilestoneSummary struct {
	ID                 string                `json:"id"`
	Title              string                `json:"title"`
	Status             model.MilestoneStatus `json:"status"`
	IsArchived         bool                  `json:"is_archived"`
	TargetDate         string                `json:"target_date,omitempty"`
	TargetStartAt      string                `json:"target_start_at,omitempty"`
	TargetEndAt        string                `json:"target_end_at,omitempty"`
	TargetTimeframe    string                `json:"target_timeframe,omitempty"`
	Tags               []string              `json:"tags"`
	Summary            string                `json:"summary"`
	TotalTasks         int                   `json:"total_tasks"`
	CompletedTasks     int                   `json:"completed_tasks"`
	ProgressPercentage float64               `json:"progress_percentage"`
}

type GetMilestoneInput struct {
	ID string `json:"id" jsonschema:"required,Milestone ID or slug (e.g. 260915-mvp)"`
}

type MilestoneDetail struct {
	ID                 string                `json:"id"`
	Title              string                `json:"title"`
	Status             model.MilestoneStatus `json:"status"`
	IsArchived         bool                  `json:"is_archived"`
	TargetDate         string                `json:"target_date,omitempty"`
	TargetStartAt      string                `json:"target_start_at,omitempty"`
	TargetEndAt        string                `json:"target_end_at,omitempty"`
	TargetTimeframe    string                `json:"target_timeframe,omitempty"`
	Tags               []string              `json:"tags"`
	Summary            string                `json:"summary"`
	Body               string                `json:"body"`
	TotalTasks         int                   `json:"total_tasks"`
	CompletedTasks     int                   `json:"completed_tasks"`
	ProgressPercentage float64               `json:"progress_percentage"`
	AssignedTasks      []string              `json:"assigned_tasks"`
	FilePath           string                `json:"file_path,omitempty"`
}

type CreateMilestoneInput struct {
	Title      string   `json:"title" jsonschema:"required,Descriptive milestone title"`
	TargetDate string   `json:"target_date,omitempty" jsonschema:"Target delivery date in ISO-8601 format (YYYY-MM-DD)"`
	Tags       []string `json:"tags,omitempty" jsonschema:"Categorization tags"`
	Summary    string   `json:"summary" jsonschema:"required,High-level deliverable summary"`
	Body       string   `json:"body,omitempty" jsonschema:"Detailed milestone roadmap markdown body"`
}

type UpdateMilestoneInput struct {
	ID         string    `json:"id" jsonschema:"required,Milestone ID or slug"`
	Status     *string   `json:"status,omitempty" jsonschema:"open or closed"`
	TargetDate *string   `json:"target_date,omitempty" jsonschema:"Target delivery date in ISO-8601 format (YYYY-MM-DD)"`
	Tags       *[]string `json:"tags,omitempty" jsonschema:"Categorization tags"`
	Summary    *string   `json:"summary,omitempty" jsonschema:"High-level summary"`
	Body       *string   `json:"body,omitempty" jsonschema:"Milestone roadmap markdown body"`
}

type DeleteMilestoneInput struct {
	ID    string `json:"id" jsonschema:"required,Milestone ID or slug"`
	Force bool   `json:"force,omitempty" jsonschema:"Force deletion even if tasks are assigned to this milestone"`
}

func (s *Server) registerMilestoneTools() {
	// 1. list_milestones
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "list_milestones",
		Description: "Lists milestones with completion progress. Auto-archives 100% completed milestones unless include_archived=true.",
	}, s.toolListMilestones)

	// 2. get_milestone
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "get_milestone",
		Description: "Fetches full milestone specification and all assigned task slugs.",
	}, s.toolGetMilestone)

	// 3. create_milestone
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "create_milestone",
		Description: "Creates a new milestone file in .jokateko/milestones/.",
	}, s.toolCreateMilestone)

	// 4. update_milestone
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "update_milestone",
		Description: "Updates milestone metadata (status, target date, summary, or body).",
	}, s.toolUpdateMilestone)

	// 5. delete_milestone
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "delete_milestone",
		Description: "Deletes a milestone file and removes it from the store. Rejects if tasks are assigned unless force=true.",
	}, s.toolDeleteMilestone)
}

func (s *Server) toolListMilestones(ctx context.Context, _ *mcp.CallToolRequest, in ListMilestonesInput) (*mcp.CallToolResult, []MilestoneSummary, error) {
	milestones, err := s.store.ListMilestones(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list milestones: %w", err)
	}

	summaries := make([]MilestoneSummary, 0, len(milestones))
	for _, ms := range milestones {
		archived := isArchived(ms)
		if !in.IncludeArchived && archived {
			continue
		}

		summaries = append(summaries, MilestoneSummary{
			ID:                 ms.ID,
			Title:              ms.Title,
			Status:             ms.Status,
			IsArchived:         archived,
			TargetDate:         ms.TargetDate,
			TargetStartAt:      ms.TargetStartAt,
			TargetEndAt:        ms.TargetEndAt,
			TargetTimeframe:    ms.TargetTimeframe,
			Tags:               ms.Tags,
			Summary:            ms.Summary,
			TotalTasks:         ms.TotalTasks,
			CompletedTasks:     ms.CompletedTasks,
			ProgressPercentage: ms.ProgressPercentage,
		})
	}

	return nil, summaries, nil
}

// milestoneDetail converts a milestone into the MCP detail view, including assigned task slugs.
func (s *Server) milestoneDetail(ctx context.Context, ms model.Milestone) *MilestoneDetail {
	taskSlugs := []string{}
	if assigned, err := s.store.ListTasks(ctx, model.FilterCriteria{Milestone: ms.ID}); err == nil {
		for _, t := range assigned {
			taskSlugs = append(taskSlugs, t.ID)
		}
	}

	return &MilestoneDetail{
		ID:                 ms.ID,
		Title:              ms.Title,
		Status:             ms.Status,
		IsArchived:         isArchived(ms),
		TargetDate:         ms.TargetDate,
		TargetStartAt:      ms.TargetStartAt,
		TargetEndAt:        ms.TargetEndAt,
		TargetTimeframe:    ms.TargetTimeframe,
		Tags:               ms.Tags,
		Summary:            ms.Summary,
		Body:               ms.Body,
		TotalTasks:         ms.TotalTasks,
		CompletedTasks:     ms.CompletedTasks,
		ProgressPercentage: ms.ProgressPercentage,
		AssignedTasks:      taskSlugs,
		FilePath:           ms.FilePath,
	}
}

func (s *Server) toolGetMilestone(ctx context.Context, _ *mcp.CallToolRequest, in GetMilestoneInput) (*mcp.CallToolResult, *MilestoneDetail, error) {
	id, err := requireID(in.ID, "milestone")
	if err != nil {
		return nil, nil, err
	}
	ms, err := s.svc.GetMilestone(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	return nil, s.milestoneDetail(ctx, ms), nil
}

func (s *Server) toolCreateMilestone(ctx context.Context, _ *mcp.CallToolRequest, in CreateMilestoneInput) (*mcp.CallToolResult, *MilestoneDetail, error) {
	if err := s.checkMutations(); err != nil {
		return nil, nil, err
	}
	if strings.TrimSpace(in.Summary) == "" {
		return nil, nil, errors.New("summary is required")
	}
	if err := s.svc.CheckTags(in.Tags); err != nil {
		return nil, nil, err
	}

	ms, err := s.svc.CreateMilestone(ctx, service.NewMilestone{
		Title:      in.Title,
		TargetDate: in.TargetDate,
		Tags:       in.Tags,
		Summary:    in.Summary,
		Body:       in.Body,
	})
	if err != nil {
		return nil, nil, err
	}
	return nil, s.milestoneDetail(ctx, ms), nil
}

func (s *Server) toolUpdateMilestone(ctx context.Context, _ *mcp.CallToolRequest, in UpdateMilestoneInput) (*mcp.CallToolResult, *MilestoneDetail, error) {
	if err := s.checkMutations(); err != nil {
		return nil, nil, err
	}
	id, err := requireID(in.ID, "milestone")
	if err != nil {
		return nil, nil, err
	}
	if in.Tags != nil {
		if err := s.svc.CheckTags(*in.Tags); err != nil {
			return nil, nil, err
		}
	}

	ms, err := s.svc.UpdateMilestone(ctx, id, func(ms *model.Milestone) error {
		if in.Status != nil {
			st, err := service.ParseMilestoneStatus(*in.Status)
			if err != nil {
				return err
			}
			ms.Status = st
		}
		if in.TargetDate != nil {
			ms.TargetDate = strings.TrimSpace(*in.TargetDate)
		}
		if in.Summary != nil {
			if v := strings.TrimSpace(*in.Summary); v != "" {
				ms.Summary = v
			}
		}
		if in.Tags != nil {
			ms.Tags = *in.Tags
		}
		if in.Body != nil {
			ms.Body = *in.Body
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return nil, s.milestoneDetail(ctx, ms), nil
}

func (s *Server) toolDeleteMilestone(ctx context.Context, _ *mcp.CallToolRequest, in DeleteMilestoneInput) (*mcp.CallToolResult, *DeleteEntityOutput, error) {
	if err := s.checkMutations(); err != nil {
		return nil, nil, err
	}
	id, err := requireID(in.ID, "milestone")
	if err != nil {
		return nil, nil, err
	}
	if err := s.svc.DeleteMilestone(ctx, id, in.Force); err != nil {
		return nil, nil, err
	}
	return nil, &DeleteEntityOutput{
		Success: true,
		ID:      id,
		Message: fmt.Sprintf("Milestone %q deleted successfully", id),
	}, nil
}
