package service

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RJuho/jokateko/internal/config"
	"github.com/RJuho/jokateko/internal/model"
	"github.com/RJuho/jokateko/internal/store"
)

func TestServiceAccessors(t *testing.T) {
	svc, _, dir := newTestService(t)
	if svc.Config() != svc.cfg {
		t.Error("Config() did not return the active config")
	}
	if svc.WorkspaceDir() != dir {
		t.Errorf("WorkspaceDir() = %q, want %q", svc.WorkspaceDir(), dir)
	}
	if want := filepath.Join(dir, ".jokateko", "tasks"); svc.Dirs().Tasks != want {
		t.Errorf("Dirs().Tasks = %q, want %q", svc.Dirs().Tasks, want)
	}
	if svc.Store() != svc.store {
		t.Error("Store() did not return the backing store")
	}
}

func TestNew_NilConfigAndNotifier(t *testing.T) {
	dir := t.TempDir()
	st, err := store.OpenMemory()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })

	svc := New(nil, dir, st, nil, nil)
	if svc.Config() == nil {
		t.Fatal("expected default config when nil is passed")
	}
	if svc.Config().Project.Name != filepath.Base(dir) {
		t.Errorf("project name = %q, want %q", svc.Config().Project.Name, filepath.Base(dir))
	}
	// A nil notifier must not panic.
	svc.notify("task.updated", nil)
}

