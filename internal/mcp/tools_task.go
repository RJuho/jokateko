package mcp

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/RJuho/jokateko/internal/model"
	"github.com/RJuho/jokateko/internal/parser"
	"github.com/RJuho/jokateko/internal/service"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// --- Task Tool Input / Output Models ---

type ListTasksInput struct {
	Status    string `json:"status,omitempty" jsonschema:"Filter by workflow status (backlog, ready, in_progress, in_review, done)"`
	Milestone string `json:"milestone,omitempty" jsonschema:"Filter by milestone slug (e.g. 260915-mvp)"`
	Tag       string `json:"tag,omitempty" jsonschema:"Filter by tag name"`
	Priority  string `json:"priority,omitempty" jsonschema:"Filter by priority (low, medium, high, critical)"`
}

type TaskSummary struct {
	ID                string         `json:"id"`
	Title             string         `json:"title"`
	Status            string         `json:"status"`
	Priority          model.Priority `json:"priority"`
	Milestone         string         `json:"milestone,omitempty"`
	Tags              []string       `json:"tags"`
	Summary           string         `json:"summary"`
	Dependencies      []string       `json:"dependencies,omitempty"`
	TotalCriteria     int            `json:"total_criteria"`
	CompletedCriteria int            `json:"completed_criteria"`
	CreatedAt         string         `json:"created_at,omitempty"`
	ChangedAt         string         `json:"changed_at,omitempty"`
	TargetAt          string         `json:"target_at,omitempty"`
}

type GetTaskInput struct {
	ID string `json:"id" jsonschema:"required,Task ID or slug (e.g. 260901-setup-database)"`
}

type TaskDetail struct {
	ID                string         `json:"id"`
	Title             string         `json:"title"`
	Status            string         `json:"status"`
	Priority          model.Priority `json:"priority"`
	Milestone         string         `json:"milestone,omitempty"`
	Tags              []string       `json:"tags"`
	Summary           string         `json:"summary"`
	Dependencies      []string       `json:"dependencies"`
	Body              string         `json:"body"`
	TotalCriteria     int            `json:"total_criteria"`
	CompletedCriteria int            `json:"completed_criteria"`
	FilePath          string         `json:"file_path,omitempty"`
	CreatedAt         string         `json:"created_at,omitempty"`
	ChangedAt         string         `json:"changed_at,omitempty"`
	TargetAt          string         `json:"target_at,omitempty"`
}

type ListTaskItemsInput struct {
	ID string `json:"id" jsonschema:"required,Task ID or slug"`
}

type TaskChecklistItem struct {
	Index     int    `json:"index"`
	Text      string `json:"text"`
	Completed bool   `json:"completed"`
}

type ListTaskItemsOutput struct {
	ID        string              `json:"id"`
	Total     int                 `json:"total"`
	Completed int                 `json:"completed"`
	Items     []TaskChecklistItem `json:"items"`
}

type UpdateTaskItemInput struct {
	ID        string `json:"id" jsonschema:"required,Task ID or slug"`
	Index     int    `json:"index" jsonschema:"required,1-based index of the checklist item returned by list_task_items"`
	Completed bool   `json:"completed" jsonschema:"required,true to mark completed (- [x]), false for open (- [ ])"`
}

type UpdateTaskItemOutput struct {
	ID             string `json:"id"`
	Index          int    `json:"index"`
	Text           string `json:"text"`
	Completed      bool   `json:"completed"`
	TotalItems     int    `json:"total_items"`
	CompletedItems int    `json:"completed_items"`
}

type CreateTaskInput struct {
	Title           string   `json:"title" jsonschema:"required,Short, descriptive title"`
	Status          string   `json:"status,omitempty" jsonschema:"Column status (defaults to first column, e.g. backlog)"`
	Priority        string   `json:"priority,omitempty" jsonschema:"low, medium, high, or critical (default: medium)"`
	Milestone       string   `json:"milestone,omitempty" jsonschema:"Associated milestone slug"`
	ReopenMilestone bool     `json:"reopen_milestone,omitempty" jsonschema:"Set true if milestone is already completed/closed"`
	Tags            []string `json:"tags,omitempty" jsonschema:"Categorization tags"`
	Summary         string   `json:"summary" jsonschema:"required,1-2 sentence high-level summary"`
	Dependencies    []string `json:"dependencies,omitempty" jsonschema:"Slugs of blocking tasks"`
	TargetAt        string   `json:"target_at,omitempty" jsonschema:"Target delivery/due date in RFC3339 UTC or YYYY-MM-DD format"`
	Body            string   `json:"body" jsonschema:"required,Markdown body with acceptance criteria"`
}

