package mcp_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/RJuho/jokateko/internal/config"
	internalmcp "github.com/RJuho/jokateko/internal/mcp"
	"github.com/RJuho/jokateko/internal/model"
	"github.com/RJuho/jokateko/internal/store"
	"github.com/RJuho/jokateko/internal/writer"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func setupTestMCP(t *testing.T) (*internalmcp.Server, string, *store.Store, *mcp.ClientSession) {
	t.Helper()
	dir := t.TempDir()

	tasksDir := filepath.Join(dir, ".jokateko", "tasks")
	msDir := filepath.Join(dir, ".jokateko", "milestones")
	stratDir := filepath.Join(dir, ".jokateko", "strategies")
	glossDir := filepath.Join(dir, ".jokateko", "glossary")
	for _, d := range []string{tasksDir, msDir, stratDir, glossDir} {
		_ = os.MkdirAll(d, 0755)
	}

	cfg := config.Default(dir)
	cfg.Tags.EnforceAllowed = true
	cfg.Tags.Allowed = []string{"backend", "database", "api", "feature"}

	st, err := store.OpenMemory()
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	sc := writer.NewSuppressionCache(time.Second)
	wr := writer.New(sc)

	srv := internalmcp.New(cfg, dir, st, wr)

	// Set up in-memory client-server session
	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	ctx := t.Context()

	_, err = srv.MCPServer().Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("failed to connect server: %v", err)
	}

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1.0.0"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("failed to connect client: %v", err)
	}
	t.Cleanup(func() { _ = clientSession.Close() })

	return srv, dir, st, clientSession
}

func callToolJSON[Out any](t *testing.T, session *mcp.ClientSession, name string, args any) (Out, error) {
	t.Helper()
	ctx := t.Context()

	argsBytes, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("failed to marshal tool args: %v", err)
	}
	var rawArgs map[string]any
	_ = json.Unmarshal(argsBytes, &rawArgs)

	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      name,
		Arguments: rawArgs,
	})
	var zero Out
	if err != nil {
		return zero, err
	}
	if res.IsError {
		errMsg := ""
		for _, c := range res.Content {
			if tc, ok := c.(*mcp.TextContent); ok {
				errMsg += tc.Text
			}
		}
		return zero, &toolError{msg: errMsg}
	}

	for _, c := range res.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			var out Out
			if err := json.Unmarshal([]byte(tc.Text), &out); err == nil {
				return out, nil
			}
		}
	}

	return zero, nil
}

type toolError struct {
	msg string
}

func (e *toolError) Error() string {
	return e.msg
}

