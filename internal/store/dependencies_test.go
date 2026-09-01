package store_test

import (
	"context"
	"testing"

	"github.com/RJuho/jokateko/internal/model"
	"github.com/RJuho/jokateko/internal/store"
)

func TestDependencies(t *testing.T) {
	st, err := store.OpenMemory()
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer func() {
		_ = st.Close()
	}()

	ctx := context.Background()

	// Scenario setup:
	// taskA ("ready")
	// taskB ("ready")
	// taskC depends on taskA and taskB
	// taskD depends on taskA only

	taskA := model.Task{
		ID:       "task-a",
		Title:    "Foundation Task A",
		Status:   "ready",
		Priority: model.PriorityHigh,
		Summary:  "Summary A",
	}
	taskB := model.Task{
		ID:       "task-b",
		Title:    "Foundation Task B",
		Status:   "ready",
		Priority: model.PriorityMedium,
		Summary:  "Summary B",
	}
	taskC := model.Task{
		ID:           "task-c",
		Title:        "Dependent Task C",
		Status:       "ready",
		Priority:     model.PriorityMedium,
		Summary:      "Summary C",
		Dependencies: []string{"task-a", "task-b"},
	}
	taskD := model.Task{
		ID:           "task-d",
		Title:        "Dependent Task D",
		Status:       "ready",
		Priority:     model.PriorityLow,
		Summary:      "Summary D",
		Dependencies: []string{"task-a"},
	}

	for _, task := range []model.Task{taskA, taskB, taskC, taskD} {
		if err := st.UpsertTask(ctx, task); err != nil {
			t.Fatalf("failed to upsert %q: %v", task.ID, err)
		}
	}

	// 1. Initial State Checks
	t.Run("tasks with no dependencies report ready", func(t *testing.T) {
		done, err := st.AreDependenciesDone(ctx, "task-a")
		if err != nil || !done {
			t.Errorf("task-a should have satisfied dependencies: done=%v, err=%v", done, err)
		}

		blocking, err := st.GetBlockingTasks(ctx, "task-a")
		if err != nil || len(blocking) != 0 {
			t.Errorf("task-a should have zero blocking tasks: %+v, err=%v", blocking, err)
		}
	})

	t.Run("tasks with unfinished dependencies report not done", func(t *testing.T) {
		done, err := st.AreDependenciesDone(ctx, "task-c")
		if err != nil || done {
			t.Errorf("task-c should have unsatisfied dependencies: done=%v, err=%v", done, err)
		}

		blocking, err := st.GetBlockingTasks(ctx, "task-c")
		if err != nil || len(blocking) != 2 {
			t.Fatalf("expected 2 blocking tasks for task-c, got %d: %+v", len(blocking), blocking)
		}

		downstream, err := st.GetDownstreamTasks(ctx, "task-a")
		if err != nil || len(downstream) != 2 {
			t.Fatalf("expected 2 downstream tasks for task-a (task-c and task-d), got %d: %+v", len(downstream), downstream)
		}
	})

	// 2. Partial Completion: Task A is completed
	t.Run("completing task-a unblocks task-d but leaves task-c blocked by task-b", func(t *testing.T) {
		if err := st.UpdateTaskStatus(ctx, "task-a", "done"); err != nil {
			t.Fatalf("failed to update task-a to done: %v", err)
		}

		// task-d only depended on task-a, so it is now unblocked
		doneD, err := st.AreDependenciesDone(ctx, "task-d")
		if err != nil || !doneD {
			t.Errorf("task-d should now be unblocked: done=%v, err=%v", doneD, err)
		}

		// task-c still depends on task-b
		doneC, err := st.AreDependenciesDone(ctx, "task-c")
		if err != nil || doneC {
			t.Errorf("task-c should still be blocked by task-b: done=%v, err=%v", doneC, err)
		}

		blockingC, err := st.GetBlockingTasks(ctx, "task-c")
		if err != nil || len(blockingC) != 1 || blockingC[0].ID != "task-b" {
			t.Errorf("expected only task-b blocking task-c, got: %+v", blockingC)
		}

		// FindUnblockedTasks from task-a should return task-d, but NOT task-c
		unblockedByA, err := st.FindUnblockedTasks(ctx, "task-a")
		if err != nil {
			t.Fatalf("FindUnblockedTasks failed: %v", err)
		}
		if len(unblockedByA) != 1 || unblockedByA[0].ID != "task-d" {
			t.Errorf("expected only task-d unblocked by task-a, got: %+v", unblockedByA)
		}
	})

	// 3. Full Completion: Task B is completed
	t.Run("completing task-b resolves remaining blocker and unblocks task-c", func(t *testing.T) {
		if err := st.UpdateTaskStatus(ctx, "task-b", "done"); err != nil {
			t.Fatalf("failed to update task-b to done: %v", err)
		}

		doneC, err := st.AreDependenciesDone(ctx, "task-c")
		if err != nil || !doneC {
			t.Errorf("task-c should now have all dependencies satisfied: done=%v, err=%v", doneC, err)
		}

		blockingC, err := st.GetBlockingTasks(ctx, "task-c")
		if err != nil || len(blockingC) != 0 {
			t.Errorf("task-c should now have zero blocking tasks: %+v", blockingC)
		}

		unblockedByB, err := st.FindUnblockedTasks(ctx, "task-b")
		if err != nil {
			t.Fatalf("FindUnblockedTasks failed: %v", err)
		}
		if len(unblockedByB) != 1 || unblockedByB[0].ID != "task-c" {
			t.Errorf("expected task-c unblocked by task-b, got: %+v", unblockedByB)
		}
	})
}
