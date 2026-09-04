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
	"github.com/RJuho/jokateko/internal/validator"
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
		})
	}

	return nil, summaries, nil
}

func (s *Server) toolGetTask(ctx context.Context, _ *mcp.CallToolRequest, in GetTaskInput) (*mcp.CallToolResult, *TaskDetail, error) {
	id := strings.TrimSpace(in.ID)
	if id == "" {
		return nil, nil, errors.New("task id is required")
	}

	task, err := s.store.GetTask(ctx, id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, nil, fmt.Errorf("task %q not found", id)
		}
		return nil, nil, fmt.Errorf("failed to get task %q: %w", id, err)
	}

	return nil, &TaskDetail{
		ID:                task.ID,
		Title:             task.Title,
		Status:            task.Status,
		Priority:          task.Priority,
		Milestone:         task.Milestone,
		Tags:              task.Tags,
		Summary:           task.Summary,
		Dependencies:      task.Dependencies,
		Body:              task.Body,
		TotalCriteria:     task.TotalCriteria,
		CompletedCriteria: task.CompletedCriteria,
		FilePath:          task.FilePath,
	}, nil
}

func (s *Server) toolListTaskItems(ctx context.Context, _ *mcp.CallToolRequest, in ListTaskItemsInput) (*mcp.CallToolResult, *ListTaskItemsOutput, error) {
	id := strings.TrimSpace(in.ID)
	if id == "" {
		return nil, nil, errors.New("task id is required")
	}

	task, err := s.store.GetTask(ctx, id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, nil, fmt.Errorf("task %q not found", id)
		}
		return nil, nil, fmt.Errorf("failed to get task %q: %w", id, err)
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

func (s *Server) toolUpdateTaskItem(ctx context.Context, _ *mcp.CallToolRequest, in UpdateTaskItemInput) (*mcp.CallToolResult, *UpdateTaskItemOutput, error) {
	id := strings.TrimSpace(in.ID)
	if id == "" {
		return nil, nil, errors.New("task id is required")
	}

	task, err := s.store.GetTask(ctx, id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, nil, fmt.Errorf("task %q not found", id)
		}
		return nil, nil, fmt.Errorf("failed to get task %q: %w", id, err)
	}

	newBody, err := parser.UpdateCheckboxByIndex(task.Body, in.Index, in.Completed)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to update checkbox item: %w", err)
	}

	fm := model.TaskFrontmatter{
		Title:        task.Title,
		Status:       task.Status,
		Priority:     task.Priority,
		Milestone:    task.Milestone,
		Tags:         task.Tags,
		Summary:      task.Summary,
		Dependencies: task.Dependencies,
	}

	fileBytes, err := parser.Format(fm, newBody)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to format task markdown: %w", err)
	}

	filePath := task.FilePath
	if filePath == "" {
		filePath = filepath.Join(s.TasksDir(), fmt.Sprintf("%s.md", id))
	}

	if err := s.writer.WriteFile(filePath, fileBytes, 0644); err != nil {
		return nil, nil, fmt.Errorf("failed to save task file: %w", err)
	}

	total, completed, items := parser.ExtractAcceptanceCriteria([]byte(newBody))
	task.Body = newBody
	task.TotalCriteria = total
	task.CompletedCriteria = completed
	task.ModTime = time.Now()

	if err := s.store.UpsertTask(ctx, task); err != nil {
		return nil, nil, fmt.Errorf("failed to index updated task: %w", err)
	}

	text := ""
	if in.Index >= 1 && in.Index <= len(items) {
		text = items[in.Index-1].Text
	}

	return nil, &UpdateTaskItemOutput{
		ID:             id,
		Index:          in.Index,
		Text:           text,
		Completed:      in.Completed,
		TotalItems:     total,
		CompletedItems: completed,
	}, nil
}

