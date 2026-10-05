package service

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/RJuho/jokateko/internal/config"
)

// assertLastEvent fails unless the most recent notifier event equals want.
func assertLastEvent(t *testing.T, n *recordingNotifier, want string) {
	t.Helper()
	events := n.Events()
	if len(events) == 0 {
		t.Fatalf("no events recorded, want last event %q", want)
	}
	if got := events[len(events)-1]; got != want {
		t.Fatalf("last event = %q, want %q (all: %v)", got, want, events)
	}
}

// assertFileContains fails unless the file at path contains every substring in wants.
func assertFileContains(t *testing.T, path string, wants ...string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	for _, want := range wants {
		if !strings.Contains(string(data), want) {
			t.Fatalf("file %s does not contain %q:\n%s", path, want, data)
		}
	}
}

// assertFileNotContains fails if the file at path contains substr.
func assertFileNotContains(t *testing.T, path, substr string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if strings.Contains(string(data), substr) {
		t.Fatalf("file %s unexpectedly contains %q:\n%s", path, substr, data)
	}
}

func TestGetTask_NotFound(t *testing.T) {
	svc, _, _ := newTestService(t)
	_, err := svc.GetTask(t.Context(), "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
	if !strings.Contains(err.Error(), `task "missing" not found`) {
		t.Fatalf("unexpected error message %q", err.Error())
	}
}

func TestGetTask_StoreError(t *testing.T) {
	svc, _, _ := newTestService(t)
	_ = svc.Store().Close()
	_, err := svc.GetTask(t.Context(), "any")
	if err == nil || errors.Is(err, ErrNotFound) {
		t.Fatalf("expected a non-NotFound store error, got %v", err)
	}
	if !strings.Contains(err.Error(), "failed to get task") {
		t.Fatalf("unexpected error message %q", err.Error())
	}
}

func TestDefaultTaskStatus(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(c *config.Config)
		want   string
	}{
		{
			name:   "configured default create state",
			mutate: func(c *config.Config) { c.Board.DefaultCreateState = " ready " },
			want:   "ready",
		},
		{
			name: "first column when default is empty",
			mutate: func(c *config.Config) {
				c.Board.DefaultCreateState = ""
				c.Board.Columns = []config.ColumnConfig{{ID: "todo"}, {ID: "doing"}}
			},
			want: "todo",
		},
		{
			name: "backlog when nothing configured",
			mutate: func(c *config.Config) {
				c.Board.DefaultCreateState = ""
				c.Board.Columns = nil
			},
			want: "backlog",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, n, _ := newTestService(t)
			tt.mutate(svc.cfg)
			if got := svc.defaultTaskStatus(); got != tt.want {
				t.Fatalf("defaultTaskStatus() = %q, want %q", got, tt.want)
			}
			task, err := svc.CreateTask(t.Context(), NewTask{Title: "T", Summary: "s"})
			if err != nil {
				t.Fatal(err)
			}
			if task.Status != tt.want {
				t.Fatalf("created status = %q, want %q", task.Status, tt.want)
			}
			assertLastEvent(t, n, "task.created")
			assertFileContains(t, task.FilePath, "status = '"+tt.want+"'")
		})
	}
}

func TestCheckStatus_NoColumnsAcceptsAnything(t *testing.T) {
	svc, _, _ := newTestService(t)
	svc.cfg.Board.Columns = nil
	if err := svc.CheckStatus("whatever"); err != nil {
		t.Fatalf("expected any status to be accepted, got %v", err)
	}
}

func TestCreateTask_Validation(t *testing.T) {
	tests := []struct {
		name string
		in   NewTask
	}{
		{"missing title", NewTask{Summary: "s"}},
		{"missing summary", NewTask{Title: "T"}},
		{"bad target", NewTask{Title: "T", Summary: "s", TargetAt: "tomorrow"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, n, _ := newTestService(t)
			if _, err := svc.CreateTask(t.Context(), tt.in); !errors.Is(err, ErrInvalid) {
				t.Fatalf("expected ErrInvalid, got %v", err)
			}
			if got := n.Events(); len(got) != 0 {
				t.Fatalf("expected no events, got %v", got)
			}
		})
	}
}

