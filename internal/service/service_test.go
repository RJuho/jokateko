package service

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/RJuho/jokateko/internal/config"
	"github.com/RJuho/jokateko/internal/model"
	"github.com/RJuho/jokateko/internal/store"
	"github.com/RJuho/jokateko/internal/writer"
)

type recordingNotifier struct {
	mu     sync.Mutex
	events []string
}

func (n *recordingNotifier) Broadcast(eventType string, _ any) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.events = append(n.events, eventType)
}

func (n *recordingNotifier) Events() []string {
	n.mu.Lock()
	defer n.mu.Unlock()
	return append([]string(nil), n.events...)
}

func newTestService(t *testing.T) (*Service, *recordingNotifier, string) {
	t.Helper()
	dir := t.TempDir()
	st, err := store.OpenMemory()
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	n := &recordingNotifier{}
	svc := New(config.Default(dir), dir, st, writer.New(writer.NewSuppressionCache(time.Second)), n)
	svc.now = func() time.Time { return time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC) }
	return svc, n, dir
}

func TestSlugify(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"Hello World", "hello-world"},
		{"  Spaces  and__underscores  ", "spaces-and-underscores"},
		{"Ünïcödé & symbols!", "ncd-symbols"},
		{"!!!", "fallback"},
		{"a very long title that will certainly exceed the forty character limit", "a-very-long-title-that-will-certainly-ex"},
		// Truncation must not leave a trailing hyphen.
		{"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa bbb", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
	}
	for _, tt := range tests {
		if got := Slugify(tt.in, "fallback"); got != tt.want {
			t.Errorf("Slugify(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestValidateID(t *testing.T) {
	valid := []string{"a", "260901-setup-db", "zero-cgo", "x1"}
	for _, id := range valid {
		if err := ValidateID(id); err != nil {
			t.Errorf("ValidateID(%q) = %v, want nil", id, err)
		}
	}
	invalid := []string{"", "../escape", "a/b", `a\b`, "..", ".hidden", "-leading", "UPPER", "with space", "a.md"}
	for _, id := range invalid {
		if err := ValidateID(id); !errors.Is(err, ErrInvalid) {
			t.Errorf("ValidateID(%q) = %v, want ErrInvalid", id, err)
		}
	}
}

func TestCreateTask_RejectsPathTraversal(t *testing.T) {
	svc, _, dir := newTestService(t)
	_, err := svc.CreateTask(t.Context(), NewTask{ID: "../../evil", Title: "Evil", Summary: "s"})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("expected ErrInvalid, got %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(dir, "evil.md")); statErr == nil {
		t.Fatal("file was written outside the tasks directory")
	}
}

func TestCreateTask_IDCollisions(t *testing.T) {
	svc, _, _ := newTestService(t)
	ctx := t.Context()

	first, err := svc.CreateTask(ctx, NewTask{Title: "Same Title", Summary: "one"})
	if err != nil {
		t.Fatalf("first create: %v", err)
	}
	second, err := svc.CreateTask(ctx, NewTask{Title: "Same Title", Summary: "two"})
	if err != nil {
		t.Fatalf("second create: %v", err)
	}
	if first.ID != "261004-same-title" || second.ID != "261004-same-title-2" {
		t.Fatalf("unexpected generated IDs %q, %q", first.ID, second.ID)
	}

	got, err := svc.GetTask(ctx, first.ID)
	if err != nil || got.Summary != "one" {
		t.Fatalf("first task was overwritten: %+v, %v", got, err)
	}

	if _, err := svc.CreateTask(ctx, NewTask{ID: first.ID, Title: "Explicit", Summary: "x"}); !errors.Is(err, ErrConflict) {
		t.Fatalf("explicit duplicate ID: expected ErrConflict, got %v", err)
	}
}

func TestCreateTask_ValidatesStatus(t *testing.T) {
	svc, _, _ := newTestService(t)
	if _, err := svc.CreateTask(t.Context(), NewTask{Title: "T", Summary: "s", Status: "nope"}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("expected ErrInvalid for unknown column, got %v", err)
	}
}

func TestMutations_Notify(t *testing.T) {
	svc, n, _ := newTestService(t)
	ctx := t.Context()

	a, err := svc.CreateTask(ctx, NewTask{Title: "A", Summary: "a", Body: "- [ ] item"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := svc.CreateTask(ctx, NewTask{Title: "B", Summary: "b"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.UpdateTaskStatus(ctx, a.ID, "ready"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.UpdateTaskItem(ctx, a.ID, 1, true); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.AddTaskDependency(ctx, b.ID, a.ID); err != nil {
		t.Fatal(err)
	}
	if err := svc.DeleteTask(ctx, a.ID, false); !errors.Is(err, ErrConflict) {
		t.Fatalf("expected ErrConflict deleting a task with dependents, got %v", err)
	}
	if err := svc.DeleteTask(ctx, b.ID, false); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateStrategy(ctx, NewStrategy{Title: "Zero CGO"}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateGlossaryTerm(ctx, NewGlossaryTerm{Title: "Term"}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateMilestone(ctx, NewMilestone{Title: "MVP"}); err != nil {
		t.Fatal(err)
	}
	if err := svc.DeleteStrategy(ctx, "zero-cgo"); err != nil {
		t.Fatal(err)
	}

	want := []string{
		"task.created", "task.created", "task.updated", "task.updated", "task.updated",
		"task.deleted", "strategy.created", "glossary.created", "milestone.created", "strategy.deleted",
	}
	got := n.Events()
	if len(got) != len(want) {
		t.Fatalf("events = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("events = %v, want %v", got, want)
		}
	}
}

func TestAddTaskDependency_RejectsCycle(t *testing.T) {
	svc, _, _ := newTestService(t)
	ctx := t.Context()

	a, _ := svc.CreateTask(ctx, NewTask{Title: "A", Summary: "a"})
	b, _ := svc.CreateTask(ctx, NewTask{Title: "B", Summary: "b", Dependencies: []string{a.ID}})

	if _, _, err := svc.AddTaskDependency(ctx, a.ID, b.ID); !errors.Is(err, ErrConflict) {
		t.Fatalf("expected ErrConflict for cycle, got %v", err)
	}
	if _, _, err := svc.AddTaskDependency(ctx, a.ID, a.ID); !errors.Is(err, ErrInvalid) {
		t.Fatalf("expected ErrInvalid for self dependency, got %v", err)
	}
	if _, _, err := svc.AddTaskDependency(ctx, a.ID, "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound for missing dependency, got %v", err)
	}
}

func TestUpdateTask_ErrorWritesNothing(t *testing.T) {
	svc, n, _ := newTestService(t)
	ctx := t.Context()

	task, _ := svc.CreateTask(ctx, NewTask{Title: "A", Summary: "a"})
	before, _ := os.ReadFile(task.FilePath)

	_, err := svc.UpdateTask(ctx, task.ID, func(t *model.Task) error {
		t.Title = "changed"
		return invalidf("nope")
	})
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("expected ErrInvalid, got %v", err)
	}
	after, _ := os.ReadFile(task.FilePath)
	if string(before) != string(after) {
		t.Fatal("file changed despite update error")
	}
	if got := n.Events(); len(got) != 1 {
		t.Fatalf("expected only the create event, got %v", got)
	}
}

func TestCheckBodyEdit(t *testing.T) {
	svc, _, _ := newTestService(t)
	if err := svc.CheckBodyEdit("backlog", "a", "b", "hint"); err != nil {
		t.Errorf("backlog body edit should be allowed: %v", err)
	}
	if err := svc.CheckBodyEdit("ready", "- [ ] x", "- [x] x", "hint"); err != nil {
		t.Errorf("checkbox toggle should be allowed: %v", err)
	}
	if err := svc.CheckBodyEdit("ready", "a", "b", "hint"); !errors.Is(err, ErrConflict) {
		t.Errorf("body edit outside editable states: expected ErrConflict, got %v", err)
	}
}
