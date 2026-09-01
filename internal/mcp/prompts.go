package mcp

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/RJuho/jokateko/internal/model"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func (s *Server) registerPrompts() {
	// 1. next_task
	s.mcpServer.AddPrompt(&mcp.Prompt{
		Name:        "next_task",
		Description: "Recommends the next unblocked ready task with highest priority and relevant Tier-1 architectural rules.",
	}, s.promptNextTask)
}

func priorityScore(p model.Priority) int {
	switch p {
	case model.PriorityCritical:
		return 4
	case model.PriorityHigh:
		return 3
	case model.PriorityMedium:
		return 2
	case model.PriorityLow:
		return 1
	default:
		return 0
	}
}

func (s *Server) promptNextTask(ctx context.Context, _ *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	readyTasks, err := s.store.ListTasks(ctx, model.FilterCriteria{Status: "ready"})
	if err != nil {
		return nil, fmt.Errorf("failed to list ready tasks: %w", err)
	}

	// Filter tasks whose dependencies are all satisfied
	var unblockedTasks []model.Task
	for _, t := range readyTasks {
		done, err := s.store.AreDependenciesDone(ctx, t.ID)
		if err == nil && done {
			unblockedTasks = append(unblockedTasks, t)
		}
	}

	// Sort by priority descending (critical > high > medium > low)
	slices.SortFunc(unblockedTasks, func(a, b model.Task) int {
		return priorityScore(b.Priority) - priorityScore(a.Priority)
	})

	// Fetch Tier-1 core strategies
	strats, err := s.store.ListStrategies(ctx)
	var tier1Strats []model.Strategy
	if err == nil {
		for _, st := range strats {
			if st.Tier == model.TierCore {
				tier1Strats = append(tier1Strats, st)
			}
		}
	}

	var sb strings.Builder
	if len(unblockedTasks) == 0 {
		sb.WriteString("There are currently no unblocked tasks ready to work on in the 'ready' column.\n")
		sb.WriteString("Check the backlog or inspect active in-progress tasks to resolve blocking dependencies.\n")
	} else {
		topTask := unblockedTasks[0]
		fmt.Fprintf(&sb, "### Recommended Next Task: %s (`%s`)\n\n", topTask.Title, topTask.ID)
		fmt.Fprintf(&sb, "- **Priority:** %s\n", topTask.Priority)
		if topTask.Milestone != "" {
			fmt.Fprintf(&sb, "- **Milestone:** %s\n", topTask.Milestone)
		}
		if len(topTask.Tags) > 0 {
			fmt.Fprintf(&sb, "- **Tags:** %s\n", strings.Join(topTask.Tags, ", "))
		}
		fmt.Fprintf(&sb, "- **Summary:** %s\n\n", topTask.Summary)

		if topTask.Body != "" {
			sb.WriteString("#### Specification & Acceptance Criteria:\n\n")
			sb.WriteString(topTask.Body)
			sb.WriteString("\n\n")
		}

		if len(tier1Strats) > 0 {
			sb.WriteString("#### Core Architectural Guidelines (Tier-1):\n")
			for _, st := range tier1Strats {
				fmt.Fprintf(&sb, "- **%s**: %s\n", st.Title, st.Summary)
			}
			sb.WriteString("\n")
		}

		sb.WriteString("#### Instructions for Agent:\n")
		sb.WriteString("1. Inspect full acceptance criteria before modifying any code.\n")
		sb.WriteString("2. Use `update_task_status` to transition this task to `in_progress`.\n")
		sb.WriteString("3. Use `list_task_items` and `update_task_item` to toggle checklist items as you complete them.\n")
		sb.WriteString("4. When finished, call `complete_task` with `what_done` and `why_done` documentation.\n")
	}

	return &mcp.GetPromptResult{
		Description: "Next unblocked ready task recommendation",
		Messages: []*mcp.PromptMessage{
			{
				Role: "user",
				Content: &mcp.TextContent{
					Text: sb.String(),
				},
			},
		},
	}, nil
}