func TestCreateTask_WriteError(t *testing.T) {
	svc, n, _ := newTestService(t)
	// Occupy the tasks directory path with a regular file so MkdirAll fails.
	if err := os.MkdirAll(filepath.Dir(svc.Dirs().Tasks), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(svc.Dirs().Tasks, []byte("not a dir"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := svc.CreateTask(t.Context(), NewTask{Title: "T", Summary: "s"})
	if err == nil || !strings.Contains(err.Error(), "failed to save task file") {
		t.Fatalf("expected save error, got %v", err)
	}
	if got := n.Events(); len(got) != 0 {
		t.Fatalf("expected no events, got %v", got)
	}
}

func TestNormalizeTargetAt(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    string
		wantErr bool
	}{
		{"rfc3339 utc", "2026-12-24T18:00:00Z", "2026-12-24T18:00:00Z", false},
		{"rfc3339 offset", "2026-12-24T20:00:00+02:00", "2026-12-24T18:00:00Z", false},
		{"date only", "2026-12-24", "2026-12-24T00:00:00Z", false},
		{"clear", "", "", false},
		{"whitespace clears", "   ", "", false},
		{"invalid", "24.12.2026", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeTargetAt(tt.raw)
			if tt.wantErr {
				if !errors.Is(err, ErrInvalid) {
					t.Fatalf("expected ErrInvalid, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("NormalizeTargetAt(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}

func TestSetTaskTarget(t *testing.T) {
	svc, n, _ := newTestService(t)
	ctx := t.Context()
	task, err := svc.CreateTask(ctx, NewTask{Title: "T", Summary: "s"})
	if err != nil {
		t.Fatal(err)
	}

	steps := []struct {
		name    string
		raw     string
		want    string
		wantErr error
	}{
		{"rfc3339", "2026-12-24T18:00:00Z", "2026-12-24T18:00:00Z", nil},
		{"date only", "2027-01-15", "2027-01-15T00:00:00Z", nil},
		{"invalid", "not-a-date", "", ErrInvalid},
		{"clear", "", "", nil},
	}
	for _, st := range steps {
		t.Run(st.name, func(t *testing.T) {
			before := len(n.Events())
			got, err := svc.SetTaskTarget(ctx, task.ID, st.raw)
			if st.wantErr != nil {
				if !errors.Is(err, st.wantErr) {
					t.Fatalf("expected %v, got %v", st.wantErr, err)
				}
				if len(n.Events()) != before {
					t.Fatal("event emitted for rejected update")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got.TargetAt != st.want {
				t.Fatalf("TargetAt = %q, want %q", got.TargetAt, st.want)
			}
			assertLastEvent(t, n, "task.updated")
			if st.want == "" {
				assertFileNotContains(t, task.FilePath, "target_at")
			} else {
				assertFileContains(t, task.FilePath, "target_at = '"+st.want+"'")
			}
		})
	}

	if _, err := svc.SetTaskTarget(ctx, "missing", "2026-01-01"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown task: expected ErrNotFound, got %v", err)
	}
}

func TestAddTaskNote(t *testing.T) {
	svc, n, _ := newTestService(t)
	ctx := t.Context()
	task, err := svc.CreateTask(ctx, NewTask{Title: "T", Summary: "s", Body: "## Context\nSome context."})
	if err != nil {
		t.Fatal(err)
	}

	got, err := svc.AddTaskNote(ctx, task.ID, "  Found a corner case.  ")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got.Body, "## Notes") || !strings.Contains(got.Body, "Found a corner case.") {
		t.Fatalf("note missing from body: %q", got.Body)
	}
	assertLastEvent(t, n, "task.updated")
	assertFileContains(t, task.FilePath, "## Notes", "### [2026-10-04 12:00 UTC]", "Found a corner case.")

	t.Run("empty note", func(t *testing.T) {
		if _, err := svc.AddTaskNote(ctx, task.ID, "   "); !errors.Is(err, ErrInvalid) {
			t.Fatalf("expected ErrInvalid, got %v", err)
		}
	})
	t.Run("unknown task", func(t *testing.T) {
		if _, err := svc.AddTaskNote(ctx, "missing", "note"); !errors.Is(err, ErrNotFound) {
			t.Fatalf("expected ErrNotFound, got %v", err)
		}
	})
}

func TestUpdateTaskStatus_Validation(t *testing.T) {
	svc, _, _ := newTestService(t)
	ctx := t.Context()
	task, _ := svc.CreateTask(ctx, NewTask{Title: "T", Summary: "s"})
	if _, err := svc.UpdateTaskStatus(ctx, task.ID, " "); !errors.Is(err, ErrInvalid) {
		t.Fatalf("empty status: expected ErrInvalid, got %v", err)
	}
	if _, err := svc.UpdateTaskStatus(ctx, task.ID, "nope"); !errors.Is(err, ErrInvalid) {
		t.Fatalf("unknown status: expected ErrInvalid, got %v", err)
	}
	if _, err := svc.UpdateTaskItem(ctx, task.ID, 5, true); !errors.Is(err, ErrInvalid) {
		t.Fatalf("bad checkbox index: expected ErrInvalid, got %v", err)
	}
}

func TestAddTaskDependency_AlreadyPresentAndMissingTask(t *testing.T) {
	svc, n, _ := newTestService(t)
	ctx := t.Context()
	a, _ := svc.CreateTask(ctx, NewTask{Title: "A", Summary: "a"})
	b, _ := svc.CreateTask(ctx, NewTask{Title: "B", Summary: "b", Dependencies: []string{a.ID}})

	before := len(n.Events())
	_, added, err := svc.AddTaskDependency(ctx, b.ID, a.ID)
	if err != nil || added {
		t.Fatalf("duplicate dependency: added=%v err=%v, want false/nil", added, err)
	}
	if len(n.Events()) != before {
		t.Fatal("event emitted for no-op dependency add")
	}
	if _, _, err := svc.AddTaskDependency(ctx, "missing", a.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing task: expected ErrNotFound, got %v", err)
	}
}

func TestRemoveTaskDependency(t *testing.T) {
	svc, n, _ := newTestService(t)
	ctx := t.Context()
	a, _ := svc.CreateTask(ctx, NewTask{Title: "A", Summary: "a"})
	c, _ := svc.CreateTask(ctx, NewTask{Title: "C", Summary: "c"})
	b, err := svc.CreateTask(ctx, NewTask{Title: "B", Summary: "b", Dependencies: []string{a.ID, c.ID}})
	if err != nil {
		t.Fatal(err)
	}

	got, err := svc.RemoveTaskDependency(ctx, b.ID, a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got.Dependencies, []string{c.ID}) {
		t.Fatalf("dependencies = %v, want [%s]", got.Dependencies, c.ID)
	}
	assertLastEvent(t, n, "task.updated")
	assertFileContains(t, b.FilePath, "dependencies = ['"+c.ID+"']")
	assertFileNotContains(t, b.FilePath, a.ID)

	tests := []struct {
		name    string
		id, dep string
		wantErr error
	}{
		{"dependency not present", b.ID, a.ID, ErrNotFound},
		{"unknown dependency", b.ID, "nope", ErrNotFound},
		{"unknown task", "missing", a.ID, ErrNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := len(n.Events())
			if _, err := svc.RemoveTaskDependency(ctx, tt.id, tt.dep); !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
			if len(n.Events()) != before {
				t.Fatal("event emitted for rejected removal")
			}
		})
	}
}

func TestCompleteTask(t *testing.T) {
	svc, n, _ := newTestService(t)
	ctx := t.Context()

	a, _ := svc.CreateTask(ctx, NewTask{Title: "A", Summary: "a", Body: "- [x] done item"})
	b, _ := svc.CreateTask(ctx, NewTask{Title: "B", Summary: "b", Dependencies: []string{a.ID}})

	// B is blocked by A.
	_, _, err := svc.CompleteTask(ctx, CompleteTaskInput{ID: b.ID, Summary: "s", WhatDone: "w", WhyDone: "y"})
	if !errors.Is(err, ErrConflict) || !strings.Contains(err.Error(), a.ID) {
		t.Fatalf("blocked completion: expected ErrConflict naming %s, got %v", a.ID, err)
	}

	done, unblocked, err := svc.CompleteTask(ctx, CompleteTaskInput{
		ID: a.ID, Summary: " Shipped it ", WhatDone: "Implemented A", WhyDone: "Needed by B",
	})
	if err != nil {
		t.Fatal(err)
	}
	if done.Status != "done" || done.Summary != "Shipped it" {
		t.Fatalf("completed task = status %q summary %q", done.Status, done.Summary)
	}
	if len(unblocked) != 1 || unblocked[0].ID != b.ID {
		t.Fatalf("unblocked = %+v, want [%s]", unblocked, b.ID)
	}
	assertLastEvent(t, n, "task.updated")
	assertFileContains(t, a.FilePath, "status = 'done'", "summary = 'Shipped it'",
		"## Completion Summary", "Implemented A", "Needed by B")

	t.Run("already done is idempotent", func(t *testing.T) {
		again, _, err := svc.CompleteTask(ctx, CompleteTaskInput{ID: a.ID, Summary: "Again", WhatDone: "Redo", WhyDone: "Why"})
		if err != nil {
			t.Fatal(err)
		}
		if strings.Count(again.Body, "## Completion Summary") != 1 {
			t.Fatalf("expected exactly one completion summary, got body %q", again.Body)
		}
		assertFileContains(t, a.FilePath, "Redo")
		assertFileNotContains(t, a.FilePath, "Implemented A")
	})

	t.Run("ignore dependencies", func(t *testing.T) {
		c, _ := svc.CreateTask(ctx, NewTask{Title: "C", Summary: "c"})
		d, _ := svc.CreateTask(ctx, NewTask{Title: "D", Summary: "d", Dependencies: []string{c.ID}})
		got, _, err := svc.CompleteTask(ctx, CompleteTaskInput{ID: d.ID, Summary: "s", WhatDone: "w", WhyDone: "y", IgnoreDependencies: true})
		if err != nil || got.Status != "done" {
			t.Fatalf("ignore_dependencies: status %q err %v", got.Status, err)
		}
	})

	t.Run("unchecked items conflict", func(t *testing.T) {
		e, _ := svc.CreateTask(ctx, NewTask{Title: "E", Summary: "e", Body: "- [ ] open item"})
		before := len(n.Events())
		if _, _, err := svc.CompleteTask(ctx, CompleteTaskInput{ID: e.ID, Summary: "s", WhatDone: "w", WhyDone: "y"}); !errors.Is(err, ErrConflict) {
			t.Fatalf("expected ErrConflict, got %v", err)
		}
		if len(n.Events()) != before {
			t.Fatal("event emitted for rejected completion")
		}
		assertFileNotContains(t, e.FilePath, "status = 'done'")
	})

	validation := []struct {
		name string
		in   CompleteTaskInput
		want error
	}{
		{"missing summary", CompleteTaskInput{ID: a.ID, WhatDone: "w", WhyDone: "y"}, ErrInvalid},
		{"missing what", CompleteTaskInput{ID: a.ID, Summary: "s", WhyDone: "y"}, ErrInvalid},
		{"missing why", CompleteTaskInput{ID: a.ID, Summary: "s", WhatDone: "w"}, ErrInvalid},
		{"unknown task", CompleteTaskInput{ID: "missing", Summary: "s", WhatDone: "w", WhyDone: "y"}, ErrNotFound},
	}
	for _, tt := range validation {
		t.Run(tt.name, func(t *testing.T) {
			if _, _, err := svc.CompleteTask(ctx, tt.in); !errors.Is(err, tt.want) {
				t.Fatalf("expected %v, got %v", tt.want, err)
			}
		})
	}
}

func TestDeleteTask_ForceAndNotFound(t *testing.T) {
	svc, n, _ := newTestService(t)
	ctx := t.Context()
	a, _ := svc.CreateTask(ctx, NewTask{Title: "A", Summary: "a"})
	_, _ = svc.CreateTask(ctx, NewTask{Title: "B", Summary: "b", Dependencies: []string{a.ID}})

	if err := svc.DeleteTask(ctx, "missing", false); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
	if err := svc.DeleteTask(ctx, a.ID, true); err != nil {
		t.Fatal(err)
	}
	assertLastEvent(t, n, "task.deleted")
	if _, err := os.Stat(a.FilePath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("task file still exists: %v", err)
	}
}
