package store_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/RJuho/jokateko/internal/model"
	"github.com/RJuho/jokateko/internal/store"
)

func TestOpenMemory(t *testing.T) {
	st, err := store.OpenMemory()
	if err != nil {
		t.Fatalf("failed to open in-memory store: %v", err)
	}
	defer func() {
		_ = st.Close()
	}()

	if st.DB() == nil {
		t.Fatal("expected non-nil DB")
	}
	if st.Queries() == nil {
		t.Fatal("expected non-nil Queries")
	}

	ctx := context.Background()
	if err := st.DB().PingContext(ctx); err != nil {
		t.Fatalf("ping failed: %v", err)
	}
}

func TestPragmasApplied(t *testing.T) {
	st, err := store.OpenMemory()
	if err != nil {
		t.Fatalf("failed to open in-memory store: %v", err)
	}
	defer func() {
		_ = st.Close()
	}()

	ctx := context.Background()

	var busyTimeout int
	if err := st.DB().QueryRowContext(ctx, "PRAGMA busy_timeout;").Scan(&busyTimeout); err != nil {
		t.Fatalf("failed to query busy_timeout: %v", err)
	}
	if busyTimeout != 10000 {
		t.Errorf("expected busy_timeout = 10000, got %d", busyTimeout)
	}

	var journalMode string
	if err := st.DB().QueryRowContext(ctx, "PRAGMA journal_mode;").Scan(&journalMode); err != nil {
		t.Fatalf("failed to query journal_mode: %v", err)
	}
	if journalMode != "memory" {
		t.Errorf("expected journal_mode = memory, got %s", journalMode)
	}

	var synchronous int
	if err := st.DB().QueryRowContext(ctx, "PRAGMA synchronous;").Scan(&synchronous); err != nil {
		t.Fatalf("failed to query synchronous: %v", err)
	}
	if synchronous != 0 {
		t.Errorf("expected synchronous = 0 (OFF), got %d", synchronous)
	}

	var foreignKeys int
	if err := st.DB().QueryRowContext(ctx, "PRAGMA foreign_keys;").Scan(&foreignKeys); err != nil {
		t.Fatalf("failed to query foreign_keys: %v", err)
	}
	if foreignKeys != 1 {
		t.Errorf("expected foreign_keys = 1 (ON), got %d", foreignKeys)
	}
}

func TestSchemaApplied(t *testing.T) {
	st, err := store.Open("file:test_schema?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer func() {
		_ = st.Close()
	}()

	ctx := context.Background()

	// Verify all expected tables exist in sqlite_master
	expectedTables := []string{
		"tasks",
		"milestones",
		"strategies",
		"glossary",
		"task_dependencies",
		"entity_tags",
		"fts_entities",
	}

	for _, tbl := range expectedTables {
		var name string
		query := "SELECT name FROM sqlite_master WHERE type IN ('table', 'virtual') AND name = ?;"
		err := st.DB().QueryRowContext(ctx, query, tbl).Scan(&name)
		if err != nil {
			t.Errorf("table %q does not exist in schema: %v", tbl, err)
		}
	}
}