func (s *Server) toolCreateTask(ctx context.Context, _ *mcp.CallToolRequest, in CreateTaskInput) (*mcp.CallToolResult, *TaskDetail, error) {
	title := strings.TrimSpace(in.Title)
	if title == "" {
		return nil, nil, errors.New("title is required")
	}

	summary := strings.TrimSpace(in.Summary)
	if summary == "" {
		return nil, nil, errors.New("summary is required")
	}

	// 1. Tag validation guard
	if s.cfg.Tags.EnforceAllowed && len(in.Tags) > 0 {
		for _, tag := range in.Tags {
			if !slices.Contains(s.cfg.Tags.Allowed, tag) {
				return nil, nil, fmt.Errorf("tag %q is not permitted. Allowed tags: %v", tag, s.cfg.Tags.Allowed)
			}
		}
	}

	// 2. Closed Milestone Guard
	milestoneSlug := strings.TrimSpace(in.Milestone)
	if milestoneSlug != "" {
		ms, err := s.store.GetMilestone(ctx, milestoneSlug)
		if err == nil {
			isCompleted := ms.Status == model.MilestoneStatusClosed || (ms.TotalTasks > 0 && ms.CompletedTasks == ms.TotalTasks)
			if isCompleted && !in.ReopenMilestone {
				return nil, nil, fmt.Errorf("cannot attach task to completed milestone %q (100%% tasks done). Set reopen_milestone=true to explicitly attach tasks to this milestone", milestoneSlug)
			}
		}
	}

	status := in.Status
	if status == "" {
		if len(s.cfg.Board.Columns) > 0 {
			status = s.cfg.Board.Columns[0].ID
		} else {
			status = "backlog"
		}
	} else if !s.isValidColumn(status) {
		return nil, nil, fmt.Errorf("status %q is not a valid board column", status)
	}

	priority := model.Priority(in.Priority)
	if !priority.IsValid() {
		priority = model.PriorityMedium
	}

	tags := in.Tags
	if tags == nil {
		tags = []string{}
	}

	deps := in.Dependencies
	if deps == nil {
		deps = []string{}
	}

	id := fmt.Sprintf("%s-%s", time.Now().Format("060102"), slugify(title))
	fm := model.TaskFrontmatter{
		Title:        title,
		Status:       status,
		Priority:     priority,
		Milestone:    milestoneSlug,
		Tags:         tags,
		Summary:      summary,
		Dependencies: deps,
	}

	fileBytes, err := parser.Format(fm, in.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to format task markdown: %w", err)
	}

	filePath := filepath.Join(s.TasksDir(), fmt.Sprintf("%s.md", id))
	if err := s.writer.WriteFile(filePath, fileBytes, 0644); err != nil {
		return nil, nil, fmt.Errorf("failed to save task file: %w", err)
	}

	total, completed, _ := parser.ExtractAcceptanceCriteria([]byte(in.Body))
	task := model.Task{
		ID:                id,
		Title:             title,
		Status:            status,
		Priority:          priority,
		Milestone:         milestoneSlug,
		Tags:              tags,
		Summary:           summary,
		Dependencies:      deps,
		Body:              in.Body,
		TotalCriteria:     total,
		CompletedCriteria: completed,
		FilePath:          filePath,
		ModTime:           time.Now(),
	}

	if err := s.store.UpsertTask(ctx, task); err != nil {
		return nil, nil, fmt.Errorf("failed to index created task: %w", err)
	}

	return nil, &TaskDetail{
		ID:                task.ID,
		Title:             task.Title,
		Status:            task.Status,
		Priority:          task.Priority,
		Milestone:         task.Milestone,
		Tags:              task.Tags,
		Summary:           task.Summary,
		Dependencies:      task.Dependencies,
		Body:              task.Body,
		TotalCriteria:     task.TotalCriteria,
		CompletedCriteria: task.CompletedCriteria,
		FilePath:          task.FilePath,
	}, nil
}

