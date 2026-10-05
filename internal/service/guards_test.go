package service

import (
	"errors"
	"slices"
	"testing"

	"github.com/RJuho/jokateko/internal/model"
	"github.com/RJuho/jokateko/internal/validator"
)

// assertParseClean runs the same validation as `jokateko parse` over the workspace.
func assertParseClean(t *testing.T, dir string) {
	t.Helper()
	res, err := validator.ValidateWorkspace(dir)
	if err != nil {
		t.Fatalf("validate workspace: %v", err)
	}
	if res.HasErrors() {
		t.Fatalf("parse reports errors:\n%s", res.FormatReport())
	}
}

func mustCreateTask(t *testing.T, svc *Service, in NewTask) model.Task {
	t.Helper()
	if in.Summary == "" {
		in.Summary = "s"
	}
	task, err := svc.CreateTask(t.Context(), in)
	if err != nil {
		t.Fatalf("create task %q: %v", in.ID, err)
	}
	return task
}

func TestCreateTask_VerifiesReferences(t *testing.T) {
	svc, n, dir := newTestService(t)
	ctx := t.Context()
	mustCreateTask(t, svc, NewTask{ID: "a", Title: "A"})

	tests := []struct {
		name string
		in   NewTask
		want error
	}{
		{"unknown milestone", NewTask{Title: "X", Summary: "s", Milestone: "nope"}, ErrNotFound},
		{"unknown dependency", NewTask{Title: "X", Summary: "s", Dependencies: []string{"a", "nope"}}, ErrNotFound},
		{"self dependency", NewTask{ID: "self", Title: "X", Summary: "s", Dependencies: []string{"self"}}, ErrInvalid},
		{"unknown priority", NewTask{Title: "X", Summary: "s", Priority: "ultra"}, ErrInvalid},
		{"tag outside vocabulary", NewTask{Title: "X", Summary: "s", Tags: []string{"nope"}}, ErrInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := len(n.Events())
			if _, err := svc.CreateTask(ctx, tt.in); !errors.Is(err, tt.want) {
				t.Fatalf("got %v, want %v", err, tt.want)
			}
			if len(n.Events()) != before {
				t.Fatal("event emitted for rejected create")
			}
		})
	}

	b := mustCreateTask(t, svc, NewTask{ID: "b", Title: "B", Dependencies: []string{" a ", ""}, Tags: []string{"backend"}})
	if !slices.Equal(b.Dependencies, []string{"a"}) {
		t.Fatalf("dependencies = %v, want [a]", b.Dependencies)
	}
	assertParseClean(t, dir)
}

func TestPatchTask_VerifiesReferences(t *testing.T) {
	svc, _, dir := newTestService(t)
	ctx := t.Context()
	mustCreateTask(t, svc, NewTask{ID: "a", Title: "A"})
	mustCreateTask(t, svc, NewTask{ID: "b", Title: "B", Dependencies: []string{"a"}})

	tests := []struct {
		name  string
		id    string
		patch TaskPatch
		want  error
	}{
		{"unknown milestone", "a", TaskPatch{Milestone: new("nope")}, ErrNotFound},
		{"unknown dependency", "a", TaskPatch{Dependencies: &[]string{"nope"}}, ErrNotFound},
		{"cycle", "a", TaskPatch{Dependencies: &[]string{"b"}}, ErrConflict},
		{"unknown priority", "a", TaskPatch{Priority: new(model.Priority("ultra"))}, ErrInvalid},
		{"empty priority", "a", TaskPatch{Priority: new(model.Priority(""))}, ErrInvalid},
		{"empty title", "a", TaskPatch{Title: new(" ")}, ErrInvalid},
		{"tag outside vocabulary", "a", TaskPatch{Tags: &[]string{"nope"}}, ErrInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := svc.PatchTask(ctx, tt.id, tt.patch); !errors.Is(err, tt.want) {
				t.Fatalf("got %v, want %v", err, tt.want)
			}
		})
	}
	assertParseClean(t, dir)
}