func TestBasicOperations(t *testing.T) {
	st, err := store.Open("file:test_ops?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer func() {
		_ = st.Close()
	}()

	ctx := context.Background()
	now := time.Now().Unix()

	// 1. Task Operations
	err = st.Queries().UpsertTask(ctx, store.UpsertTaskParams{
		ID:                "260901-task-1",
		Title:             "Task One",
		Status:            "in_progress",
		Priority:          "high",
		MilestoneID:       "260915-mvp",
		Summary:           "Summary one",
		Body:              "## Description",
		TotalCriteria:     3,
		CompletedCriteria: 1,
		Filepath:          ".jokateko/tasks/260901-task-1.md",
		Mtime:             now,
	})
	if err != nil {
		t.Fatalf("failed to upsert task: %v", err)
	}

	task, err := st.Queries().GetTask(ctx, "260901-task-1")
	if err != nil {
		t.Fatalf("failed to get task: %v", err)
	}
	if task.Title != "Task One" || task.Status != "in_progress" || task.TotalCriteria != 3 {
		t.Errorf("unexpected task fields: %+v", task)
	}

	// 2. Dependencies
	err = st.Queries().AddTaskDependency(ctx, store.AddTaskDependencyParams{
		TaskID:          "260901-task-1",
		DependsOnTaskID: "260900-init",
	})
	if err != nil {
		t.Fatalf("failed to add dependency: %v", err)
	}

	deps, err := st.Queries().GetTaskDependencies(ctx, "260901-task-1")
	if err != nil {
		t.Fatalf("failed to get dependencies: %v", err)
	}
	if len(deps) != 1 || deps[0] != "260900-init" {
		t.Errorf("unexpected dependencies: %v", deps)
	}

	// 3. Entity Tags
	err = st.Queries().AddEntityTag(ctx, store.AddEntityTagParams{
		EntityType: "task",
		EntityID:   "260901-task-1",
		Tag:        "backend",
	})
	if err != nil {
		t.Fatalf("failed to add tag: %v", err)
	}

	tagCounts, err := st.Queries().GetTagCounts(ctx)
	if err != nil {
		t.Fatalf("failed to get tag counts: %v", err)
	}
	if len(tagCounts) != 1 || tagCounts[0].Tag != "backend" || tagCounts[0].TaskCount != 1 {
		t.Errorf("unexpected tag counts: %+v", tagCounts)
	}

	// 4. Milestone & Metrics
	err = st.Queries().UpsertMilestone(ctx, store.UpsertMilestoneParams{
		ID:         "260915-mvp",
		Title:      "MVP Milestone",
		Status:     "open",
		TargetDate: "2026-09-15",
		Summary:    "MVP summary",
		Body:       "Body text",
		Filepath:   ".jokateko/milestones/260915-mvp.md",
		Mtime:      now,
	})
	if err != nil {
		t.Fatalf("failed to upsert milestone: %v", err)
	}

	metrics, err := st.Queries().GetMilestoneTaskMetrics(ctx, "260915-mvp")
	if err != nil {
		t.Fatalf("failed to get milestone task metrics: %v", err)
	}
	if metrics.TotalTasks != 1 || metrics.CompletedTasks != 0 {
		t.Errorf("unexpected milestone metrics: %+v", metrics)
	}
}

func TestTransactions(t *testing.T) {
	st, err := store.Open("file:test_tx?mode=memory&cache=shared&_pragma=journal_mode(MEMORY)")
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer func() {
		_ = st.Close()
	}()

	ctx := context.Background()

	// Successful transaction
	err = st.WithTx(ctx, func(q *store.Queries) error {
		return q.UpsertTask(ctx, store.UpsertTaskParams{
			ID:       "tx-task-1",
			Title:    "TX Task 1",
			Status:   "ready",
			Priority: "medium",
			Summary:  "Summary",
		})
	})
	if err != nil {
		t.Fatalf("expected transaction to commit: %v", err)
	}

	// Verify task exists
	_, err = st.Queries().GetTask(ctx, "tx-task-1")
	if err != nil {
		t.Fatalf("expected tx-task-1 to exist: %v", err)
	}

	// Failed transaction (rollback)
	errExpected := errors.New("simulated failure")
	err = st.WithTx(ctx, func(q *store.Queries) error {
		if err := q.UpsertTask(ctx, store.UpsertTaskParams{
			ID:       "tx-task-rollback",
			Title:    "Rollback Task",
			Status:   "ready",
			Priority: "low",
			Summary:  "Summary",
		}); err != nil {
			return err
		}
		return errExpected
	})

	if !errors.Is(err, errExpected) {
		t.Errorf("expected errExpected, got: %v", err)
	}

	// Verify rolled-back task does not exist
	_, err = st.Queries().GetTask(ctx, "tx-task-rollback")
	if err == nil {
		t.Error("expected tx-task-rollback to not exist after rollback")
	}
}

func TestConcurrency(t *testing.T) {
	st, err := store.Open("file:test_concurrency?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer func() {
		_ = st.Close()
	}()

	ctx := context.Background()

	// Seed initial task
	_ = st.Queries().UpsertTask(ctx, store.UpsertTaskParams{
		ID:       "concurrency-task",
		Title:    "Initial Title",
		Status:   "ready",
		Priority: "medium",
		Summary:  "Initial summary",
	})

	const goroutines = 20
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			if idx%2 == 0 {
				// Writer
				_ = st.WithTx(ctx, func(q *store.Queries) error {
					return q.UpdateTaskStatus(ctx, store.UpdateTaskStatusParams{
						ID:     "concurrency-task",
						Status: "in_progress",
						Mtime:  time.Now().Unix(),
					})
				})
			} else {
				// Reader
				st.RLock()
				defer st.RUnlock()
				_, _ = st.Queries().GetTask(ctx, "concurrency-task")
			}
		}(i)
	}

	wg.Wait()
}