func (s *Server) toolUpdateTaskStatus(ctx context.Context, _ *mcp.CallToolRequest, in UpdateTaskStatusInput) (*mcp.CallToolResult, *UpdateTaskStatusOutput, error) {
	id := strings.TrimSpace(in.ID)
	if id == "" {
		return nil, nil, errors.New("task id is required")
	}

	status := strings.TrimSpace(in.Status)
	// Strict 'Done' Guard
	if strings.EqualFold(status, "done") {
		return nil, nil, errors.New("cannot set status to 'done' directly via update_task_status. You must invoke the complete_task tool to document what was done, why it was done, and provide an updated summary")
	}

	if !s.isValidColumn(status) {
		return nil, nil, fmt.Errorf("status %q is not a valid board column", status)
	}

	task, err := s.store.GetTask(ctx, id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, nil, fmt.Errorf("task %q not found", id)
		}
		return nil, nil, fmt.Errorf("failed to get task %q: %w", id, err)
	}

	task.Status = status
	fm := model.TaskFrontmatter{
		Title:        task.Title,
		Status:       task.Status,
		Priority:     task.Priority,
		Milestone:    task.Milestone,
		Tags:         task.Tags,
		Summary:      task.Summary,
		Dependencies: task.Dependencies,
	}

	fileBytes, err := parser.Format(fm, task.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to format task markdown: %w", err)
	}

	filePath := task.FilePath
	if filePath == "" {
		filePath = filepath.Join(s.TasksDir(), fmt.Sprintf("%s.md", id))
	}

	if err := s.writer.WriteFile(filePath, fileBytes, 0644); err != nil {
		return nil, nil, fmt.Errorf("failed to save task file: %w", err)
	}

	task.ModTime = time.Now()
	if err := s.store.UpsertTask(ctx, task); err != nil {
		return nil, nil, fmt.Errorf("failed to update task status in store: %w", err)
	}

	return nil, &UpdateTaskStatusOutput{
		ID:     id,
		Status: status,
	}, nil
}

func (s *Server) toolCompleteTask(ctx context.Context, _ *mcp.CallToolRequest, in CompleteTaskInput) (*mcp.CallToolResult, *CompleteTaskOutput, error) {
	id := strings.TrimSpace(in.ID)
	if id == "" {
		return nil, nil, errors.New("task id is required")
	}

	summary := strings.TrimSpace(in.Summary)
	if summary == "" {
		return nil, nil, errors.New("summary is required when completing a task")
	}

	whatDone := strings.TrimSpace(in.WhatDone)
	if whatDone == "" {
		return nil, nil, errors.New("what_done documentation is required when completing a task")
	}

	whyDone := strings.TrimSpace(in.WhyDone)
	if whyDone == "" {
		return nil, nil, errors.New("why_done rationale is required when completing a task")
	}

	task, err := s.store.GetTask(ctx, id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, nil, fmt.Errorf("task %q not found", id)
		}
		return nil, nil, fmt.Errorf("failed to get task %q: %w", id, err)
	}

	// 1. Dependency Safety Verification
	if !in.IgnoreDependencies && len(task.Dependencies) > 0 {
		blockers, err := s.store.GetBlockingTasks(ctx, id)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to check task dependencies: %w", err)
		}
		if len(blockers) > 0 {
			var blockerSlugs []string
			for _, b := range blockers {
				blockerSlugs = append(blockerSlugs, b.ID)
			}
			return nil, nil, fmt.Errorf("cannot complete task: blocking dependencies remain unfinished: %v. Use ignore_dependencies=true to override", blockerSlugs)
		}
	}

	// 2. Acceptance Criteria Verification (Strict Open Checkbox Guard)
	newBody, err := parser.CompleteTaskBody(task.Body, time.Now(), whatDone, whyDone)
	if err != nil {
		return nil, nil, err
	}

	// 3. Update task
	task.Status = "done"
	task.Summary = summary
	task.Body = newBody

	total, completed, _ := parser.ExtractAcceptanceCriteria([]byte(newBody))
	task.TotalCriteria = total
	task.CompletedCriteria = completed
	task.ModTime = time.Now()

	fm := model.TaskFrontmatter{
		Title:        task.Title,
		Status:       task.Status,
		Priority:     task.Priority,
		Milestone:    task.Milestone,
		Tags:         task.Tags,
		Summary:      task.Summary,
		Dependencies: task.Dependencies,
	}

	fileBytes, err := parser.Format(fm, newBody)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to format task markdown: %w", err)
	}

	filePath := task.FilePath
	if filePath == "" {
		filePath = filepath.Join(s.TasksDir(), fmt.Sprintf("%s.md", id))
	}

	if err := s.writer.WriteFile(filePath, fileBytes, 0644); err != nil {
		return nil, nil, fmt.Errorf("failed to save completed task file: %w", err)
	}

	if err := s.store.UpsertTask(ctx, task); err != nil {
		return nil, nil, fmt.Errorf("failed to index completed task: %w", err)
	}

	// 4. Check downstream unblocked tasks
	unblockedTasks, err := s.store.FindUnblockedTasks(ctx, id)
	var unblocked []UnblockedTaskInfo
	if err == nil {
		for _, dt := range unblockedTasks {
			unblocked = append(unblocked, UnblockedTaskInfo{
				ID:     dt.ID,
				Title:  dt.Title,
				Status: dt.Status,
			})
		}
	}

	msg := fmt.Sprintf("Task %s marked as done.", id)
	if len(unblocked) > 0 {
		msg = fmt.Sprintf("Task %s marked as done. %d downstream task(s) unblocked.", id, len(unblocked))
	}

	return nil, &CompleteTaskOutput{
		Success:        true,
		ID:             id,
		Status:         "done",
		Summary:        summary,
		UnblockedTasks: unblocked,
		Message:        msg,
	}, nil
}

