package model_test

import (
	"testing"

	"github.com/RJuho/jokateko/internal/model"
)

func TestPriorityRank(t *testing.T) {
	tests := []struct {
		p        model.Priority
		expected int
	}{
		{model.PriorityCritical, 0},
		{model.PriorityHigh, 1},
		{model.PriorityMedium, 2},
		{model.PriorityLow, 3},
		{model.Priority("unknown"), 2},
		{model.Priority(""), 2},
	}

	for _, tt := range tests {
		got := model.PriorityRank(tt.p)
		if got != tt.expected {
			t.Errorf("PriorityRank(%q) = %d, want %d", tt.p, got, tt.expected)
		}
	}
}

func TestSortTasksForColumn_DefaultWorkflow(t *testing.T) {
	t.Run("backlog sorts by priority then changed_at desc", func(t *testing.T) {
		col := model.Column{ID: "backlog", Name: "Backlog"}
		tasks := []model.Task{
			{ID: "task-1", Priority: model.PriorityLow, ChangedAt: "2026-09-01T10:00:00Z"},
			{ID: "task-2", Priority: model.PriorityCritical, ChangedAt: "2026-09-01T09:00:00Z"},
			{ID: "task-3", Priority: model.PriorityHigh, ChangedAt: "2026-09-02T10:00:00Z"},
			{ID: "task-4", Priority: model.PriorityHigh, ChangedAt: "2026-09-03T10:00:00Z"}, // newer high
			{ID: "task-5", Priority: model.PriorityHigh, ChangedAt: "2026-09-02T10:00:00Z"}, // same changed_at as task-3, tie-break id
		}

		model.SortTasksForColumn(tasks, col)

		expectedOrder := []string{"task-2", "task-4", "task-3", "task-5", "task-1"}
		for i, expected := range expectedOrder {
			if tasks[i].ID != expected {
				t.Fatalf("at index %d: expected %q, got %q", i, expected, tasks[i].ID)
			}
		}
	})

	t.Run("in_progress sorts by priority then target_at asc with empty target_at fallback", func(t *testing.T) {
		col := model.Column{ID: "in_progress", Name: "In Progress"}
		tasks := []model.Task{
			{ID: "t-no-target-old", Priority: model.PriorityHigh, TargetAt: "", ChangedAt: "2026-09-01T10:00:00Z"},
			{ID: "t-target-later", Priority: model.PriorityHigh, TargetAt: "2026-09-15T00:00:00Z", ChangedAt: "2026-09-01T10:00:00Z"},
			{ID: "t-target-sooner", Priority: model.PriorityHigh, TargetAt: "2026-09-10T00:00:00Z", ChangedAt: "2026-09-01T10:00:00Z"},
			{ID: "t-no-target-new", Priority: model.PriorityHigh, TargetAt: "", ChangedAt: "2026-09-05T10:00:00Z"},
			{ID: "t-target-same-sooner", Priority: model.PriorityHigh, TargetAt: "2026-09-10T00:00:00Z", ChangedAt: "2026-09-04T10:00:00Z"}, // same target, newer changed_at
			{ID: "t-critical", Priority: model.PriorityCritical, TargetAt: "2026-09-20T00:00:00Z"},
		}

		model.SortTasksForColumn(tasks, col)

		// Critical comes first regardless of target_at
		// For High priority tasks:
		// 1. Target dates first:
		//    - t-target-same-sooner (2026-09-10, newer changed_at 2026-09-04)
		//    - t-target-sooner (2026-09-10, changed_at 2026-09-01)
		//    - t-target-later (2026-09-15)
		// 2. Non-target dates next (changed_at desc):
		//    - t-no-target-new (changed_at 2026-09-05)
		//    - t-no-target-old (changed_at 2026-09-01)
		expectedOrder := []string{
			"t-critical",
			"t-target-same-sooner",
			"t-target-sooner",
			"t-target-later",
			"t-no-target-new",
			"t-no-target-old",
		}

		for i, expected := range expectedOrder {
			if tasks[i].ID != expected {
				t.Fatalf("at index %d: expected %q, got %q", i, expected, tasks[i].ID)
			}
		}
	})

	t.Run("done sorts by changed_at desc regardless of priority", func(t *testing.T) {
		col := model.Column{ID: "done", Name: "Done"}
		tasks := []model.Task{
			{ID: "done-crit-old", Priority: model.PriorityCritical, ChangedAt: "2026-09-01T10:00:00Z"},
			{ID: "done-low-new", Priority: model.PriorityLow, ChangedAt: "2026-09-05T10:00:00Z"},
			{ID: "done-med-mid", Priority: model.PriorityMedium, ChangedAt: "2026-09-03T10:00:00Z"},
		}

		model.SortTasksForColumn(tasks, col)

		// Expected order: done-low-new (Sep 5) -> done-med-mid (Sep 3) -> done-crit-old (Sep 1)
		expectedOrder := []string{"done-low-new", "done-med-mid", "done-crit-old"}
		for i, expected := range expectedOrder {
			if tasks[i].ID != expected {
				t.Fatalf("at index %d: expected %q, got %q", i, expected, tasks[i].ID)
			}
		}
	})
}