func TestEntityCRUD(t *testing.T) {
	st, err := store.OpenMemory()
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer func() {
		_ = st.Close()
	}()

	ctx := context.Background()

	// 1. Tasks
	task := model.Task{
		ID:           "260901-task-crud",
		Title:        "CRUD Task",
		Status:       "ready",
		Priority:     model.PriorityHigh,
		Milestone:    "260915-crud-ms",
		Tags:         []string{"backend", "db"},
		Summary:      "Task summary",
		Dependencies: []string{"260900-init"},
		Body:         "## Details",
	}
	if err := st.UpsertTask(ctx, task); err != nil {
		t.Fatalf("UpsertTask failed: %v", err)
	}

	gotTask, err := st.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}
	if gotTask.Title != task.Title || len(gotTask.Tags) != 2 || len(gotTask.Dependencies) != 1 {
		t.Errorf("unexpected task: %+v", gotTask)
	}

	// Update status
	if err := st.UpdateTaskStatus(ctx, task.ID, "done"); err != nil {
		t.Fatalf("UpdateTaskStatus failed: %v", err)
	}
	// Update criteria
	if err := st.UpdateTaskCriteria(ctx, task.ID, 3, 3, "## Details updated"); err != nil {
		t.Fatalf("UpdateTaskCriteria failed: %v", err)
	}
	gotTask, _ = st.GetTask(ctx, task.ID)
	if gotTask.Status != "done" || gotTask.TotalCriteria != 3 || gotTask.CompletedCriteria != 3 {
		t.Errorf("unexpected updated task: %+v", gotTask)
	}

	// Filter tasks
	filtered, err := st.ListTasks(ctx, model.FilterCriteria{Status: "done", Tag: "backend"})
	if err != nil {
		t.Fatalf("ListTasks failed: %v", err)
	}
	if len(filtered) != 1 {
		t.Errorf("expected 1 task matching filter, got %d", len(filtered))
	}

	filteredNone, _ := st.ListTasks(ctx, model.FilterCriteria{Status: "ready"})
	if len(filteredNone) != 0 {
		t.Errorf("expected 0 tasks, got %d", len(filteredNone))
	}

	// 2. Milestones
	ms := model.Milestone{
		ID:         "260915-crud-ms",
		Title:      "CRUD Milestone",
		Status:     model.MilestoneStatusOpen,
		TargetDate: "2026-09-15",
		Tags:       []string{"release"},
		Summary:    "Milestone summary",
		Body:       "Body text",
	}
	if err := st.UpsertMilestone(ctx, ms); err != nil {
		t.Fatalf("UpsertMilestone failed: %v", err)
	}

	gotMS, err := st.GetMilestone(ctx, ms.ID)
	if err != nil {
		t.Fatalf("GetMilestone failed: %v", err)
	}
	// TotalTasks should be 1 (task above), CompletedTasks should be 1, Progress 100%, IsArchived true
	if gotMS.TotalTasks != 1 || gotMS.CompletedTasks != 1 || gotMS.ProgressPercentage != 100.0 || !gotMS.IsArchived {
		t.Errorf("unexpected milestone metrics: %+v", gotMS)
	}

	allMS, err := st.ListMilestones(ctx)
	if err != nil || len(allMS) != 1 {
		t.Errorf("ListMilestones failed: %v, len=%d", err, len(allMS))
	}

	// 3. Strategies
	strat := model.Strategy{
		ID:      "strat-cgo-free",
		Title:   "Zero CGO Invariant",
		Tier:    model.TierCore,
		Tags:    []string{"architecture", "backend"},
		Summary: "Pure Go without C dependencies",
		Body:    "Strictly enforce pure Go.",
	}
	if err := st.UpsertStrategy(ctx, strat); err != nil {
		t.Fatalf("UpsertStrategy failed: %v", err)
	}

	gotStrat, err := st.GetStrategy(ctx, strat.ID)
	if err != nil {
		t.Fatalf("GetStrategy failed: %v", err)
	}
	if gotStrat.Tier != model.TierCore || len(gotStrat.Tags) != 2 {
		t.Errorf("unexpected strategy: %+v", gotStrat)
	}

	tier1Strats, err := st.ListStrategies(ctx, model.TierCore)
	if err != nil || len(tier1Strats) != 1 {
		t.Errorf("ListStrategies tier 1 failed: %v, count=%d", err, len(tier1Strats))
	}
	tier2Strats, _ := st.ListStrategies(ctx, model.TierDomain)
	if len(tier2Strats) != 0 {
		t.Errorf("expected 0 tier 2 strategies, got %d", len(tier2Strats))
	}

	// 4. Glossary
	term := model.GlossaryTerm{
		ID:      "term-task-as-code",
		Title:   "Tasks-as-Code",
		Tags:    []string{"concept"},
		Summary: "Markdown files as the source of truth for work items",
		Body:    "Each markdown file represents an actionable ticket.",
	}
	if err := st.UpsertGlossaryTerm(ctx, term); err != nil {
		t.Fatalf("UpsertGlossaryTerm failed: %v", err)
	}

	gotTerm, err := st.GetGlossaryTerm(ctx, term.ID)
	if err != nil {
		t.Fatalf("GetGlossaryTerm failed: %v", err)
	}
	if gotTerm.Title != term.Title || len(gotTerm.Tags) != 1 {
		t.Errorf("unexpected term: %+v", gotTerm)
	}

	allTerms, err := st.ListGlossaryTerms(ctx)
	if err != nil || len(allTerms) != 1 {
		t.Errorf("ListGlossaryTerms failed: %v, count=%d", err, len(allTerms))
	}

	// 5. Tags Aggregation
	tagCounts, err := st.GetTagCounts(ctx)
	if err != nil {
		t.Fatalf("GetTagCounts failed: %v", err)
	}
	var foundBackend bool
	for _, tc := range tagCounts {
		if tc.Tag == "backend" {
			foundBackend = true
			if tc.TaskCount != 1 || tc.StrategyCount != 1 {
				t.Errorf("unexpected counts for 'backend': %+v", tc)
			}
		}
	}
	if !foundBackend {
		t.Errorf("tag 'backend' not found in tagCounts: %+v", tagCounts)
	}

	// 6. Delete operations
	if err := st.DeleteTask(ctx, task.ID); err != nil {
		t.Fatalf("DeleteTask failed: %v", err)
	}
	if _, err := st.GetTask(ctx, task.ID); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("expected ErrNotFound for deleted task, got %v", err)
	}

	if err := st.DeleteMilestone(ctx, ms.ID); err != nil {
		t.Fatalf("DeleteMilestone failed: %v", err)
	}
	if _, err := st.GetMilestone(ctx, ms.ID); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("expected ErrNotFound for deleted milestone, got %v", err)
	}

	if err := st.DeleteStrategy(ctx, strat.ID); err != nil {
		t.Fatalf("DeleteStrategy failed: %v", err)
	}
	if _, err := st.GetStrategy(ctx, strat.ID); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("expected ErrNotFound for deleted strategy, got %v", err)
	}

	if err := st.DeleteGlossaryTerm(ctx, term.ID); err != nil {
		t.Fatalf("DeleteGlossaryTerm failed: %v", err)
	}
	if _, err := st.GetGlossaryTerm(ctx, term.ID); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("expected ErrNotFound for deleted glossary term, got %v", err)
	}
}