func (s *Server) toolUpdateTaskContent(ctx context.Context, _ *mcp.CallToolRequest, in UpdateTaskContentInput) (*mcp.CallToolResult, *TaskDetail, error) {
	id := strings.TrimSpace(in.ID)
	if id == "" {
		return nil, nil, errors.New("task id is required")
	}

	task, err := s.store.GetTask(ctx, id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, nil, fmt.Errorf("task %q not found", id)
		}
		return nil, nil, fmt.Errorf("failed to get task %q: %w", id, err)
	}

	if in.Title != nil {
		t := strings.TrimSpace(*in.Title)
		if t != "" {
			task.Title = t
		}
	}
	if in.Summary != nil {
		s := strings.TrimSpace(*in.Summary)
		if s != "" {
			task.Summary = s
		}
	}
	if in.Priority != nil {
		p := model.Priority(*in.Priority)
		if p.IsValid() {
			task.Priority = p
		}
	}
	if in.Milestone != nil {
		msSlug := strings.TrimSpace(*in.Milestone)
		if msSlug != "" {
			ms, err := s.store.GetMilestone(ctx, msSlug)
			if err == nil {
				isCompleted := ms.Status == model.MilestoneStatusClosed || (ms.TotalTasks > 0 && ms.CompletedTasks == ms.TotalTasks)
				if isCompleted && !in.ReopenMilestone {
					return nil, nil, fmt.Errorf("cannot attach task to completed milestone %q. Set reopen_milestone=true to proceed", msSlug)
				}
			}
		}
		task.Milestone = msSlug
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
		task.Tags = tags
	}
	if in.Dependencies != nil {
		task.Dependencies = *in.Dependencies
	}
	if in.Body != nil && *in.Body != task.Body {
		if !s.cfg.IsTaskEditable(task.Status) && !parser.IsOnlyCheckboxToggle(task.Body, *in.Body) {
			editable := s.cfg.Board.EditableStates
			if len(editable) == 0 {
				editable = []string{"backlog"}
			}
			return nil, nil, fmt.Errorf("cannot edit task body while task is in %q status; task body is only editable in [%s]; move task to an editable status to revise specification, or use add_task_note to append notes", task.Status, strings.Join(editable, ", "))
		}
		task.Body = *in.Body
	}

	total, completed, _ := parser.ExtractAcceptanceCriteria([]byte(task.Body))
	task.TotalCriteria = total
	task.CompletedCriteria = completed
	task.ModTime = time.Now()

	fm := model.TaskFrontmatter{
		Title:        task.Title,
		Status:       task.Status,
		Priority:     task.Priority,
		Milestone:    task.Milestone,
		Tags:         task.Tags,
		Summary:      task.Summary,
		Dependencies: task.Dependencies,
	}

	fileBytes, err := parser.Format(fm, task.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to format task markdown: %w", err)
	}

	filePath := task.FilePath
	if filePath == "" {
		filePath = filepath.Join(s.TasksDir(), fmt.Sprintf("%s.md", id))
	}

	if err := s.writer.WriteFile(filePath, fileBytes, 0644); err != nil {
		return nil, nil, fmt.Errorf("failed to save task file: %w", err)
	}

	if err := s.store.UpsertTask(ctx, task); err != nil {
		return nil, nil, fmt.Errorf("failed to index updated task: %w", err)
	}

	return nil, &TaskDetail{
		ID:                task.ID,
		Title:             task.Title,
		Status:            task.Status,
		Priority:          task.Priority,
		Milestone:         task.Milestone,
		Tags:              task.Tags,
		Summary:           task.Summary,
		Dependencies:      task.Dependencies,
		Body:              task.Body,
		TotalCriteria:     task.TotalCriteria,
		CompletedCriteria: task.CompletedCriteria,
		FilePath:          task.FilePath,
	}, nil
}