func TestMCP_TaskToolsLifecycle(t *testing.T) {
	_, dir, st, session := setupTestMCP(t)

	// 1. Test create_task
	createInput := internalmcp.CreateTaskInput{
		Title:        "Setup Database",
		Status:       "ready",
		Priority:     "high",
		Tags:         []string{"backend", "database"},
		Summary:      "Set up pure Go in-memory SQLite tables",
		Dependencies: []string{},
		Body: `## Acceptance Criteria
- [ ] Implement schema migrations
- [ ] Connect with sqlc
`,
	}

	created, err := callToolJSON[internalmcp.TaskDetail](t, session, "create_task", createInput)
	if err != nil {
		t.Fatalf("create_task failed: %v", err)
	}
	if created.Title != "Setup Database" || created.Status != "ready" {
		t.Errorf("unexpected created task: %+v", created)
	}
	if created.TotalCriteria != 2 || created.CompletedCriteria != 0 {
		t.Errorf("expected 2 total criteria, 0 completed, got %d/%d", created.CompletedCriteria, created.TotalCriteria)
	}

	// Verify file on disk
	taskFilePath := filepath.Join(dir, ".jokateko", "tasks", created.ID+".md")
	if _, err := os.Stat(taskFilePath); os.IsNotExist(err) {
		t.Fatalf("expected task file to exist at %s", taskFilePath)
	}

	// 2. Test get_task
	fetched, err := callToolJSON[internalmcp.TaskDetail](t, session, "get_task", internalmcp.GetTaskInput{ID: created.ID})
	if err != nil {
		t.Fatalf("get_task failed: %v", err)
	}
	if fetched.ID != created.ID || fetched.Title != created.Title {
		t.Errorf("fetched task mismatch: %+v", fetched)
	}

	// 3. Test list_tasks
	list, err := callToolJSON[[]internalmcp.TaskSummary](t, session, "list_tasks", internalmcp.ListTasksInput{})
	if err != nil {
		t.Fatalf("list_tasks failed: %v", err)
	}
	if len(list) != 1 || list[0].ID != created.ID {
		t.Errorf("expected 1 task in list, got %+v", list)
	}

	// 4. Test list_task_items
	itemsOutput, err := callToolJSON[internalmcp.ListTaskItemsOutput](t, session, "list_task_items", internalmcp.ListTaskItemsInput{ID: created.ID})
	if err != nil {
		t.Fatalf("list_task_items failed: %v", err)
	}
	if itemsOutput.Total != 2 || len(itemsOutput.Items) != 2 {
		t.Fatalf("expected 2 items, got %+v", itemsOutput)
	}
	if itemsOutput.Items[0].Completed || itemsOutput.Items[1].Completed {
		t.Errorf("expected all items to be uncompleted initially")
	}

	// 5. Test update_task_item: check item 1
	updatedItem, err := callToolJSON[internalmcp.UpdateTaskItemOutput](t, session, "update_task_item", internalmcp.UpdateTaskItemInput{
		ID:        created.ID,
		Index:     1,
		Completed: true,
	})
	if err != nil {
		t.Fatalf("update_task_item failed: %v", err)
	}
	if !updatedItem.Completed || updatedItem.CompletedItems != 1 {
		t.Errorf("expected item 1 to be completed, got %+v", updatedItem)
	}

	// 6. Test update_task_status:
	// 6a. Try setting status to 'done' directly -> MUST BE STRICTLY REJECTED
	_, err = callToolJSON[internalmcp.UpdateTaskStatusOutput](t, session, "update_task_status", internalmcp.UpdateTaskStatusInput{
		ID:     created.ID,
		Status: "done",
	})
	if err == nil {
		t.Fatal("expected update_task_status with 'done' to be strictly rejected")
	}
	if !strings.Contains(err.Error(), "complete_task") {
		t.Errorf("expected error to instruct using complete_task, got %v", err)
	}

	// 6b. Set status to 'in_progress' -> MUST SUCCEED
	statusOut, err := callToolJSON[internalmcp.UpdateTaskStatusOutput](t, session, "update_task_status", internalmcp.UpdateTaskStatusInput{
		ID:     created.ID,
		Status: "in_progress",
	})
	if err != nil {
		t.Fatalf("update_task_status failed: %v", err)
	}
	if statusOut.Status != "in_progress" {
		t.Errorf("expected status 'in_progress', got %q", statusOut.Status)
	}

	// 7. Test complete_task:
	// 7a. Try completing when item 2 is still unchecked -> MUST BE STRICTLY REJECTED by Open Checkbox Guard
	_, err = callToolJSON[internalmcp.CompleteTaskOutput](t, session, "complete_task", internalmcp.CompleteTaskInput{
		ID:       created.ID,
		Summary:  "Completed database setup",
		WhatDone: "Implemented database migrations",
		WhyDone:  "Required for data storage",
	})
	if err == nil {
		t.Fatal("expected complete_task to fail while open checkboxes remain")
	}
	if !strings.Contains(err.Error(), "acceptance criteria items remain uncompleted") {
		t.Errorf("expected open checkbox guard error message, got %v", err)
	}

	// 7b. Check item 2
	_, err = callToolJSON[internalmcp.UpdateTaskItemOutput](t, session, "update_task_item", internalmcp.UpdateTaskItemInput{
		ID:        created.ID,
		Index:     2,
		Completed: true,
	})
	if err != nil {
		t.Fatalf("failed to complete item 2: %v", err)
	}

	// 7c. Now complete_task MUST SUCCEED
	compOut, err := callToolJSON[internalmcp.CompleteTaskOutput](t, session, "complete_task", internalmcp.CompleteTaskInput{
		ID:       created.ID,
		Summary:  "Completed database setup",
		WhatDone: "Implemented database migrations with sqlc",
		WhyDone:  "Required for pure Go in-memory storage",
	})
	if err != nil {
		t.Fatalf("complete_task failed: %v", err)
	}
	if !compOut.Success || compOut.Status != "done" {
		t.Errorf("expected successful completion, got %+v", compOut)
	}

	// Verify Completion Summary in file
	contentBytes, _ := os.ReadFile(taskFilePath)
	fileContent := string(contentBytes)
	if !strings.Contains(fileContent, "## Completion Summary") || !strings.Contains(fileContent, "Implemented database migrations with sqlc") {
		t.Errorf("expected Completion Summary in file, got:\n%s", fileContent)
	}

	// 8. Test update_task_content
	newTitle := "Setup Database (V2)"
	updatedContent, err := callToolJSON[internalmcp.TaskDetail](t, session, "update_task_content", internalmcp.UpdateTaskContentInput{
		ID:    created.ID,
		Title: &newTitle,
	})
	if err != nil {
		t.Fatalf("update_task_content failed: %v", err)
	}
	if updatedContent.Title != "Setup Database (V2)" {
		t.Errorf("expected updated title, got %q", updatedContent.Title)
	}

	// Check database store has updated title
	taskInStore, _ := st.GetTask(t.Context(), created.ID)
	if taskInStore.Title != "Setup Database (V2)" {
		t.Errorf("store title mismatch: %q", taskInStore.Title)
	}
}