func TestBoardState(t *testing.T) {
	st, err := store.OpenMemory()
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer func() {
		_ = st.Close()
	}()

	ctx := context.Background()

	_ = st.UpsertTask(ctx, model.Task{
		ID:       "task-ready-1",
		Title:    "Ready Task 1",
		Status:   "ready",
		Priority: model.PriorityMedium,
		Summary:  "Summary",
	})
	_ = st.UpsertTask(ctx, model.Task{
		ID:       "task-ready-2",
		Title:    "Ready Task 2",
		Status:   "ready",
		Priority: model.PriorityHigh,
		Summary:  "Summary",
	})
	_ = st.UpsertTask(ctx, model.Task{
		ID:       "task-in-prog",
		Title:    "Progress Task",
		Status:   "in_progress",
		Priority: model.PriorityLow,
		Summary:  "Summary",
	})

	cols := []model.Column{
		{ID: "backlog", Name: "Backlog", Color: "slate"},
		{ID: "ready", Name: "Ready", Color: "blue"},
		{ID: "in_progress", Name: "In Progress", Color: "amber"},
		{ID: "done", Name: "Done", Color: "emerald"},
	}

	board, err := st.GetBoardState(ctx, "Test Project", cols)
	if err != nil {
		t.Fatalf("GetBoardState failed: %v", err)
	}

	if board.ProjectName != "Test Project" || len(board.Columns) != 4 {
		t.Fatalf("unexpected board: %+v", board)
	}

	expectedCounts := map[string]int{
		"backlog":     0,
		"ready":       2,
		"in_progress": 1,
		"done":        0,
	}
	for _, col := range board.Columns {
		if col.Count != expectedCounts[col.ID] {
			t.Errorf("col %q expected count %d, got %d", col.ID, expectedCounts[col.ID], col.Count)
		}
	}
}
