package model_test

import (
	"testing"

	"github.com/RJuho/jokateko/internal/model"
)

func TestPriority(t *testing.T) {
	for _, p := range model.ValidPriorities {
		if !p.IsValid() {
			t.Errorf("expected priority %q to be valid", p)
		}
	}

	invalid := model.Priority("urgent")
	if invalid.IsValid() {
		t.Error("expected 'urgent' to be invalid priority")
	}
}

func TestTask(t *testing.T) {
	task := model.Task{
		ID:           "260901-test-task",
		Title:        "Test Task",
		Status:       "in_progress",
		Priority:     model.PriorityHigh,
		Milestone:    "260915-mvp",
		Tags:         []string{"backend", "sqlite"},
		Summary:      "Task summary",
		Dependencies: []string{"260900-dep-task"},
		Body:         "Task body content",
	}

	if !task.HasDependency("260900-dep-task") {
		t.Error("expected task to have dependency '260900-dep-task'")
	}
	if task.HasDependency("other-task") {
		t.Error("did not expect task to have dependency 'other-task'")
	}

	if !task.HasTag("backend") || !task.HasTag("sqlite") {
		t.Error("expected task to have backend and sqlite tags")
	}
	if task.HasTag("frontend") {
		t.Error("did not expect task to have frontend tag")
	}

	if task.IsDone() {
		t.Error("expected task to not be done")
	}

	task.Status = "done"
	if !task.IsDone() {
		t.Error("expected task to be done")
	}

	fm := task.Frontmatter()
	if fm.Title != task.Title || fm.Status != "done" || fm.Priority != model.PriorityHigh {
		t.Errorf("frontmatter mismatch: %+v", fm)
	}
}

func TestMilestoneProgress(t *testing.T) {
	ms := model.Milestone{
		ID:     "260915-mvp",
		Title:  "MVP Release",
		Status: model.MilestoneStatusOpen,
	}

	// Case 1: 0 tasks assigned
	ms.RecalculateProgress(nil)
	if ms.TotalTasks != 0 || ms.CompletedTasks != 0 || ms.ProgressPercentage != 0.0 || ms.IsArchived {
		t.Errorf("unexpected metrics for 0 tasks: %+v", ms)
	}

	tasks := []model.Task{
		{ID: "t1", Milestone: "260915-mvp", Status: "done"},
		{ID: "t2", Milestone: "260915-mvp", Status: "in_progress"},
		{ID: "t3", Milestone: "260915-mvp", Status: "backlog"},
		{ID: "t4", Milestone: "other-ms", Status: "done"}, // Different milestone
	}

	// Case 2: 1 of 3 done (33.3%)
	ms.RecalculateProgress(tasks)
	if ms.TotalTasks != 3 || ms.CompletedTasks != 1 {
		t.Errorf("expected 3 total and 1 completed, got: total=%d, completed=%d", ms.TotalTasks, ms.CompletedTasks)
	}
	if ms.ProgressPercentage != 33.3 {
		t.Errorf("expected 33.3%% progress, got: %f", ms.ProgressPercentage)
	}
	if ms.IsArchived {
		t.Error("milestone should not be archived when tasks are incomplete")
	}

	// Case 3: 3 of 3 done (100% -> Auto-archive)
	tasks[1].Status = "done"
	tasks[2].Status = "done"
	ms.RecalculateProgress(tasks)
	if ms.TotalTasks != 3 || ms.CompletedTasks != 3 || ms.ProgressPercentage != 100.0 {
		t.Errorf("expected 100%% completion, got: %+v", ms)
	}
	if !ms.IsArchived {
		t.Error("milestone should be auto-archived when all tasks are done")
	}
}

func TestStrategy(t *testing.T) {
	for _, tier := range model.ValidTiers {
		if !tier.IsValid() {
			t.Errorf("expected tier %d to be valid", tier)
		}
		if tier.Name() == "Unknown" {
			t.Errorf("expected valid tier name for %d", tier)
		}
	}

	invalidTier := model.Tier(99)
	if invalidTier.IsValid() {
		t.Error("expected tier 99 to be invalid")
	}
	if invalidTier.Name() != "Unknown" {
		t.Errorf("expected 'Unknown' tier name, got: %s", invalidTier.Name())
	}

	strat := model.Strategy{
		ID:      "architecture",
		Title:   "Zero CGO Architecture",
		Tier:    model.TierCore,
		Tags:    []string{"architecture", "go"},
		Summary: "Summary text",
	}

	if !strat.HasTag("architecture") {
		t.Error("expected strategy to have 'architecture' tag")
	}

	fm := strat.Frontmatter()
	if fm.Tier != model.TierCore || fm.Title != strat.Title {
		t.Errorf("frontmatter mismatch: %+v", fm)
	}
}

func TestGlossaryTerm(t *testing.T) {
	term := model.GlossaryTerm{
		ID:      "tasks-as-code",
		Title:   "Tasks-as-Code",
		Tags:    []string{"methodology", "core"},
		Summary: "Summary text",
	}

	if !term.HasTag("methodology") {
		t.Error("expected glossary term to have 'methodology' tag")
	}

	fm := term.Frontmatter()
	if fm.Title != term.Title || len(fm.Tags) != 2 {
		t.Errorf("frontmatter mismatch: %+v", fm)
	}
}

func TestFilterCriteria(t *testing.T) {
	task := model.Task{
		ID:        "t1",
		Title:     "Build SQLite In-Memory Database",
		Status:    "in_progress",
		Priority:  model.PriorityHigh,
		Milestone: "mvp",
		Tags:      []string{"backend", "database"},
		Summary:   "Setup pure Go SQLite using modernc.org/sqlite",
	}

	// Empty criteria matches everything
	if !(model.FilterCriteria{}).Matches(task) {
		t.Error("empty filter should match task")
	}

	// Matching status
	if !(model.FilterCriteria{Status: "in_progress"}).Matches(task) {
		t.Error("status in_progress should match")
	}
	if (model.FilterCriteria{Status: "done"}).Matches(task) {
		t.Error("status done should not match")
	}

	// Matching milestone
	if !(model.FilterCriteria{Milestone: "mvp"}).Matches(task) {
		t.Error("milestone mvp should match")
	}
	if (model.FilterCriteria{Milestone: "v2"}).Matches(task) {
		t.Error("milestone v2 should not match")
	}

	// Matching tag
	if !(model.FilterCriteria{Tag: "database"}).Matches(task) {
		t.Error("tag database should match")
	}
	if (model.FilterCriteria{Tag: "ui"}).Matches(task) {
		t.Error("tag ui should not match")
	}

	// Matching priority
	if !(model.FilterCriteria{Priority: model.PriorityHigh}).Matches(task) {
		t.Error("priority high should match")
	}
	if (model.FilterCriteria{Priority: model.PriorityLow}).Matches(task) {
		t.Error("priority low should not match")
	}

	// Search query matching title or summary
	if !(model.FilterCriteria{SearchQuery: "sqlite"}).Matches(task) {
		t.Error("query 'sqlite' should match title and summary")
	}
	if !(model.FilterCriteria{SearchQuery: "MODERNC"}).Matches(task) {
		t.Error("query 'MODERNC' should match case-insensitively")
	}
	if (model.FilterCriteria{SearchQuery: "react"}).Matches(task) {
		t.Error("query 'react' should not match")
	}
}