type UpdateTaskStatusInput struct {
	ID     string `json:"id" jsonschema:"required,Task ID or slug"`
	Status string `json:"status" jsonschema:"required,Target column status (e.g. ready, in_progress, in_review)"`
}

type UpdateTaskStatusOutput struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

type CompleteTaskInput struct {
	ID                 string `json:"id" jsonschema:"required,Task ID or slug"`
	Summary            string `json:"summary" jsonschema:"required,Updated 1-2 sentence high-level summary reflecting the completed state"`
	WhatDone           string `json:"what_done" jsonschema:"required,Detailed technical description of what was implemented"`
	WhyDone            string `json:"why_done" jsonschema:"required,Architectural reasoning and rationale for decisions made"`
	IgnoreDependencies bool   `json:"ignore_dependencies,omitempty" jsonschema:"Force completion even if blocking dependencies are not yet done"`
}

type UnblockedTaskInfo struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Status string `json:"status"`
}

type CompleteTaskOutput struct {
	Success        bool                `json:"success"`
	ID             string              `json:"id"`
	Status         string              `json:"status"`
	Summary        string              `json:"summary"`
	UnblockedTasks []UnblockedTaskInfo `json:"unblocked_tasks"`
	Message        string              `json:"message"`
}

type UpdateTaskContentInput struct {
	ID              string    `json:"id" jsonschema:"required,Task ID or slug"`
	Title           *string   `json:"title,omitempty" jsonschema:"New title"`
	Summary         *string   `json:"summary,omitempty" jsonschema:"Updated summary"`
	Priority        *string   `json:"priority,omitempty" jsonschema:"low, medium, high, critical"`
	Milestone       *string   `json:"milestone,omitempty" jsonschema:"Associated milestone slug"`
	ReopenMilestone bool      `json:"reopen_milestone,omitempty" jsonschema:"Set true if target milestone is already completed"`
	Tags            *[]string `json:"tags,omitempty" jsonschema:"New tag list"`
	Dependencies    *[]string `json:"dependencies,omitempty" jsonschema:"New dependency slugs"`
	TargetAt        *string   `json:"target_at,omitempty" jsonschema:"Updated target delivery/due date in RFC3339 UTC or YYYY-MM-DD format"`
	Body            *string   `json:"body,omitempty" jsonschema:"New markdown body"`
}

type DeleteTaskInput struct {
	ID    string `json:"id" jsonschema:"required,Task ID or slug (e.g. 260901-setup-database)"`
	Force bool   `json:"force,omitempty" jsonschema:"Force deletion even if other tasks depend on this task"`
}

type AddTaskDependencyInput struct {
	ID           string `json:"id" jsonschema:"required,Task ID or slug that will depend on dependency_id"`
	DependencyID string `json:"dependency_id" jsonschema:"required,Task ID or slug of the prerequisite task"`
}

type RemoveTaskDependencyInput struct {
	ID           string `json:"id" jsonschema:"required,Task ID or slug"`
	DependencyID string `json:"dependency_id" jsonschema:"required,Task ID or slug of the dependency to remove"`
}

type TaskDependencyOutput struct {
	Success      bool     `json:"success"`
	ID           string   `json:"id"`
	DependencyID string   `json:"dependency_id"`
	Dependencies []string `json:"dependencies"`
	Message      string   `json:"message"`
}

type AddTaskNoteInput struct {
	ID   string `json:"id" jsonschema:"required,Task ID or slug"`
	Note string `json:"note" jsonschema:"required,Markdown note content to append under ## Notes"`
}

type AddTaskNoteOutput struct {
	Success           bool   `json:"success"`
	ID                string `json:"id"`
	Note              string `json:"note"`
	TotalCriteria     int    `json:"total_criteria"`
	CompletedCriteria int    `json:"completed_criteria"`
	Message           string `json:"message"`
}

type SetTaskTargetInput struct {
	ID       string `json:"id" jsonschema:"required,Task ID or slug (e.g. 260901-setup-database)"`
	TargetAt string `json:"target_at" jsonschema:"Target delivery/due date in RFC3339 UTC, YYYY-MM-DD, or YYYY-MM-DDTHH:MM format (pass empty string to clear)"`
}