func (s *Server) isValidColumn(colID string) bool {
	for _, c := range s.cfg.Board.Columns {
		if c.ID == colID {
			return true
		}
	}
	return false
}

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var sb strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			sb.WriteRune(r)
		} else if r == ' ' || r == '-' || r == '_' {
			if sb.Len() > 0 && sb.String()[sb.Len()-1] != '-' {
				sb.WriteByte('-')
			}
		}
	}
	res := strings.Trim(sb.String(), "-")
	if res == "" {
		return "task"
	}
	if len(res) > 40 {
		return res[:40]
	}
	return res
}

func (s *Server) toolDeleteTask(ctx context.Context, _ *mcp.CallToolRequest, in DeleteTaskInput) (*mcp.CallToolResult, *DeleteEntityOutput, error) {
	if !s.cfg.MCP.AllowMutations {
		return nil, nil, errors.New("mutations are disabled in configuration")
	}

	id := strings.TrimSpace(in.ID)
	if id == "" {
		return nil, nil, errors.New("task id is required")
	}

	task, err := s.store.GetTask(ctx, id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, nil, fmt.Errorf("task %q not found", id)
		}
		return nil, nil, fmt.Errorf("failed to get task %q: %w", id, err)
	}

	if !in.Force {
		downstream, err := s.store.GetDownstreamTasks(ctx, id)
		if err == nil && len(downstream) > 0 {
			downstreamIDs := make([]string, len(downstream))
			for i, d := range downstream {
				downstreamIDs[i] = d.ID
			}
			return nil, nil, fmt.Errorf("cannot delete task %q: %d task(s) depend on it (%s). Set force=true to delete anyway", id, len(downstream), strings.Join(downstreamIDs, ", "))
		}
	}

	if task.FilePath != "" {
		if err := s.writer.RemoveFile(task.FilePath); err != nil {
			return nil, nil, fmt.Errorf("failed to remove task file: %w", err)
		}
	}

	if err := s.store.DeleteTask(ctx, id); err != nil {
		return nil, nil, fmt.Errorf("failed to delete task from store: %w", err)
	}

	return nil, &DeleteEntityOutput{
		Success: true,
		ID:      id,
		Message: fmt.Sprintf("Task %q deleted successfully", id),
	}, nil
}