func TestPriority_UsesConfiguredList(t *testing.T) {
	svc, _, _ := newTestService(t)
	svc.cfg.Priorities = []model.PriorityConfig{{ID: "p1", Name: "P1"}, {ID: "p2", Name: "P2"}}

	task := mustCreateTask(t, svc, NewTask{ID: "a", Title: "A"})
	if task.Priority != "p1" {
		t.Fatalf("default priority = %q, want first configured p1", task.Priority)
	}
	if _, err := svc.CreateTask(t.Context(), NewTask{Title: "B", Summary: "s", Priority: model.PriorityHigh}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("built-in priority not configured: got %v, want ErrInvalid", err)
	}
	if _, err := svc.PatchTask(t.Context(), "a", TaskPatch{Priority: new(model.Priority("p2"))}); err != nil {
		t.Fatalf("configured priority rejected: %v", err)
	}
}

func TestMilestoneArchivedGuard(t *testing.T) {
	svc, _, _ := newTestService(t)
	ctx := t.Context()
	if _, err := svc.CreateMilestone(ctx, NewMilestone{ID: "closed", Title: "Closed", Status: model.MilestoneStatusClosed}); err != nil {
		t.Fatal(err)
	}

	_, err := svc.CreateTask(ctx, NewTask{Title: "A", Summary: "s", Milestone: "closed"})
	if !errors.Is(err, ErrConflict) || ErrorCode(err) != CodeMilestoneArchived {
		t.Fatalf("got %v (code %q), want ErrConflict with %q", err, ErrorCode(err), CodeMilestoneArchived)
	}
	a := mustCreateTask(t, svc, NewTask{ID: "a", Title: "A", Milestone: "closed", ReopenMilestone: true})

	// Unchanged milestone does not trigger the guard.
	if _, err := svc.PatchTask(ctx, a.ID, TaskPatch{Title: new("A2"), Milestone: new("closed")}); err != nil {
		t.Fatalf("edit with unchanged milestone: %v", err)
	}

	mustCreateTask(t, svc, NewTask{ID: "b", Title: "B"})
	if _, err := svc.PatchTask(ctx, "b", TaskPatch{Milestone: new("closed")}); ErrorCode(err) != CodeMilestoneArchived {
		t.Fatalf("patch onto closed milestone: got %v", err)
	}
	if _, err := svc.PatchTask(ctx, "b", TaskPatch{Milestone: new("closed"), ReopenMilestone: true}); err != nil {
		t.Fatalf("patch with reopen: %v", err)
	}
}

func TestIsArchived(t *testing.T) {
	tests := []struct {
		ms   model.Milestone
		want bool
	}{
		{model.Milestone{Status: model.MilestoneStatusOpen}, false},
		{model.Milestone{Status: model.MilestoneStatusClosed}, true},
		{model.Milestone{Status: model.MilestoneStatusOpen, TotalTasks: 2, CompletedTasks: 1}, false},
		{model.Milestone{Status: model.MilestoneStatusOpen, TotalTasks: 2, CompletedTasks: 2}, true},
	}
	for _, tt := range tests {
		if got := IsArchived(tt.ms); got != tt.want {
			t.Errorf("IsArchived(%+v) = %v, want %v", tt.ms, got, tt.want)
		}
	}
}