func TestMCP_Guards(t *testing.T) {
	_, _, st, session := setupTestMCP(t)
	ctx := t.Context()

	// 1. Tag validation guard: attempt to create task with unallowed tag
	_, err := callToolJSON[internalmcp.TaskDetail](t, session, "create_task", internalmcp.CreateTaskInput{
		Title:   "Unallowed Tag Task",
		Summary: "Summary",
		Tags:    []string{"forbidden-tag"},
		Body:    "Body",
	})
	if err == nil {
		t.Fatal("expected create_task with forbidden tag to fail")
	}
	if !strings.Contains(err.Error(), "forbidden-tag") {
		t.Errorf("expected tag error, got: %v", err)
	}

	// 2. Closed Milestone Guard:
	// Create closed milestone in store
	_ = st.UpsertMilestone(ctx, model.Milestone{
		ID:             "260901-closed-ms",
		Title:          "Closed Milestone",
		Status:         model.MilestoneStatusClosed,
		TotalTasks:     1,
		CompletedTasks: 1,
	})

	// Try attaching task without reopen_milestone -> MUST FAIL
	_, err = callToolJSON[internalmcp.TaskDetail](t, session, "create_task", internalmcp.CreateTaskInput{
		Title:     "Task in Closed MS",
		Milestone: "260901-closed-ms",
		Summary:   "Summary",
		Body:      "Body",
	})
	if err == nil {
		t.Fatal("expected closed milestone guard to reject attaching task")
	}
	if !strings.Contains(err.Error(), "reopen_milestone=true") {
		t.Errorf("expected reopen_milestone error message, got: %v", err)
	}

	// With reopen_milestone=true -> MUST SUCCEED
	created, err := callToolJSON[internalmcp.TaskDetail](t, session, "create_task", internalmcp.CreateTaskInput{
		Title:           "Task in Closed MS",
		Milestone:       "260901-closed-ms",
		ReopenMilestone: true,
		Summary:         "Summary",
		Body:            "Body",
	})
	if err != nil {
		t.Fatalf("expected create_task with reopen_milestone=true to succeed, got: %v", err)
	}
	if created.Milestone != "260901-closed-ms" {
		t.Errorf("unexpected milestone: %q", created.Milestone)
	}

	// 3. Unfinished Dependencies Guard:
	// Create blocker task (status: ready)
	blocker, _ := callToolJSON[internalmcp.TaskDetail](t, session, "create_task", internalmcp.CreateTaskInput{
		Title:   "Blocker Task",
		Status:  "ready",
		Summary: "Summary",
		Body:    "Body",
	})

	// Create dependent task
	dependent, _ := callToolJSON[internalmcp.TaskDetail](t, session, "create_task", internalmcp.CreateTaskInput{
		Title:        "Dependent Task",
		Status:       "ready",
		Dependencies: []string{blocker.ID},
		Summary:      "Summary",
		Body:         "Body",
	})

	// Try completing dependent task -> MUST FAIL
	_, err = callToolJSON[internalmcp.CompleteTaskOutput](t, session, "complete_task", internalmcp.CompleteTaskInput{
		ID:       dependent.ID,
		Summary:  "Summary",
		WhatDone: "Done",
		WhyDone:  "Rationale",
	})
	if err == nil {
		t.Fatal("expected complete_task to fail due to unfinished dependencies")
	}
	if !strings.Contains(err.Error(), "blocking dependencies remain unfinished") {
		t.Errorf("expected unfinished dependencies error, got: %v", err)
	}

	// With ignore_dependencies=true -> MUST SUCCEED
	comp, err := callToolJSON[internalmcp.CompleteTaskOutput](t, session, "complete_task", internalmcp.CompleteTaskInput{
		ID:                 dependent.ID,
		Summary:            "Summary",
		WhatDone:           "Done",
		WhyDone:            "Rationale",
		IgnoreDependencies: true,
	})
	if err != nil {
		t.Fatalf("complete_task with ignore_dependencies=true failed: %v", err)
	}
	if !comp.Success {
		t.Errorf("expected success: %+v", comp)
	}
}