func (s *Server) toolAddTaskDependency(ctx context.Context, _ *mcp.CallToolRequest, in AddTaskDependencyInput) (*mcp.CallToolResult, *TaskDependencyOutput, error) {
	if !s.cfg.MCP.AllowMutations {
		return nil, nil, errors.New("mutations are disabled in configuration")
	}

	id := strings.TrimSpace(in.ID)
	if id == "" {
		return nil, nil, errors.New("task id is required")
	}
	depID := strings.TrimSpace(in.DependencyID)
	if depID == "" {
		return nil, nil, errors.New("dependency_id is required")
	}

	if id == depID {
		return nil, nil, fmt.Errorf("task %q cannot depend on itself", id)
	}

	task, err := s.store.GetTask(ctx, id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, nil, fmt.Errorf("task %q not found", id)
		}
		return nil, nil, fmt.Errorf("failed to get task %q: %w", id, err)
	}

	if _, err := s.store.GetTask(ctx, depID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, nil, fmt.Errorf("dependency task %q not found", depID)
		}
		return nil, nil, fmt.Errorf("failed to get dependency task %q: %w", depID, err)
	}

	if slices.Contains(task.Dependencies, depID) {
		return nil, &TaskDependencyOutput{
			Success:      true,
			ID:           id,
			DependencyID: depID,
			Dependencies: task.Dependencies,
			Message:      fmt.Sprintf("Dependency %q already exists on task %q", depID, id),
		}, nil
	}

	tasks, err := s.store.ListTasks(ctx, model.FilterCriteria{})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list tasks for cycle check: %w", err)
	}

	depsGraph := make(map[string][]string, len(tasks)+1)
	taskFiles := make(map[string]string, len(tasks)+1)
	for _, t := range tasks {
		depsGraph[t.ID] = t.Dependencies
		taskFiles[t.ID] = t.FilePath
	}
	depsGraph[id] = append(slices.Clone(task.Dependencies), depID)

	cycleDiags := validator.DetectCycles(depsGraph, taskFiles)
	if len(cycleDiags) > 0 {
		var msgs []string
		for _, d := range cycleDiags {
			if len(d.Context) > 0 {
				msgs = append(msgs, d.Context...)
			} else {
				msgs = append(msgs, d.Message)
			}
		}
		return nil, nil, fmt.Errorf("circular dependency detected: %s", strings.Join(msgs, "; "))
	}

	task.Dependencies = append(task.Dependencies, depID)
	task.ModTime = time.Now()

	fm := model.TaskFrontmatter{
		Title:        task.Title,
		Status:       task.Status,
		Priority:     task.Priority,
		Milestone:    task.Milestone,
		Tags:         task.Tags,
		Summary:      task.Summary,
		Dependencies: task.Dependencies,
	}

	fileBytes, err := parser.Format(fm, task.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to format task markdown: %w", err)
	}

	filePath := task.FilePath
	if filePath == "" {
		filePath = filepath.Join(s.TasksDir(), fmt.Sprintf("%s.md", id))
		task.FilePath = filePath
	}

	if err := s.writer.WriteFile(filePath, fileBytes, 0644); err != nil {
		return nil, nil, fmt.Errorf("failed to save task file: %w", err)
	}

	if err := s.store.UpsertTask(ctx, task); err != nil {
		return nil, nil, fmt.Errorf("failed to update task in store: %w", err)
	}

	return nil, &TaskDependencyOutput{
		Success:      true,
		ID:           id,
		DependencyID: depID,
		Dependencies: task.Dependencies,
		Message:      fmt.Sprintf("Dependency %q added to task %q", depID, id),
	}, nil
}

