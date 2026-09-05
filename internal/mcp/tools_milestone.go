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
		isArchived := ms.Status == model.MilestoneStatusClosed || (ms.TotalTasks > 0 && ms.CompletedTasks == ms.TotalTasks)
		if !in.IncludeArchived && isArchived {
			continue
		}

		summaries = append(summaries, MilestoneSummary{
			ID:                 ms.ID,
			Title:              ms.Title,
			Status:             ms.Status,
			IsArchived:         isArchived,
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

func (s *Server) toolGetMilestone(ctx context.Context, _ *mcp.CallToolRequest, in GetMilestoneInput) (*mcp.CallToolResult, *MilestoneDetail, error) {
	id := strings.TrimSpace(in.ID)
	if id == "" {
		return nil, nil, errors.New("milestone id is required")
	}

	ms, err := s.store.GetMilestone(ctx, id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, nil, fmt.Errorf("milestone %q not found", id)
		}
		return nil, nil, fmt.Errorf("failed to get milestone %q: %w", id, err)
	}

	assignedTasks, err := s.store.ListTasks(ctx, model.FilterCriteria{Milestone: id})
	taskSlugs := make([]string, 0, len(assignedTasks))
	if err == nil {
		for _, t := range assignedTasks {
			taskSlugs = append(taskSlugs, t.ID)
		}
	}

	isArchived := ms.Status == model.MilestoneStatusClosed || (ms.TotalTasks > 0 && ms.CompletedTasks == ms.TotalTasks)

	return nil, &MilestoneDetail{
		ID:                 ms.ID,
		Title:              ms.Title,
		Status:             ms.Status,
		IsArchived:         isArchived,
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
	}, nil
}

func (s *Server) toolCreateMilestone(ctx context.Context, _ *mcp.CallToolRequest, in CreateMilestoneInput) (*mcp.CallToolResult, *MilestoneDetail, error) {
	title := strings.TrimSpace(in.Title)
	if title == "" {
		return nil, nil, errors.New("title is required")
	}

	summary := strings.TrimSpace(in.Summary)
	if summary == "" {
		return nil, nil, errors.New("summary is required")
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

	targetDate := strings.TrimSpace(in.TargetDate)
	id := fmt.Sprintf("%s-%s", time.Now().Format("060102"), slugify(title))

	fm := model.MilestoneFrontmatter{
		Title:      title,
		Status:     model.MilestoneStatusOpen,
		TargetDate: targetDate,
		Tags:       tags,
		Summary:    summary,
	}

	fileBytes, err := parser.Format(fm, in.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to format milestone markdown: %w", err)
	}

	filePath := filepath.Join(s.MilestonesDir(), fmt.Sprintf("%s.md", id))
	if err := s.writer.WriteFile(filePath, fileBytes, 0644); err != nil {
		return nil, nil, fmt.Errorf("failed to save milestone file: %w", err)
	}

	ms := model.Milestone{
		ID:         id,
		Title:      title,
		Status:     model.MilestoneStatusOpen,
		TargetDate: targetDate,
		Tags:       tags,
		Summary:    summary,
		Body:       in.Body,
		FilePath:   filePath,
		ModTime:    time.Now(),
	}

	if err := s.store.UpsertMilestone(ctx, ms); err != nil {
		return nil, nil, fmt.Errorf("failed to index milestone: %w", err)
	}

	return nil, &MilestoneDetail{
		ID:                 ms.ID,
		Title:              ms.Title,
		Status:             ms.Status,
		IsArchived:         false,
		TargetDate:         ms.TargetDate,
		TargetStartAt:      ms.TargetStartAt,
		TargetEndAt:        ms.TargetEndAt,
		TargetTimeframe:    ms.TargetTimeframe,
		Tags:               ms.Tags,
		Summary:            ms.Summary,
		Body:               ms.Body,
		TotalTasks:         0,
		CompletedTasks:     0,
		ProgressPercentage: 0,
		AssignedTasks:      []string{},
		FilePath:           filePath,
	}, nil
}

func (s *Server) toolUpdateMilestone(ctx context.Context, _ *mcp.CallToolRequest, in UpdateMilestoneInput) (*mcp.CallToolResult, *MilestoneDetail, error) {
	id := strings.TrimSpace(in.ID)
	if id == "" {
		return nil, nil, errors.New("milestone id is required")
	}

	ms, err := s.store.GetMilestone(ctx, id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, nil, fmt.Errorf("milestone %q not found", id)
		}
		return nil, nil, fmt.Errorf("failed to get milestone %q: %w", id, err)
	}

	if in.Status != nil {
		switch st := strings.ToLower(strings.TrimSpace(*in.Status)); st {
		case "open":
			ms.Status = model.MilestoneStatusOpen
		case "closed":
			ms.Status = model.MilestoneStatusClosed
		default:
			return nil, nil, fmt.Errorf("invalid status %q; expected 'open' or 'closed'", *in.Status)
		}
	}

	if in.TargetDate != nil {
		ms.TargetDate = strings.TrimSpace(*in.TargetDate)
	}

	if in.Summary != nil {
		s := strings.TrimSpace(*in.Summary)
		if s != "" {
			ms.Summary = s
		}
	}

	if in.Tags != nil {
		tags := *in.Tags
		if s.cfg.Tags.EnforceAllowed && len(tags) > 0 {
			for _, tag := range tags {
				if !slices.Contains(s.cfg.Tags.Allowed, tag) {
					return nil, nil, fmt.Errorf("tag %q is not permitted. Allowed tags: %v", tag, s.cfg.Tags.Allowed)
				}
			}
		}
		ms.Tags = tags
	}

	if in.Body != nil {
		ms.Body = *in.Body
	}

	fm := model.MilestoneFrontmatter{
		Title:      ms.Title,
		Status:     ms.Status,
		TargetDate: ms.TargetDate,
		Tags:       ms.Tags,
		Summary:    ms.Summary,
	}

	fileBytes, err := parser.Format(fm, ms.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to format milestone markdown: %w", err)
	}

	filePath := ms.FilePath
	if filePath == "" {
		filePath = filepath.Join(s.MilestonesDir(), fmt.Sprintf("%s.md", id))
	}

	if err := s.writer.WriteFile(filePath, fileBytes, 0644); err != nil {
		return nil, nil, fmt.Errorf("failed to save milestone file: %w", err)
	}

	ms.ModTime = time.Now()
	if err := s.store.UpsertMilestone(ctx, ms); err != nil {
		return nil, nil, fmt.Errorf("failed to index updated milestone: %w", err)
	}

	if updatedMs, err := s.store.GetMilestone(ctx, id); err == nil {
		ms = updatedMs
	}

	assignedTasks, _ := s.store.ListTasks(ctx, model.FilterCriteria{Milestone: id})
	taskSlugs := make([]string, 0, len(assignedTasks))
	for _, t := range assignedTasks {
		taskSlugs = append(taskSlugs, t.ID)
	}

	isArchived := ms.Status == model.MilestoneStatusClosed || (ms.TotalTasks > 0 && ms.CompletedTasks == ms.TotalTasks)

	return nil, &MilestoneDetail{
		ID:                 ms.ID,
		Title:              ms.Title,
		Status:             ms.Status,
		IsArchived:         isArchived,
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
		FilePath:           filePath,
	}, nil
}

func (s *Server) toolDeleteMilestone(ctx context.Context, _ *mcp.CallToolRequest, in DeleteMilestoneInput) (*mcp.CallToolResult, *DeleteEntityOutput, error) {
	if !s.cfg.MCP.AllowMutations {
		return nil, nil, errors.New("mutations are disabled in configuration")
	}

	id := strings.TrimSpace(in.ID)
	if id == "" {
		return nil, nil, errors.New("milestone id is required")
	}

	ms, err := s.store.GetMilestone(ctx, id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, nil, fmt.Errorf("milestone %q not found", id)
		}
		return nil, nil, fmt.Errorf("failed to get milestone %q: %w", id, err)
	}

	if !in.Force && ms.TotalTasks > 0 {
		return nil, nil, fmt.Errorf("cannot delete milestone %q: %d task(s) are assigned to it. Set force=true to delete anyway", id, ms.TotalTasks)
	}

	if ms.FilePath != "" {
		if err := s.writer.RemoveFile(ms.FilePath); err != nil {
			return nil, nil, fmt.Errorf("failed to remove milestone file: %w", err)
		}
	}

	if err := s.store.DeleteMilestone(ctx, id); err != nil {
		return nil, nil, fmt.Errorf("failed to delete milestone from store: %w", err)
	}

	return nil, &DeleteEntityOutput{
		Success: true,
		ID:      id,
		Message: fmt.Sprintf("Milestone %q deleted successfully", id),
	}, nil
}