func TestEntityTagsEnforced(t *testing.T) {
	svc, _, _ := newTestService(t)
	ctx := t.Context()
	bad := []string{"nope"}

	if _, err := svc.CreateMilestone(ctx, NewMilestone{Title: "M", Tags: bad}); !errors.Is(err, ErrInvalid) {
		t.Errorf("milestone create: got %v", err)
	}
	if _, err := svc.CreateStrategy(ctx, NewStrategy{Title: "S", Tags: bad}); !errors.Is(err, ErrInvalid) {
		t.Errorf("strategy create: got %v", err)
	}
	if _, err := svc.CreateGlossaryTerm(ctx, NewGlossaryTerm{Title: "G", Tags: bad}); !errors.Is(err, ErrInvalid) {
		t.Errorf("glossary create: got %v", err)
	}

	ms, err := svc.CreateMilestone(ctx, NewMilestone{Title: "M"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.UpdateMilestone(ctx, ms.ID, func(m *model.Milestone) error { m.Tags = bad; return nil }); !errors.Is(err, ErrInvalid) {
		t.Errorf("milestone update: got %v", err)
	}
	st, err := svc.CreateStrategy(ctx, NewStrategy{Title: "S"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.UpdateStrategy(ctx, st.ID, func(s *model.Strategy) error { s.Tags = bad; return nil }); !errors.Is(err, ErrInvalid) {
		t.Errorf("strategy update: got %v", err)
	}
	term, err := svc.CreateGlossaryTerm(ctx, NewGlossaryTerm{Title: "G"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.UpdateGlossaryTerm(ctx, term.ID, func(g *model.GlossaryTerm) error { g.Tags = bad; return nil }); !errors.Is(err, ErrInvalid) {
		t.Errorf("glossary update: got %v", err)
	}
}

func TestLegacyTagsStayEditable(t *testing.T) {
	svc, _, _ := newTestService(t)
	ctx := t.Context()
	mustCreateTask(t, svc, NewTask{ID: "a", Title: "A"})
	// Simulate a tag that was allowed before the vocabulary changed.
	svc.cfg.Tags.Allowed = append(svc.cfg.Tags.Allowed, "legacy")
	if _, err := svc.PatchTask(ctx, "a", TaskPatch{Tags: &[]string{"legacy"}}); err != nil {
		t.Fatal(err)
	}
	svc.cfg.Tags.Allowed = svc.cfg.Tags.Allowed[:len(svc.cfg.Tags.Allowed)-1]

	if _, err := svc.PatchTask(ctx, "a", TaskPatch{Title: new("A2"), Tags: &[]string{"legacy", "backend"}}); err != nil {
		t.Fatalf("keeping a legacy tag while adding an allowed one: %v", err)
	}
}

func TestForceDeleteTask_RemovesDependencyReferences(t *testing.T) {
	svc, n, dir := newTestService(t)
	ctx := t.Context()
	mustCreateTask(t, svc, NewTask{ID: "a", Title: "A"})
	mustCreateTask(t, svc, NewTask{ID: "other", Title: "Other"})
	b := mustCreateTask(t, svc, NewTask{ID: "b", Title: "B", Dependencies: []string{"a", "other"}})

	if err := svc.DeleteTask(ctx, "a", false); !errors.Is(err, ErrConflict) {
		t.Fatalf("delete without force: got %v, want ErrConflict", err)
	}
	before := len(n.Events())
	if err := svc.DeleteTask(ctx, "a", true); err != nil {
		t.Fatal(err)
	}
	if got := n.Events()[before:]; !slices.Equal(got, []string{"task.updated", "task.deleted"}) {
		t.Fatalf("events = %v, want [task.updated task.deleted]", got)
	}
	got, err := svc.GetTask(ctx, "b")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got.Dependencies, []string{"other"}) {
		t.Fatalf("dependencies = %v, want [other]", got.Dependencies)
	}
	assertFileContains(t, b.FilePath, "dependencies = ['other']")
	assertParseClean(t, dir)
}

func TestForceDeleteMilestone_DetachesTasks(t *testing.T) {
	svc, n, dir := newTestService(t)
	ctx := t.Context()
	if _, err := svc.CreateMilestone(ctx, NewMilestone{ID: "m", Title: "M"}); err != nil {
		t.Fatal(err)
	}
	mustCreateTask(t, svc, NewTask{ID: "a", Title: "A", Milestone: "m"})
	mustCreateTask(t, svc, NewTask{ID: "b", Title: "B", Milestone: "m"})

	before := len(n.Events())
	if err := svc.DeleteMilestone(ctx, "m", true); err != nil {
		t.Fatal(err)
	}
	if got := n.Events()[before:]; !slices.Equal(got, []string{"task.updated", "task.updated", "milestone.deleted"}) {
		t.Fatalf("events = %v", got)
	}
	for _, id := range []string{"a", "b"} {
		task, err := svc.GetTask(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		if task.Milestone != "" {
			t.Fatalf("task %s milestone = %q, want empty", id, task.Milestone)
		}
	}
	assertParseClean(t, dir)
}