func TestSortTasksForColumn_CustomColumnConfig(t *testing.T) {
	t.Run("custom sort_by created_at desc", func(t *testing.T) {
		col := model.Column{
			ID:            "backlog",
			SortBy:        "created_at",
			SortDirection: "desc",
		}
		tasks := []model.Task{
			{ID: "t1", Priority: model.PriorityHigh, CreatedAt: "2026-09-01T10:00:00Z"},
			{ID: "t2", Priority: model.PriorityHigh, CreatedAt: "2026-09-05T10:00:00Z"},
			{ID: "t3", Priority: model.PriorityHigh, CreatedAt: "2026-09-03T10:00:00Z"},
		}

		model.SortTasksForColumn(tasks, col)

		expectedOrder := []string{"t2", "t3", "t1"}
		for i, expected := range expectedOrder {
			if tasks[i].ID != expected {
				t.Fatalf("at index %d: expected %q, got %q", i, expected, tasks[i].ID)
			}
		}
	})

	t.Run("custom sort_by title asc", func(t *testing.T) {
		col := model.Column{
			ID:            "ready",
			SortBy:        "title",
			SortDirection: "asc",
		}
		tasks := []model.Task{
			{ID: "t1", Priority: model.PriorityMedium, Title: "Zebra task"},
			{ID: "t2", Priority: model.PriorityMedium, Title: "Apple task"},
			{ID: "t3", Priority: model.PriorityMedium, Title: "Banana task"},
		}

		model.SortTasksForColumn(tasks, col)

		expectedOrder := []string{"t2", "t3", "t1"}
		for i, expected := range expectedOrder {
			if tasks[i].ID != expected {
				t.Fatalf("at index %d: expected %q, got %q", i, expected, tasks[i].ID)
			}
		}
	})
}

func TestSortTasksByColumnOrder(t *testing.T) {
	columns := []model.Column{
		{ID: "backlog", Name: "Backlog"},
		{ID: "ready", Name: "Ready"},
		{ID: "in_progress", Name: "In Progress"},
	}

	tasks := []model.Task{
		{ID: "t-prog-2", Status: "in_progress", Priority: model.PriorityMedium, ChangedAt: "2026-09-01T00:00:00Z"},
		{ID: "t-back-1", Status: "backlog", Priority: model.PriorityLow, ChangedAt: "2026-09-01T00:00:00Z"},
		{ID: "t-ready-1", Status: "ready", Priority: model.PriorityHigh, TargetAt: "2026-09-10T00:00:00Z"},
		{ID: "t-back-2", Status: "backlog", Priority: model.PriorityCritical, ChangedAt: "2026-09-01T00:00:00Z"},
		{ID: "t-prog-1", Status: "in_progress", Priority: model.PriorityCritical, TargetAt: "2026-09-12T00:00:00Z"},
	}

	sorted := model.SortTasksByColumnOrder(tasks, columns)

	// Expected column order: backlog -> ready -> in_progress
	// Inside backlog: t-back-2 (critical) before t-back-1 (low)
	// Inside ready: t-ready-1
	// Inside in_progress: t-prog-1 (critical) before t-prog-2 (medium)
	expectedIDs := []string{
		"t-back-2",
		"t-back-1",
		"t-ready-1",
		"t-prog-1",
		"t-prog-2",
	}

	for i, expected := range expectedIDs {
		if sorted[i].ID != expected {
			t.Fatalf("at index %d: expected %q, got %q", i, expected, sorted[i].ID)
		}
	}
}