func TestKindError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		kind error
		msg  string
	}{
		{"invalid", invalidf("bad %s", "x"), ErrInvalid, "bad x"},
		{"conflict", conflictf("clash %d", 1), ErrConflict, "clash 1"},
		{"not found", notFoundf("gone %q", "y"), ErrNotFound, `gone "y"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.msg {
				t.Errorf("Error() = %q, want %q", tt.err.Error(), tt.msg)
			}
			if !errors.Is(tt.err, tt.kind) {
				t.Errorf("errors.Is(%v, %v) = false", tt.err, tt.kind)
			}
		})
	}
}

func TestCheckTags(t *testing.T) {
	tests := []struct {
		name    string
		enforce bool
		allowed []string
		tags    []string
		wantErr bool
	}{
		{"not enforced accepts anything", false, []string{"backend"}, []string{"random"}, false},
		{"enforced allowed tag", true, []string{"backend", "ui"}, []string{"ui"}, false},
		{"enforced rejects unknown", true, []string{"backend"}, []string{"backend", "random"}, true},
		{"enforced empty list", true, nil, nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _, _ := newTestService(t)
			svc.cfg.Tags = config.TagsConfig{Allowed: tt.allowed, EnforceAllowed: tt.enforce}
			err := svc.CheckTags(tt.tags)
			if tt.wantErr {
				if !errors.Is(err, ErrInvalid) || !strings.Contains(err.Error(), "random") {
					t.Fatalf("expected ErrInvalid naming the tag, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error %v", err)
			}
		})
	}
}

func TestAllocateID_ExistingFileOnDisk(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "taken.md"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	none := func(string) bool { return false }
	if _, err := allocateID(dir, "taken", "", none); !errors.Is(err, ErrConflict) {
		t.Fatalf("explicit on-disk ID: expected ErrConflict, got %v", err)
	}
	id, err := allocateID(dir, "", "taken", none)
	if err != nil || id != "taken-2" {
		t.Fatalf("generated ID = %q, %v; want taken-2", id, err)
	}
	if _, err := allocateID(dir, "", "BAD ID", none); !errors.Is(err, ErrInvalid) {
		t.Fatalf("invalid generated ID: expected ErrInvalid, got %v", err)
	}
	all := func(string) bool { return true }
	if _, err := allocateID(dir, "", "full", all); !errors.Is(err, ErrConflict) {
		t.Fatalf("exhausted IDs: expected ErrConflict, got %v", err)
	}
}

func TestParseMilestoneStatus(t *testing.T) {
	tests := []struct {
		raw     string
		want    model.MilestoneStatus
		wantErr bool
	}{
		{"open", model.MilestoneStatusOpen, false},
		{"closed", model.MilestoneStatusClosed, false},
		{"  CLOSED ", model.MilestoneStatusClosed, false},
		{"Open", model.MilestoneStatusOpen, false},
		{"", "", true},
		{"archived", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			got, err := ParseMilestoneStatus(tt.raw)
			if tt.wantErr {
				if !errors.Is(err, ErrInvalid) {
					t.Fatalf("expected ErrInvalid, got %v", err)
				}
				return
			}
			if err != nil || got != tt.want {
				t.Fatalf("ParseMilestoneStatus(%q) = %q, %v; want %q", tt.raw, got, err, tt.want)
			}
		})
	}
}

func TestCreateMilestone_Validation(t *testing.T) {
	svc, _, _ := newTestService(t)
	ctx := t.Context()
	if _, err := svc.CreateMilestone(ctx, NewMilestone{}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("missing title: expected ErrInvalid, got %v", err)
	}
	if _, err := svc.CreateMilestone(ctx, NewMilestone{Title: "M", Status: "weird"}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("bad status: expected ErrInvalid, got %v", err)
	}
	if _, err := svc.CreateMilestone(ctx, NewMilestone{ID: "Bad ID", Title: "M"}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("bad id: expected ErrInvalid, got %v", err)
	}
}

func TestMilestoneLifecycle(t *testing.T) {
	svc, n, _ := newTestService(t)
	ctx := t.Context()

	ms, err := svc.CreateMilestone(ctx, NewMilestone{Title: "MVP", TargetDate: "2026-12-01", Summary: "first"})
	if err != nil {
		t.Fatal(err)
	}
	assertLastEvent(t, n, "milestone.created")
	assertFileContains(t, ms.FilePath, "title = 'MVP'", "status = 'open'")

	got, err := svc.GetMilestone(ctx, ms.ID)
	if err != nil || got.Title != "MVP" {
		t.Fatalf("GetMilestone = %+v, %v", got, err)
	}
	if _, err := svc.GetMilestone(ctx, "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	updated, err := svc.UpdateMilestone(ctx, ms.ID, func(m *model.Milestone) error {
		m.Status = model.MilestoneStatusClosed
		m.Summary = "done"
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != model.MilestoneStatusClosed {
		t.Fatalf("status = %q, want closed", updated.Status)
	}
	assertLastEvent(t, n, "milestone.updated")
	assertFileContains(t, ms.FilePath, "status = 'closed'", "summary = 'done'")

	t.Run("update errors", func(t *testing.T) {
		before := len(n.Events())
		if _, err := svc.UpdateMilestone(ctx, "missing", func(*model.Milestone) error { return nil }); !errors.Is(err, ErrNotFound) {
			t.Fatalf("expected ErrNotFound, got %v", err)
		}
		if _, err := svc.UpdateMilestone(ctx, ms.ID, func(*model.Milestone) error { return invalidf("no") }); !errors.Is(err, ErrInvalid) {
			t.Fatalf("expected ErrInvalid, got %v", err)
		}
		if len(n.Events()) != before {
			t.Fatal("event emitted for rejected update")
		}
	})

	// Assign a task: deletion must be refused unless forced.
	task, err := svc.CreateTask(ctx, NewTask{Title: "T", Summary: "s", Milestone: ms.ID})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.DeleteMilestone(ctx, ms.ID, false); !errors.Is(err, ErrConflict) {
		t.Fatalf("delete with assigned task: expected ErrConflict, got %v", err)
	}
	if _, err := os.Stat(ms.FilePath); err != nil {
		t.Fatalf("milestone file removed despite refusal: %v", err)
	}
	if err := svc.DeleteTask(ctx, task.ID, true); err != nil {
		t.Fatal(err)
	}
	if err := svc.DeleteMilestone(ctx, ms.ID, false); err != nil {
		t.Fatal(err)
	}
	assertLastEvent(t, n, "milestone.deleted")
	if _, err := os.Stat(ms.FilePath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("milestone file still exists: %v", err)
	}
	if err := svc.DeleteMilestone(ctx, ms.ID, false); !errors.Is(err, ErrNotFound) {
		t.Fatalf("delete missing: expected ErrNotFound, got %v", err)
	}
}

func TestDeleteMilestone_Force(t *testing.T) {
	svc, n, _ := newTestService(t)
	ctx := t.Context()
	ms, _ := svc.CreateMilestone(ctx, NewMilestone{Title: "M"})
	if _, err := svc.CreateTask(ctx, NewTask{Title: "T", Summary: "s", Milestone: ms.ID}); err != nil {
		t.Fatal(err)
	}
	if err := svc.DeleteMilestone(ctx, ms.ID, true); err != nil {
		t.Fatal(err)
	}
	assertLastEvent(t, n, "milestone.deleted")
	if _, err := os.Stat(ms.FilePath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("milestone file still exists: %v", err)
	}
}

func TestStrategyLifecycle(t *testing.T) {
	svc, n, _ := newTestService(t)
	ctx := t.Context()

	if _, err := svc.CreateStrategy(ctx, NewStrategy{}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("missing title: expected ErrInvalid, got %v", err)
	}
	if _, err := svc.CreateStrategy(ctx, NewStrategy{Title: "X", Tier: 9}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("bad tier: expected ErrInvalid, got %v", err)
	}
	if _, err := svc.CreateStrategy(ctx, NewStrategy{ID: "Bad ID", Title: "X"}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("bad id: expected ErrInvalid, got %v", err)
	}

	st, err := svc.CreateStrategy(ctx, NewStrategy{Title: "Zero CGO", Tier: model.TierDomain})
	if err != nil {
		t.Fatal(err)
	}
	assertLastEvent(t, n, "strategy.created")

	updated, err := svc.UpdateStrategy(ctx, st.ID, func(s *model.Strategy) error {
		s.Summary = "No C dependencies"
		s.Body = "Pure Go only."
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Summary != "No C dependencies" || !strings.Contains(updated.BodyHTML, "Pure Go only.") {
		t.Fatalf("unexpected updated strategy %+v", updated)
	}
	assertLastEvent(t, n, "strategy.updated")
	assertFileContains(t, st.FilePath, "summary = 'No C dependencies'", "Pure Go only.")

	before := len(n.Events())
	if _, err := svc.UpdateStrategy(ctx, "missing", func(*model.Strategy) error { return nil }); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
	if _, err := svc.UpdateStrategy(ctx, st.ID, func(*model.Strategy) error { return conflictf("no") }); !errors.Is(err, ErrConflict) {
		t.Fatalf("expected ErrConflict, got %v", err)
	}
	if len(n.Events()) != before {
		t.Fatal("event emitted for rejected update")
	}
	if err := svc.DeleteStrategy(ctx, "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestGlossaryLifecycle(t *testing.T) {
	svc, n, _ := newTestService(t)
	ctx := t.Context()

	if _, err := svc.CreateGlossaryTerm(ctx, NewGlossaryTerm{}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("missing title: expected ErrInvalid, got %v", err)
	}
	if _, err := svc.CreateGlossaryTerm(ctx, NewGlossaryTerm{ID: "Bad ID", Title: "X"}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("bad id: expected ErrInvalid, got %v", err)
	}

	term, err := svc.CreateGlossaryTerm(ctx, NewGlossaryTerm{Title: "Spec First", Summary: "write specs first"})
	if err != nil {
		t.Fatal(err)
	}
	assertLastEvent(t, n, "glossary.created")
	assertFileContains(t, term.FilePath, "title = 'Spec First'")

	got, err := svc.GetGlossaryTerm(ctx, term.ID)
	if err != nil || got.Title != "Spec First" {
		t.Fatalf("GetGlossaryTerm = %+v, %v", got, err)
	}
	if _, err := svc.GetGlossaryTerm(ctx, "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	updated, err := svc.UpdateGlossaryTerm(ctx, term.ID, func(g *model.GlossaryTerm) error {
		g.Summary = "specs before code"
		return nil
	})
	if err != nil || updated.Summary != "specs before code" {
		t.Fatalf("UpdateGlossaryTerm = %+v, %v", updated, err)
	}
	assertLastEvent(t, n, "glossary.updated")
	assertFileContains(t, term.FilePath, "summary = 'specs before code'")

	before := len(n.Events())
	if _, err := svc.UpdateGlossaryTerm(ctx, "missing", func(*model.GlossaryTerm) error { return nil }); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
	if _, err := svc.UpdateGlossaryTerm(ctx, term.ID, func(*model.GlossaryTerm) error { return invalidf("no") }); !errors.Is(err, ErrInvalid) {
		t.Fatalf("expected ErrInvalid, got %v", err)
	}
	if len(n.Events()) != before {
		t.Fatal("event emitted for rejected update")
	}

	if err := svc.DeleteGlossaryTerm(ctx, term.ID); err != nil {
		t.Fatal(err)
	}
	assertLastEvent(t, n, "glossary.deleted")
	if _, err := os.Stat(term.FilePath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("glossary file still exists: %v", err)
	}
	if err := svc.DeleteGlossaryTerm(ctx, term.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("delete missing: expected ErrNotFound, got %v", err)
	}
}

func TestWrapGet(t *testing.T) {
	boom := errors.New("boom")
	tests := []struct {
		name     string
		err      error
		wantKind error
		wantMsg  string
	}{
		{"ok", nil, nil, ""},
		{"not found", store.ErrNotFound, ErrNotFound, `widget "w1" not found`},
		{"other", boom, boom, `failed to get widget "w1": boom`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := wrapGet(42, tt.err, "widget", "w1")
			if tt.err == nil {
				if err != nil || got != 42 {
					t.Fatalf("wrapGet = %d, %v; want 42, nil", got, err)
				}
				return
			}
			if got != 0 {
				t.Errorf("expected zero value, got %d", got)
			}
			if !errors.Is(err, tt.wantKind) || err.Error() != tt.wantMsg {
				t.Fatalf("wrapGet error = %v, want kind %v msg %q", err, tt.wantKind, tt.wantMsg)
			}
		})
	}
}

// blockDir replaces dir with a regular file so that writes into it fail.
func blockDir(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(dir), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(dir); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dir, []byte("blocker"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestWriteEntity_FilesystemErrors(t *testing.T) {
	tests := []struct {
		name   string
		dir    func(d config.Dirs) string
		create func(svc *Service) error
		kind   string
	}{
		{"milestone", func(d config.Dirs) string { return d.Milestones }, func(svc *Service) error {
			_, err := svc.CreateMilestone(t.Context(), NewMilestone{Title: "M"})
			return err
		}, "milestone"},
		{"strategy", func(d config.Dirs) string { return d.Strategies }, func(svc *Service) error {
			_, err := svc.CreateStrategy(t.Context(), NewStrategy{Title: "S"})
			return err
		}, "strategy"},
		{"glossary", func(d config.Dirs) string { return d.Glossary }, func(svc *Service) error {
			_, err := svc.CreateGlossaryTerm(t.Context(), NewGlossaryTerm{Title: "G"})
			return err
		}, "glossary term"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, n, _ := newTestService(t)
			blockDir(t, tt.dir(svc.Dirs()))
			err := tt.create(svc)
			if err == nil || !strings.Contains(err.Error(), "failed to save "+tt.kind+" file") {
				t.Fatalf("expected save error, got %v", err)
			}
			if got := n.Events(); len(got) != 0 {
				t.Fatalf("expected no events, got %v", got)
			}
		})
	}

	t.Run("format error", func(t *testing.T) {
		svc, _, dir := newTestService(t)
		err := svc.writeEntity("thing", filepath.Join(dir, "x.md"), func() {}, "")
		if err == nil || !strings.Contains(err.Error(), "failed to format thing markdown") {
			t.Fatalf("expected format error, got %v", err)
		}
	})
}

func TestRemoveEntityFile(t *testing.T) {
	svc, _, dir := newTestService(t)

	if err := svc.removeEntityFile("thing", ""); err != nil {
		t.Fatalf("empty path: %v", err)
	}
	if err := svc.removeEntityFile("thing", filepath.Join(dir, "absent.md")); err != nil {
		t.Fatalf("missing file should be ignored: %v", err)
	}

	// A non-empty directory cannot be removed with os.Remove.
	busy := filepath.Join(dir, "busy.md")
	if err := os.MkdirAll(filepath.Join(busy, "child"), 0o755); err != nil {
		t.Fatal(err)
	}
	err := svc.removeEntityFile("thing", busy)
	if err == nil || !strings.Contains(err.Error(), "failed to remove thing file") {
		t.Fatalf("expected remove error, got %v", err)
	}
}

func TestDeleteEntities_RemoveFileError(t *testing.T) {
	svc, n, _ := newTestService(t)
	ctx := t.Context()

	ms, _ := svc.CreateMilestone(ctx, NewMilestone{Title: "M"})
	st, _ := svc.CreateStrategy(ctx, NewStrategy{Title: "S"})
	term, _ := svc.CreateGlossaryTerm(ctx, NewGlossaryTerm{Title: "G"})
	task, _ := svc.CreateTask(ctx, NewTask{Title: "T", Summary: "s"})

	tests := []struct {
		name string
		path string
		del  func() error
	}{
		{"milestone", ms.FilePath, func() error { return svc.DeleteMilestone(ctx, ms.ID, true) }},
		{"strategy", st.FilePath, func() error { return svc.DeleteStrategy(ctx, st.ID) }},
		{"glossary", term.FilePath, func() error { return svc.DeleteGlossaryTerm(ctx, term.ID) }},
		{"task", task.FilePath, func() error { return svc.DeleteTask(ctx, task.ID, true) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Replace the entity file with a non-empty directory so removal fails.
			if err := os.Remove(tt.path); err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(filepath.Join(tt.path, "child"), 0o755); err != nil {
				t.Fatal(err)
			}
			before := len(n.Events())
			if err := tt.del(); err == nil || !strings.Contains(err.Error(), "failed to remove") {
				t.Fatalf("expected remove error, got %v", err)
			}
			if len(n.Events()) != before {
				t.Fatal("event emitted despite failed delete")
			}
		})
	}
}

func TestStoreClosedErrors(t *testing.T) {
	svc, _, _ := newTestService(t)
	ctx := t.Context()
	_ = svc.Store().Close()

	tests := []struct {
		name string
		call func() error
	}{
		{"get milestone", func() error { _, err := svc.GetMilestone(ctx, "m"); return err }},
		{"get strategy", func() error { _, err := svc.GetStrategy(ctx, "s"); return err }},
		{"get glossary", func() error { _, err := svc.GetGlossaryTerm(ctx, "g"); return err }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.call()
			if err == nil || errors.Is(err, ErrNotFound) || !strings.Contains(err.Error(), "failed to get") {
				t.Fatalf("expected wrapped store error, got %v", err)
			}
		})
	}
}
