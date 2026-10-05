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
	"github.com/RJuho/jokateko/internal/service"
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

	srv := internalmcp.New(service.New(cfg, dir, st, wr, nil))

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

	// 4. References and priority are verified by the service for MCP writes too.
	refTests := []struct {
		name    string
		tool    string
		args    any
		wantErr string
	}{
		{"create unknown milestone", "create_task", internalmcp.CreateTaskInput{Title: "X", Summary: "s", Milestone: "260101-ghost"}, "not found"},
		{"create unknown dependency", "create_task", internalmcp.CreateTaskInput{Title: "X", Summary: "s", Dependencies: []string{"260101-ghost"}}, "not found"},
		{"create unknown priority", "create_task", internalmcp.CreateTaskInput{Title: "X", Summary: "s", Priority: "ultra"}, "not a configured priority"},
		{"update cycle", "update_task_content", internalmcp.UpdateTaskContentInput{ID: blocker.ID, Dependencies: &[]string{dependent.ID}}, "circular dependency"},
		{"update unknown priority", "update_task_content", internalmcp.UpdateTaskContentInput{ID: blocker.ID, Priority: new("ultra")}, "not a configured priority"},
		{"update closed milestone", "update_task_content", internalmcp.UpdateTaskContentInput{ID: blocker.ID, Milestone: new("260901-closed-ms")}, "reopen_milestone=true"},
	}
	for _, tt := range refTests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := callToolJSON[map[string]any](t, session, tt.tool, tt.args)
			expectToolError(t, err, tt.wantErr)
		})
	}

	// 5. Tag vocabulary applies to every entity update.
	term, err := callToolJSON[map[string]any](t, session, "create_glossary_term", internalmcp.CreateGlossaryTermInput{Title: "Tagged Term", Summary: "s"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = callToolJSON[map[string]any](t, session, "update_glossary_term", internalmcp.UpdateGlossaryTermInput{ID: term["id"].(string), Tags: []string{"nope"}})
	expectToolError(t, err, "not permitted")
}

func TestMCP_TaskDependencies(t *testing.T) {
	_, _, st, session := setupTestMCP(t)

	// Create task A
	taskA, err := callToolJSON[internalmcp.TaskDetail](t, session, "create_task", internalmcp.CreateTaskInput{
		Title:   "Task Alpha",
		Status:  "ready",
		Summary: "Alpha summary",
		Body:    "Alpha body",
	})
	if err != nil {
		t.Fatalf("failed to create task A: %v", err)
	}

	// Create task B
	taskB, err := callToolJSON[internalmcp.TaskDetail](t, session, "create_task", internalmcp.CreateTaskInput{
		Title:   "Task Beta",
		Status:  "ready",
		Summary: "Beta summary",
		Body:    "Beta body",
	})
	if err != nil {
		t.Fatalf("failed to create task B: %v", err)
	}

	// Create task C
	taskC, err := callToolJSON[internalmcp.TaskDetail](t, session, "create_task", internalmcp.CreateTaskInput{
		Title:   "Task Gamma",
		Status:  "ready",
		Summary: "Gamma summary",
		Body:    "Gamma body",
	})
	if err != nil {
		t.Fatalf("failed to create task C: %v", err)
	}

	// 1. Self-dependency rejection
	_, err = callToolJSON[internalmcp.TaskDependencyOutput](t, session, "add_task_dependency", internalmcp.AddTaskDependencyInput{
		ID:           taskA.ID,
		DependencyID: taskA.ID,
	})
	if err == nil {
		t.Fatal("expected self-dependency to be rejected")
	}
	if !strings.Contains(err.Error(), "cannot depend on itself") {
		t.Errorf("expected 'cannot depend on itself' in error, got: %v", err)
	}

	// 2. Non-existent task / dependency
	_, err = callToolJSON[internalmcp.TaskDependencyOutput](t, session, "add_task_dependency", internalmcp.AddTaskDependencyInput{
		ID:           "non-existent-task",
		DependencyID: taskA.ID,
	})
	if err == nil {
		t.Fatal("expected error for non-existent target task")
	}

	_, err = callToolJSON[internalmcp.TaskDependencyOutput](t, session, "add_task_dependency", internalmcp.AddTaskDependencyInput{
		ID:           taskA.ID,
		DependencyID: "non-existent-dep",
	})
	if err == nil {
		t.Fatal("expected error for non-existent dependency task")
	}

	// 3. Add dependency: B depends on A
	out, err := callToolJSON[internalmcp.TaskDependencyOutput](t, session, "add_task_dependency", internalmcp.AddTaskDependencyInput{
		ID:           taskB.ID,
		DependencyID: taskA.ID,
	})
	if err != nil {
		t.Fatalf("failed to add dependency: %v", err)
	}
	if !out.Success || len(out.Dependencies) != 1 || out.Dependencies[0] != taskA.ID {
		t.Fatalf("unexpected dependency output: %+v", out)
	}

	// Verify in store
	updatedB, err := st.GetTask(t.Context(), taskB.ID)
	if err != nil {
		t.Fatalf("failed to fetch task B: %v", err)
	}
	if len(updatedB.Dependencies) != 1 || updatedB.Dependencies[0] != taskA.ID {
		t.Errorf("store dependency mismatch on task B: %+v", updatedB.Dependencies)
	}

	// 4. Duplicate edge handling (idempotent, no duplicate entries)
	outDup, err := callToolJSON[internalmcp.TaskDependencyOutput](t, session, "add_task_dependency", internalmcp.AddTaskDependencyInput{
		ID:           taskB.ID,
		DependencyID: taskA.ID,
	})
	if err != nil {
		t.Fatalf("duplicate add_task_dependency failed: %v", err)
	}
	if len(outDup.Dependencies) != 1 {
		t.Errorf("expected exactly 1 dependency after duplicate add, got: %+v", outDup.Dependencies)
	}

	// 5. Add dependency: C depends on B
	_, err = callToolJSON[internalmcp.TaskDependencyOutput](t, session, "add_task_dependency", internalmcp.AddTaskDependencyInput{
		ID:           taskC.ID,
		DependencyID: taskB.ID,
	})
	if err != nil {
		t.Fatalf("failed to add dependency C -> B: %v", err)
	}

	// 6. Direct cycle rejection: A depends on B (when B depends on A) -> MUST FAIL
	_, err = callToolJSON[internalmcp.TaskDependencyOutput](t, session, "add_task_dependency", internalmcp.AddTaskDependencyInput{
		ID:           taskA.ID,
		DependencyID: taskB.ID,
	})
	if err == nil {
		t.Fatal("expected cycle rejection (A -> B when B -> A)")
	}
	if !strings.Contains(err.Error(), "circular dependency detected") {
		t.Errorf("expected circular dependency error, got: %v", err)
	}

	// 7. Transitive cycle rejection: A depends on C (when C -> B -> A) -> MUST FAIL
	_, err = callToolJSON[internalmcp.TaskDependencyOutput](t, session, "add_task_dependency", internalmcp.AddTaskDependencyInput{
		ID:           taskA.ID,
		DependencyID: taskC.ID,
	})
	if err == nil {
		t.Fatal("expected transitive cycle rejection (A -> C when C -> B -> A)")
	}
	if !strings.Contains(err.Error(), "circular dependency detected") {
		t.Errorf("expected circular dependency error, got: %v", err)
	}

	// 8. Remove non-existent dependency -> MUST FAIL
	_, err = callToolJSON[internalmcp.TaskDependencyOutput](t, session, "remove_task_dependency", internalmcp.RemoveTaskDependencyInput{
		ID:           taskB.ID,
		DependencyID: taskC.ID,
	})
	if err == nil {
		t.Fatal("expected remove_task_dependency to fail for non-existent dependency")
	}

	// 9. Remove existing dependency: remove A from B
	remOut, err := callToolJSON[internalmcp.TaskDependencyOutput](t, session, "remove_task_dependency", internalmcp.RemoveTaskDependencyInput{
		ID:           taskB.ID,
		DependencyID: taskA.ID,
	})
	if err != nil {
		t.Fatalf("failed to remove dependency: %v", err)
	}
	if !remOut.Success || len(remOut.Dependencies) != 0 {
		t.Fatalf("unexpected remove dependency output: %+v", remOut)
	}

	// Verify in store
	updatedBAfterRem, err := st.GetTask(t.Context(), taskB.ID)
	if err != nil {
		t.Fatalf("failed to fetch task B after remove: %v", err)
	}
	if len(updatedBAfterRem.Dependencies) != 0 {
		t.Errorf("expected 0 dependencies after remove, got: %+v", updatedBAfterRem.Dependencies)
	}
}

func TestMCP_TaskNotesAndEditableStates(t *testing.T) {
	srv, _, st, session := setupTestMCP(t)
	_ = srv

	// 1. Create a task in backlog with 1 acceptance criterion
	createOut, err := callToolJSON[internalmcp.TaskDetail](t, session, "create_task", internalmcp.CreateTaskInput{
		Title:    "Notes and Edit Test",
		Summary:  "Testing editable states and notes",
		Priority: "medium",
		Tags:     []string{"backend"},
		Body:     "## Acceptance Criteria\n- [ ] Initial criteria\n",
	})
	if err != nil {
		t.Fatalf("failed to create task: %v", err)
	}
	if createOut.Status != "backlog" || createOut.TotalCriteria != 1 {
		t.Fatalf("unexpected task state: %+v", createOut)
	}

	// 2. In backlog (editable state), modifying body should SUCCEED
	newBody := "## Acceptance Criteria\n- [ ] Modified initial criteria\n"
	updOut, err := callToolJSON[internalmcp.TaskDetail](t, session, "update_task_content", internalmcp.UpdateTaskContentInput{
		ID:   createOut.ID,
		Body: &newBody,
	})
	if err != nil {
		t.Fatalf("expected update_task_content body to succeed in backlog: %v", err)
	}
	if !strings.Contains(updOut.Body, "Modified initial criteria") {
		t.Fatalf("expected updated body, got: %s", updOut.Body)
	}

	// 3. Move task to "ready" (non-editable state by default)
	_, err = callToolJSON[internalmcp.UpdateTaskStatusOutput](t, session, "update_task_status", internalmcp.UpdateTaskStatusInput{
		ID:     createOut.ID,
		Status: "ready",
	})
	if err != nil {
		t.Fatalf("failed to update status to ready: %v", err)
	}

	// 4. In ready (non-editable state), modifying body specification text should FAIL
	attemptBody := "## Acceptance Criteria\n- [ ] Attempted rewrite\n"
	_, err = callToolJSON[internalmcp.TaskDetail](t, session, "update_task_content", internalmcp.UpdateTaskContentInput{
		ID:   createOut.ID,
		Body: &attemptBody,
	})
	if err == nil {
		t.Fatal("expected update_task_content body to fail in non-editable status 'ready'")
	}
	if !strings.Contains(err.Error(), "only editable in") {
		t.Errorf("expected error message mentioning editable status, got: %v", err)
	}

	// 5. In ready, toggling checkbox (- [ ] -> - [x]) should SUCCEED
	checkedBody := "## Acceptance Criteria\n- [x] Modified initial criteria\n"
	toggleOut, err := callToolJSON[internalmcp.TaskDetail](t, session, "update_task_content", internalmcp.UpdateTaskContentInput{
		ID:   createOut.ID,
		Body: &checkedBody,
	})
	if err != nil {
		t.Fatalf("expected checkbox toggle to succeed in ready status: %v", err)
	}
	if toggleOut.CompletedCriteria != 1 {
		t.Errorf("expected 1 completed criteria, got %d", toggleOut.CompletedCriteria)
	}

	// 6. Append note with a new required checkbox via add_task_note
	noteOut, err := callToolJSON[internalmcp.AddTaskNoteOutput](t, session, "add_task_note", internalmcp.AddTaskNoteInput{
		ID:   createOut.ID,
		Note: "Investigation complete.\n- [ ] Follow-up bug check",
	})
	if err != nil {
		t.Fatalf("failed to add task note: %v", err)
	}
	if !noteOut.Success {
		t.Fatalf("expected noteOut.Success to be true")
	}
	if noteOut.TotalCriteria != 2 || noteOut.CompletedCriteria != 1 {
		t.Fatalf("expected total 2 and completed 1 criteria, got total %d, completed %d", noteOut.TotalCriteria, noteOut.CompletedCriteria)
	}

	// Verify in store
	taskInStore, err := st.GetTask(t.Context(), createOut.ID)
	if err != nil {
		t.Fatalf("failed to get task from store: %v", err)
	}
	if !strings.Contains(taskInStore.Body, "## Notes") || !strings.Contains(taskInStore.Body, "Follow-up bug check") {
		t.Errorf("expected notes with follow-up in store body, got:\n%s", taskInStore.Body)
	}

	// 7. Attempt to complete task -> should FAIL because new checkbox in Notes is not completed!
	_, err = callToolJSON[internalmcp.CompleteTaskOutput](t, session, "complete_task", internalmcp.CompleteTaskInput{
		ID:       createOut.ID,
		Summary:  "Finishing work",
		WhatDone: "Implemented test feature",
		WhyDone:  "Required for verification",
	})
	if err == nil {
		t.Fatal("expected complete_task to fail due to unchecked criteria in Notes")
	}
	if !strings.Contains(err.Error(), "remain uncompleted") {
		t.Errorf("expected uncompleted criteria error, got: %v", err)
	}

	// 8. Toggle the note's checkbox to checked
	fullyCheckedBody := strings.Replace(taskInStore.Body, "- [ ] Follow-up bug check", "- [x] Follow-up bug check", 1)
	_, err = callToolJSON[internalmcp.TaskDetail](t, session, "update_task_content", internalmcp.UpdateTaskContentInput{
		ID:   createOut.ID,
		Body: &fullyCheckedBody,
	})
	if err != nil {
		t.Fatalf("expected toggling note checkbox to succeed: %v", err)
	}

	// 9. Complete task -> should SUCCEED now
	compOut, err := callToolJSON[internalmcp.CompleteTaskOutput](t, session, "complete_task", internalmcp.CompleteTaskInput{
		ID:       createOut.ID,
		Summary:  "Finishing work successfully",
		WhatDone: "Implemented test feature and checked all criteria",
		WhyDone:  "Required for verification",
	})
	if err != nil {
		t.Fatalf("expected complete_task to succeed now: %v", err)
	}
	if compOut.Status != "done" {
		t.Errorf("expected status 'done', got %s", compOut.Status)
	}
}

func TestMCPTaskSortingAndBoardAlignment(t *testing.T) {
	_, _, _, session := setupTestMCP(t)

	// Create tasks in backlog
	bLow, err := callToolJSON[internalmcp.TaskSummary](t, session, "create_task", internalmcp.CreateTaskInput{
		Title:    "Backlog Low",
		Priority: "low",
		Status:   "backlog",
		Summary:  "Backlog low priority task",
	})
	if err != nil {
		t.Fatalf("create_task failed: %v", err)
	}
	// Give different changed_at / timestamps
	bCrit, err := callToolJSON[internalmcp.TaskSummary](t, session, "create_task", internalmcp.CreateTaskInput{
		Title:    "Backlog Critical",
		Priority: "critical",
		Status:   "backlog",
		Summary:  "Backlog critical priority task",
	})
	if err != nil {
		t.Fatalf("create_task failed: %v", err)
	}

	// Create tasks in in_progress
	pTargetLater, err := callToolJSON[internalmcp.TaskSummary](t, session, "create_task", internalmcp.CreateTaskInput{
		Title:    "Progress Target Later",
		Priority: "high",
		Status:   "in_progress",
		Summary:  "Progress target later task",
		TargetAt: "2026-09-20T00:00:00Z",
	})
	if err != nil {
		t.Fatalf("create_task failed: %v", err)
	}
	pTargetSooner, err := callToolJSON[internalmcp.TaskSummary](t, session, "create_task", internalmcp.CreateTaskInput{
		Title:    "Progress Target Sooner",
		Priority: "high",
		Status:   "in_progress",
		Summary:  "Progress target sooner task",
		TargetAt: "2026-09-10T00:00:00Z",
	})
	if err != nil {
		t.Fatalf("create_task failed: %v", err)
	}
	pNoTarget, err := callToolJSON[internalmcp.TaskSummary](t, session, "create_task", internalmcp.CreateTaskInput{
		Title:    "Progress No Target",
		Priority: "high",
		Status:   "in_progress",
		Summary:  "Progress no target task",
	})
	if err != nil {
		t.Fatalf("create_task failed: %v", err)
	}

	// 1. Test list_tasks for status="in_progress"
	progList, err := callToolJSON[[]internalmcp.TaskSummary](t, session, "list_tasks", internalmcp.ListTasksInput{
		Status: "in_progress",
	})
	if err != nil {
		t.Fatalf("list_tasks in_progress failed: %v", err)
	}
	if len(progList) != 3 {
		t.Fatalf("expected 3 in_progress tasks, got %d", len(progList))
	}
	// Expected order: sooner deadline (pTargetSooner) -> later deadline (pTargetLater) -> no deadline (pNoTarget)
	if progList[0].ID != pTargetSooner.ID {
		t.Errorf("expected progList[0] to be %q, got %q", pTargetSooner.ID, progList[0].ID)
	}
	if progList[1].ID != pTargetLater.ID {
		t.Errorf("expected progList[1] to be %q, got %q", pTargetLater.ID, progList[1].ID)
	}
	if progList[2].ID != pNoTarget.ID {
		t.Errorf("expected progList[2] to be %q, got %q", pNoTarget.ID, progList[2].ID)
	}

	// 2. Test list_tasks without status -> orders by board column order first, then column rules
	allList, err := callToolJSON[[]internalmcp.TaskSummary](t, session, "list_tasks", internalmcp.ListTasksInput{})
	if err != nil {
		t.Fatalf("list_tasks all failed: %v", err)
	}
	if len(allList) != 5 {
		t.Fatalf("expected 5 tasks, got %d", len(allList))
	}
	// Backlog columns appear before In Progress columns
	// Backlog critical (bCrit) before Backlog low (bLow)
	// Followed by In Progress tasks
	expectedAll := []string{
		bCrit.ID,
		bLow.ID,
		pTargetSooner.ID,
		pTargetLater.ID,
		pNoTarget.ID,
	}
	for i, expID := range expectedAll {
		if allList[i].ID != expID {
			t.Errorf("allList[%d] expected %q, got %q", i, expID, allList[i].ID)
		}
	}
}