type SetTaskTargetOutput struct {
	Success  bool   `json:"success"`
	ID       string `json:"id"`
	TargetAt string `json:"target_at,omitempty"`
	Message  string `json:"message"`
}

// registerTaskTools registers all task tools with the MCP server.
func (s *Server) registerTaskTools() {
	// 1. list_tasks
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "list_tasks",
		Description: "Lists tasks with optional filtering. Returns compact summaries.",
	}, s.toolListTasks)

	// 2. get_task
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "get_task",
		Description: "Fetches complete task specification including full Markdown body and acceptance criteria.",
	}, s.toolGetTask)

	// 3. list_task_items
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "list_task_items",
		Description: "Lists all acceptance criteria and checklist items from a task with 1-based index and completion status.",
	}, s.toolListTaskItems)

	// 4. update_task_item
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "update_task_item",
		Description: "Updates the completion status of a specific checklist item in a task by its 1-based index.",
	}, s.toolUpdateTaskItem)

	// 5. create_task
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "create_task",
		Description: "Creates a new task file inside .jokateko/tasks/.",
	}, s.toolCreateTask)

	// 6. update_task_status
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "update_task_status",
		Description: "Transitions a task to a different workflow column (e.g. ready, in_progress, in_review). Rejects 'done' (use complete_task).",
	}, s.toolUpdateTaskStatus)

	// 7. complete_task
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "complete_task",
		Description: "Marks a task as completed while strictly enforcing documentation and dependency integrity.",
	}, s.toolCompleteTask)

	// 8. update_task_content
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "update_task_content",
		Description: "Updates metadata fields and/or the Markdown body of an existing task.",
	}, s.toolUpdateTaskContent)

	// 9. delete_task
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "delete_task",
		Description: "Deletes a task markdown file and removes it from the store. Rejects if other tasks depend on it unless force=true.",
	}, s.toolDeleteTask)

	// 10. add_task_dependency
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "add_task_dependency",
		Description: "Adds an upstream dependency to a task after validating existence and verifying the graph has no circular dependencies.",
	}, s.toolAddTaskDependency)

	// 11. remove_task_dependency
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "remove_task_dependency",
		Description: "Removes an upstream dependency from a task.",
	}, s.toolRemoveTaskDependency)

	// 12. add_task_note
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "add_task_note",
		Description: "Appends a timestamped note entry under the ## Notes section of a task without modifying existing specification or criteria.",
	}, s.toolAddTaskNote)

	// 13. set_task_target
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "set_task_target",
		Description: "Sets, updates, or clears the target delivery date (target_at) of a task.",
	}, s.toolSetTaskTarget)
}

func (s *Server) toolListTasks(ctx context.Context, _ *mcp.CallToolRequest, in ListTasksInput) (*mcp.CallToolResult, []TaskSummary, error) {
	filter := model.FilterCriteria{
		Status:    in.Status,
		Milestone: in.Milestone,
		Tag:       in.Tag,
		Priority:  model.Priority(in.Priority),
	}

	tasks, err := s.store.ListTasks(ctx, filter)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list tasks: %w", err)
	}

	cols := s.cfg.Columns()

	if in.Status != "" {
		idx := slices.IndexFunc(cols, func(c model.Column) bool { return c.ID == in.Status })
		var col model.Column
		if idx >= 0 {
			col = cols[idx]
		} else {
			col = model.Column{ID: in.Status}
		}
		model.SortTasksForColumn(tasks, col)
	} else {
		tasks = model.SortTasksByColumnOrder(tasks, cols)
	}

	summaries := make([]TaskSummary, 0, len(tasks))
	for _, t := range tasks {
		summaries = append(summaries, TaskSummary{
			ID:                t.ID,
			Title:             t.Title,
			Status:            t.Status,
			Priority:          t.Priority,
			Milestone:         t.Milestone,
			Tags:              t.Tags,
			Summary:           t.Summary,
			Dependencies:      t.Dependencies,
			TotalCriteria:     t.TotalCriteria,
			CompletedCriteria: t.CompletedCriteria,
			CreatedAt:         t.CreatedAt,
			ChangedAt:         t.ChangedAt,
			TargetAt:          t.TargetAt,
		})
	}

	return nil, summaries, nil
}