func (s *Server) toolRemoveTaskDependency(ctx context.Context, _ *mcp.CallToolRequest, in RemoveTaskDependencyInput) (*mcp.CallToolResult, *TaskDependencyOutput, error) {
	if !s.cfg.MCP.AllowMutations {
		return nil, nil, errors.New("mutations are disabled in configuration")
	}

	id := strings.TrimSpace(in.ID)
	if id == "" {
		return nil, nil, errors.New("task id is required")
	}
	depID := strings.TrimSpace(in.DependencyID)
	if depID == "" {
		return nil, nil, errors.New("dependency_id is required")
	}

	task, err := s.store.GetTask(ctx, id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, nil, fmt.Errorf("task %q not found", id)
		}
		return nil, nil, fmt.Errorf("failed to get task %q: %w", id, err)
	}

	idx := slices.Index(task.Dependencies, depID)
	if idx == -1 {
		return nil, nil, fmt.Errorf("dependency %q not found on task %q", depID, id)
	}

	task.Dependencies = slices.Delete(task.Dependencies, idx, idx+1)
	task.ModTime = time.Now()

	fm := model.TaskFrontmatter{
		Title:        task.Title,
		Status:       task.Status,
		Priority:     task.Priority,
		Milestone:    task.Milestone,
		Tags:         task.Tags,
		Summary:      task.Summary,
		Dependencies: task.Dependencies,
	}

	fileBytes, err := parser.Format(fm, task.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to format task markdown: %w", err)
	}

	filePath := task.FilePath
	if filePath == "" {
		filePath = filepath.Join(s.TasksDir(), fmt.Sprintf("%s.md", id))
		task.FilePath = filePath
	}

	if err := s.writer.WriteFile(filePath, fileBytes, 0644); err != nil {
		return nil, nil, fmt.Errorf("failed to save task file: %w", err)
	}

	if err := s.store.UpsertTask(ctx, task); err != nil {
		return nil, nil, fmt.Errorf("failed to update task in store: %w", err)
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
	if !s.cfg.MCP.AllowMutations {
		return nil, nil, errors.New("mutations are disabled in configuration")
	}

	id := strings.TrimSpace(in.ID)
	if id == "" {
		return nil, nil, errors.New("task id is required")
	}

	note := strings.TrimSpace(in.Note)
	if note == "" {
		return nil, nil, errors.New("note content is required")
	}

	task, err := s.store.GetTask(ctx, id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, nil, fmt.Errorf("task %q not found", id)
		}
		return nil, nil, fmt.Errorf("failed to get task %q: %w", id, err)
	}

	newBody, err := parser.AppendTaskNote(task.Body, time.Now(), note)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to append note: %w", err)
	}

	task.Body = newBody
	total, completed, _ := parser.ExtractAcceptanceCriteria([]byte(newBody))
	task.TotalCriteria = total
	task.CompletedCriteria = completed
	task.ModTime = time.Now()

	fm := model.TaskFrontmatter{
		Title:        task.Title,
		Status:       task.Status,
		Priority:     task.Priority,
		Milestone:    task.Milestone,
		Tags:         task.Tags,
		Summary:      task.Summary,
		Dependencies: task.Dependencies,
	}

	fileBytes, err := parser.Format(fm, task.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to format task markdown: %w", err)
	}

	filePath := task.FilePath
	if filePath == "" {
		filePath = filepath.Join(s.TasksDir(), fmt.Sprintf("%s.md", id))
		task.FilePath = filePath
	}

	if err := s.writer.WriteFile(filePath, fileBytes, 0644); err != nil {
		return nil, nil, fmt.Errorf("failed to save task file: %w", err)
	}

	if err := s.store.UpsertTask(ctx, task); err != nil {
		return nil, nil, fmt.Errorf("failed to update task in store: %w", err)
	}

	return nil, &AddTaskNoteOutput{
		Success:           true,
		ID:                id,
		Note:              note,
		TotalCriteria:     total,
		CompletedCriteria: completed,
		Message:           fmt.Sprintf("Note appended to task %q under ## Notes", id),
	}, nil
}



