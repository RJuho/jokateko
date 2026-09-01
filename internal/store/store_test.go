package store_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

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