func (s *Server) toolGetTask(ctx context.Context, _ *mcp.CallToolRequest, in GetTaskInput) (*mcp.CallToolResult, *TaskDetail, error) {
	id, err := requireID(in.ID, "task")
	if err != nil {
		return nil, nil, err
	}
	task, err := s.svc.GetTask(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	return nil, toTaskDetail(task), nil
}

func (s *Server) toolListTaskItems(ctx context.Context, _ *mcp.CallToolRequest, in ListTaskItemsInput) (*mcp.CallToolResult, *ListTaskItemsOutput, error) {
	id, err := requireID(in.ID, "task")
	if err != nil {
		return nil, nil, err
	}
	task, err := s.svc.GetTask(ctx, id)
	if err != nil {
		return nil, nil, err
	}

	total, completed, items := parser.ExtractAcceptanceCriteria([]byte(task.Body))
	checklistItems := make([]TaskChecklistItem, len(items))
	for i, it := range items {
		checklistItems[i] = TaskChecklistItem{
			Index:     i + 1, // 1-based index
			Text:      it.Text,
			Completed: it.Completed,
		}
	}

	return nil, &ListTaskItemsOutput{
		ID:        task.ID,
		Total:     total,
		Completed: completed,
		Items:     checklistItems,
	}, nil
}

func toTaskDetail(t model.Task) *TaskDetail {
	return &TaskDetail{
		ID:                t.ID,
		Title:             t.Title,
		Status:            t.Status,
		Priority:          t.Priority,
		Milestone:         t.Milestone,
		Tags:              t.Tags,
		Summary:           t.Summary,
		Dependencies:      t.Dependencies,
		Body:              t.Body,
		TotalCriteria:     t.TotalCriteria,
		CompletedCriteria: t.CompletedCriteria,
		FilePath:          t.FilePath,
		CreatedAt:         t.CreatedAt,
		ChangedAt:         t.ChangedAt,
		TargetAt:          t.TargetAt,
	}
}

func (s *Server) toolUpdateTaskItem(ctx context.Context, _ *mcp.CallToolRequest, in UpdateTaskItemInput) (*mcp.CallToolResult, *UpdateTaskItemOutput, error) {
	if err := s.checkMutations(); err != nil {
		return nil, nil, err
	}
	id, err := requireID(in.ID, "task")
	if err != nil {
		return nil, nil, err
	}

	task, err := s.svc.UpdateTaskItem(ctx, id, in.Index, in.Completed)
	if err != nil {
		return nil, nil, err
	}

	_, _, items := parser.ExtractAcceptanceCriteria([]byte(task.Body))
	text := ""
	if in.Index >= 1 && in.Index <= len(items) {
		text = items[in.Index-1].Text
	}

	return nil, &UpdateTaskItemOutput{
		ID:             id,
		Index:          in.Index,
		Text:           text,
		Completed:      in.Completed,
		TotalItems:     task.TotalCriteria,
		CompletedItems: task.CompletedCriteria,
	}, nil
}

func (s *Server) toolCreateTask(ctx context.Context, _ *mcp.CallToolRequest, in CreateTaskInput) (*mcp.CallToolResult, *TaskDetail, error) {
	if err := s.checkMutations(); err != nil {
		return nil, nil, err
	}
	if err := s.svc.CheckTags(in.Tags); err != nil {
		return nil, nil, err
	}
	milestone := strings.TrimSpace(in.Milestone)
	if err := s.checkMilestoneOpen(ctx, milestone, in.ReopenMilestone); err != nil {
		return nil, nil, err
	}

	task, err := s.svc.CreateTask(ctx, service.NewTask{
		Title:        in.Title,
		Status:       in.Status,
		Priority:     model.Priority(in.Priority),
		Milestone:    milestone,
		Tags:         in.Tags,
		Summary:      in.Summary,
		Dependencies: in.Dependencies,
		TargetAt:     in.TargetAt,
		Body:         in.Body,
	})
	if err != nil {
		return nil, nil, err
	}
	return nil, toTaskDetail(task), nil
}

func (s *Server) toolUpdateTaskStatus(ctx context.Context, _ *mcp.CallToolRequest, in UpdateTaskStatusInput) (*mcp.CallToolResult, *UpdateTaskStatusOutput, error) {
	if err := s.checkMutations(); err != nil {
		return nil, nil, err
	}
	id, err := requireID(in.ID, "task")
	if err != nil {
		return nil, nil, err
	}

	status := strings.TrimSpace(in.Status)
	// Strict 'Done' Guard
	if strings.EqualFold(status, "done") {
		return nil, nil, errors.New("cannot set status to 'done' directly via update_task_status. You must invoke the complete_task tool to document what was done, why it was done, and provide an updated summary")
	}
	if !s.cfg.HasColumn(status) {
		return nil, nil, fmt.Errorf("status %q is not a valid board column", status)
	}

	if _, err := s.svc.UpdateTaskStatus(ctx, id, status); err != nil {
		return nil, nil, err
	}
	return nil, &UpdateTaskStatusOutput{ID: id, Status: status}, nil
}

func (s *Server) toolCompleteTask(ctx context.Context, _ *mcp.CallToolRequest, in CompleteTaskInput) (*mcp.CallToolResult, *CompleteTaskOutput, error) {
	if err := s.checkMutations(); err != nil {
		return nil, nil, err
	}
	id, err := requireID(in.ID, "task")
	if err != nil {
		return nil, nil, err
	}

	task, unblockedTasks, err := s.svc.CompleteTask(ctx, service.CompleteTaskInput{
		ID:                 id,
		Summary:            in.Summary,
		WhatDone:           in.WhatDone,
		WhyDone:            in.WhyDone,
		IgnoreDependencies: in.IgnoreDependencies,
	})
	if err != nil {
		return nil, nil, err
	}

	unblocked := make([]UnblockedTaskInfo, 0, len(unblockedTasks))
	for _, dt := range unblockedTasks {
		unblocked = append(unblocked, UnblockedTaskInfo{ID: dt.ID, Title: dt.Title, Status: dt.Status})
	}

	msg := fmt.Sprintf("Task %s marked as done.", id)
	if len(unblocked) > 0 {
		msg = fmt.Sprintf("Task %s marked as done. %d downstream task(s) unblocked.", id, len(unblocked))
	}

	return nil, &CompleteTaskOutput{
		Success:        true,
		ID:             id,
		Status:         task.Status,
		Summary:        task.Summary,
		UnblockedTasks: unblocked,
		Message:        msg,
	}, nil
}

func (s *Server) toolUpdateTaskContent(ctx context.Context, _ *mcp.CallToolRequest, in UpdateTaskContentInput) (*mcp.CallToolResult, *TaskDetail, error) {
	if err := s.checkMutations(); err != nil {
		return nil, nil, err
	}
	id, err := requireID(in.ID, "task")
	if err != nil {
		return nil, nil, err
	}
	if in.Tags != nil {
		if err := s.svc.CheckTags(*in.Tags); err != nil {
			return nil, nil, err
		}
	}
	if in.Milestone != nil {
		if err := s.checkMilestoneOpen(ctx, strings.TrimSpace(*in.Milestone), in.ReopenMilestone); err != nil {
			return nil, nil, err
		}
	}

	task, err := s.svc.UpdateTask(ctx, id, func(t *model.Task) error {
		if in.Title != nil {
			if v := strings.TrimSpace(*in.Title); v != "" {
				t.Title = v
			}
		}
		if in.Summary != nil {
			if v := strings.TrimSpace(*in.Summary); v != "" {
				t.Summary = v
			}
		}
		if in.Priority != nil {
			if p := model.Priority(*in.Priority); p.IsValid() {
				t.Priority = p
			}
		}
		if in.Milestone != nil {
			t.Milestone = strings.TrimSpace(*in.Milestone)
		}
		if in.Tags != nil {
			t.Tags = *in.Tags
		}
		if in.Dependencies != nil {
			t.Dependencies = *in.Dependencies
		}
		if in.TargetAt != nil {
			targetAt, err := service.NormalizeTargetAt(*in.TargetAt)
			if err != nil {
				return err
			}
			t.TargetAt = targetAt
		}
		if in.Body != nil {
			if err := s.svc.CheckBodyEdit(t.Status, t.Body, *in.Body, "use add_task_note to append notes"); err != nil {
				return err
			}
			t.Body = *in.Body
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return nil, toTaskDetail(task), nil
}

func (s *Server) toolDeleteTask(ctx context.Context, _ *mcp.CallToolRequest, in DeleteTaskInput) (*mcp.CallToolResult, *DeleteEntityOutput, error) {
	if err := s.checkMutations(); err != nil {
		return nil, nil, err
	}
	id, err := requireID(in.ID, "task")
	if err != nil {
		return nil, nil, err
	}
	if err := s.svc.DeleteTask(ctx, id, in.Force); err != nil {
		return nil, nil, err
	}
	return nil, &DeleteEntityOutput{
		Success: true,
		ID:      id,
		Message: fmt.Sprintf("Task %q deleted successfully", id),
	}, nil
}

func (s *Server) toolAddTaskDependency(ctx context.Context, _ *mcp.CallToolRequest, in AddTaskDependencyInput) (*mcp.CallToolResult, *TaskDependencyOutput, error) {
	if err := s.checkMutations(); err != nil {
		return nil, nil, err
	}
	id, err := requireID(in.ID, "task")
	if err != nil {
		return nil, nil, err
	}
	depID := strings.TrimSpace(in.DependencyID)
	if depID == "" {
		return nil, nil, errors.New("dependency_id is required")
	}

	task, added, err := s.svc.AddTaskDependency(ctx, id, depID)
	if err != nil {
		return nil, nil, err
	}

	msg := fmt.Sprintf("Dependency %q added to task %q", depID, id)
	if !added {
		msg = fmt.Sprintf("Dependency %q already exists on task %q", depID, id)
	}
	return nil, &TaskDependencyOutput{
		Success:      true,
		ID:           id,
		DependencyID: depID,
		Dependencies: task.Dependencies,
		Message:      msg,
	}, nil
}

func (s *Server) toolRemoveTaskDependency(ctx context.Context, _ *mcp.CallToolRequest, in RemoveTaskDependencyInput) (*mcp.CallToolResult, *TaskDependencyOutput, error) {
	if err := s.checkMutations(); err != nil {
		return nil, nil, err
	}
	id, err := requireID(in.ID, "task")
	if err != nil {
		return nil, nil, err
	}
	depID := strings.TrimSpace(in.DependencyID)
	if depID == "" {
		return nil, nil, errors.New("dependency_id is required")
	}

	task, err := s.svc.RemoveTaskDependency(ctx, id, depID)
	if err != nil {
		return nil, nil, err
	}
	return nil, &TaskDependencyOutput{
		Success:      true,
		ID:           id,
		DependencyID: depID,
		Dependencies: task.Dependencies,
		Message:      fmt.Sprintf("Dependency %q removed from task %q", depID, id),
	}, nil
}

func (s *Server) toolAddTaskNote(ctx context.Context, _ *mcp.CallToolRequest, in AddTaskNoteInput) (*mcp.CallToolResult, *AddTaskNoteOutput, error) {
	if err := s.checkMutations(); err != nil {
		return nil, nil, err
	}
	id, err := requireID(in.ID, "task")
	if err != nil {
		return nil, nil, err
	}

	task, err := s.svc.AddTaskNote(ctx, id, in.Note)
	if err != nil {
		return nil, nil, err
	}
	return nil, &AddTaskNoteOutput{
		Success:           true,
		ID:                id,
		Note:              strings.TrimSpace(in.Note),
		TotalCriteria:     task.TotalCriteria,
		CompletedCriteria: task.CompletedCriteria,
		Message:           fmt.Sprintf("Note appended to task %q under ## Notes", id),
	}, nil
}

func (s *Server) toolSetTaskTarget(ctx context.Context, _ *mcp.CallToolRequest, in SetTaskTargetInput) (*mcp.CallToolResult, *SetTaskTargetOutput, error) {
	if err := s.checkMutations(); err != nil {
		return nil, nil, err
	}
	id, err := requireID(in.ID, "task")
	if err != nil {
		return nil, nil, err
	}

	task, err := s.svc.SetTaskTarget(ctx, id, in.TargetAt)
	if err != nil {
		return nil, nil, err
	}

	msg := fmt.Sprintf("Target date for task %q updated to %s", id, task.TargetAt)
	if task.TargetAt == "" {
		msg = fmt.Sprintf("Target date for task %q cleared", id)
	}
	return nil, &SetTaskTargetOutput{
		Success:  true,
		ID:       id,
		TargetAt: task.TargetAt,
		Message:  msg,
	}, nil
}
