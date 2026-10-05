package watcher_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/RJuho/jokateko/internal/config"
	"github.com/RJuho/jokateko/internal/store"
	"github.com/RJuho/jokateko/internal/watcher"
	"github.com/RJuho/jokateko/internal/writer"
)

const waitTimeout = 5 * time.Second

// eventually polls cond until it returns true or the deadline passes.
func eventually(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(waitTimeout)
	tick := time.NewTicker(5 * time.Millisecond)
	defer tick.Stop()
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		<-tick.C
	}
}

// recorder collects ingest events and log lines from concurrent goroutines.
type recorder struct {
	mu     sync.Mutex
	events []watcher.IngestEvent
	logs   []string
}

func (r *recorder) onChange(e watcher.IngestEvent) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, e)
}

func (r *recorder) logf(format string, args ...any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.logs = append(r.logs, fmt.Sprintf(format, args...))
}

func (r *recorder) has(e watcher.IngestEvent) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Contains(r.events, e)
}

func (r *recorder) logContains(substr string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.ContainsFunc(r.logs, func(l string) bool { return strings.Contains(l, substr) })
}

func testDirs(root string) config.Dirs {
	return config.Dirs{
		Tasks:      filepath.Join(root, "tasks"),
		Milestones: filepath.Join(root, "milestones"),
		Strategies: filepath.Join(root, "strategies"),
		Glossary:   filepath.Join(root, "glossary"),
	}
}

func openStore(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.OpenMemory()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

// writeAtomic writes data to path via an ignored temp file and a rename,
// so the watcher never observes a partially written file.
func writeAtomic(t *testing.T, path, data string) {
	t.Helper()
	tmp := filepath.Join(filepath.Dir(path), "."+filepath.Base(path)+".tmp")
	if err := os.WriteFile(tmp, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(tmp, path); err != nil {
		t.Fatal(err)
	}
}

func taskMarkdown(title string) string {
	return fmt.Sprintf("+++\ntitle = %q\nstatus = \"backlog\"\nsummary = \"s\"\n+++\n\nBody\n", title)
}

func TestStartPipeline_EndToEnd(t *testing.T) {
	root := t.TempDir()
	dirs := testDirs(root)
	if err := os.MkdirAll(dirs.Tasks, 0o755); err != nil {
		t.Fatal(err)
	}
	// Pre-existing file is loaded by the initial scan.
	if err := os.WriteFile(filepath.Join(dirs.Tasks, "existing.md"), []byte(taskMarkdown("Existing")), 0o644); err != nil {
		t.Fatal(err)
	}

	st := openStore(t)
	rec := &recorder{}
	baseline := runtime.NumGoroutine()

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	stop, err := watcher.StartPipeline(ctx, st, writer.NewSuppressionCache(time.Second), dirs, rec.onChange, rec.logf)
	if err != nil {
		t.Fatalf("StartPipeline: %v", err)
	}

	if _, err := st.GetTask(ctx, "existing"); err != nil {
		t.Fatalf("initial scan did not load existing task: %v", err)
	}
	if !rec.has(watcher.IngestEvent{EntityType: "task", EntityID: "existing", Op: watcher.OpWrite}) {
		t.Fatal("initial scan did not broadcast the existing task")
	}

	// External write: ingested into the store and broadcast.
	newPath := filepath.Join(dirs.Tasks, "new-task.md")
	writeAtomic(t, newPath, taskMarkdown("Fresh"))
	eventually(t, "write ingest event", func() bool {
		return rec.has(watcher.IngestEvent{EntityType: "task", EntityID: "new-task", Op: watcher.OpWrite})
	})
	got, err := st.GetTask(ctx, "new-task")
	if err != nil || got.Title != "Fresh" {
		t.Fatalf("store task = %+v, %v; want title Fresh", got, err)
	}

	// External glossary write in another watched dir (created by the watcher).
	writeAtomic(t, filepath.Join(dirs.Glossary, "term.md"), "+++\ntitle = \"Term\"\nsummary = \"s\"\n+++\n")
	eventually(t, "glossary ingest event", func() bool {
		return rec.has(watcher.IngestEvent{EntityType: "glossary", EntityID: "term", Op: watcher.OpWrite})
	})

	// Malformed file: the handler error is logged, the pipeline keeps running.
	writeAtomic(t, filepath.Join(dirs.Tasks, "broken.md"), "+++\ntitle = \n+++\n")
	eventually(t, "error log for malformed file", func() bool { return rec.logContains("broken.md") })

	// External delete: removed from the store and broadcast.
	if err := os.Remove(newPath); err != nil {
		t.Fatal(err)
	}
	eventually(t, "delete ingest event", func() bool {
		return rec.has(watcher.IngestEvent{EntityType: "task", EntityID: "new-task", Op: watcher.OpDelete})
	})
	if _, err := st.GetTask(ctx, "new-task"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}

	// Clean shutdown: cancelling the context and calling stop (twice) ends all goroutines.
	cancel()
	stop()
	stop()
	eventually(t, "pipeline goroutines to exit", func() bool { return runtime.NumGoroutine() <= baseline })
}

func TestStartPipeline_WatcherFailure(t *testing.T) {
	root := t.TempDir()
	dirs := testDirs(root)
	// A regular file where the tasks directory should be: the initial scan logs
	// an error and the watcher cannot create the directory.
	if err := os.WriteFile(dirs.Tasks, []byte("blocker"), 0o644); err != nil {
		t.Fatal(err)
	}

	rec := &recorder{}
	stop, err := watcher.StartPipeline(t.Context(), openStore(t), nil, dirs, nil, rec.logf)
	if err == nil || !strings.Contains(err.Error(), "failed to start filesystem watcher") {
		t.Fatalf("expected watcher start error, got %v", err)
	}
	if stop == nil {
		t.Fatal("stop must never be nil")
	}
	stop()
	if !rec.logContains("errors during initial directory scan") {
		t.Fatalf("expected initial scan warning in logs, got %v", rec.logs)
	}
}

func TestNew_DirectoryHandling(t *testing.T) {
	t.Run("missing dir is created", func(t *testing.T) {
		missing := filepath.Join(t.TempDir(), "a", "b", "tasks")
		w, err := watcher.New([]string{"", missing}, 0)
		if err != nil {
			t.Fatalf("New: %v", err)
		}
		t.Cleanup(func() { _ = w.Close() })
		if fi, err := os.Stat(missing); err != nil || !fi.IsDir() {
			t.Fatalf("watch dir was not created: %v", err)
		}
	})
	t.Run("dir blocked by a file", func(t *testing.T) {
		blocker := filepath.Join(t.TempDir(), "file")
		if err := os.WriteFile(blocker, nil, 0o644); err != nil {
			t.Fatal(err)
		}
		_, err := watcher.New([]string{filepath.Join(blocker, "tasks")}, time.Millisecond)
		if err == nil || !strings.Contains(err.Error(), "failed to create watch directory") {
			t.Fatalf("expected create error, got %v", err)
		}
	})
	t.Run("path is a file", func(t *testing.T) {
		file := filepath.Join(t.TempDir(), "file")
		if err := os.WriteFile(file, nil, 0o644); err != nil {
			t.Fatal(err)
		}
		// MkdirAll fails on an existing non-directory path.
		if _, err := watcher.New([]string{file}, time.Millisecond); err == nil {
			t.Fatal("expected an error for a regular file path")
		}
	})
}

func TestWatcher_ChannelsCloseOnShutdown(t *testing.T) {
	tests := []struct {
		name     string
		shutdown func(w *watcher.Watcher, cancel context.CancelFunc)
	}{
		{"context cancel", func(_ *watcher.Watcher, cancel context.CancelFunc) { cancel() }},
		{"close", func(w *watcher.Watcher, _ context.CancelFunc) { _ = w.Close() }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, err := watcher.New([]string{t.TempDir()}, time.Millisecond)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = w.Close() })
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			w.Start(ctx)

			tt.shutdown(w, cancel)
			timeout := time.After(waitTimeout)
			for _, ch := range []string{"events", "errors"} {
				select {
				case <-drained(w, ch):
				case <-timeout:
					t.Fatalf("%s channel did not close", ch)
				}
			}
			if err := w.Close(); err != nil {
				t.Fatalf("second Close returned %v", err)
			}
		})
	}
}

// drained returns a channel that is closed once the named watcher channel
// ("events" or "errors") has been drained and closed.
func drained(w *watcher.Watcher, name string) <-chan struct{} {
	out := make(chan struct{})
	go func() {
		if name == "events" {
			for range w.Events() {
			}
		} else {
			for range w.Errors() {
			}
		}
		close(out)
	}()
	return out
}

func TestWatcher_DeleteAndRenameEvents(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.md")
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	w, err := watcher.New([]string{dir}, 10*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = w.Close() })
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	w.Start(ctx)

	// Non-entity extensions are ignored entirely.
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(path, filepath.Join(dir, "b.md")); err != nil {
		t.Fatal(err)
	}

	got := map[string]watcher.FileOp{}
	timeout := time.After(waitTimeout)
	for len(got) < 2 {
		select {
		case ev := <-w.Events():
			got[filepath.Base(ev.Path)] = ev.Op
		case <-timeout:
			t.Fatalf("timed out; events so far %v", got)
		}
	}
	if got["a.md"] != watcher.OpDelete || got["b.md"] != watcher.OpWrite {
		t.Fatalf("unexpected events %v", got)
	}
	if _, ok := got["notes.txt"]; ok {
		t.Fatal("non-markdown file produced an event")
	}
}

func TestHandleEvent_Branches(t *testing.T) {
	root := t.TempDir()
	dirs := testDirs(root)
	for _, d := range dirs.All() {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	malformed := "+++\ntitle = \n+++\n"

	tests := []struct {
		name    string
		setup   func(t *testing.T) string // returns event path
		op      watcher.FileOp
		wantErr string
		wantCB  bool
	}{
		{name: "non-markdown ignored", op: watcher.OpWrite, setup: func(t *testing.T) string {
			return filepath.Join(dirs.Tasks, "x.toml")
		}},
		{name: "outside entity dirs ignored", op: watcher.OpWrite, setup: func(t *testing.T) string {
			return filepath.Join(root, "other", "x.md")
		}},
		{name: "write of vanished file ignored", op: watcher.OpWrite, setup: func(t *testing.T) string {
			return filepath.Join(dirs.Tasks, "gone.md")
		}},
		{name: "read error", op: watcher.OpWrite, wantErr: "failed to read file", setup: func(t *testing.T) string {
			p := filepath.Join(dirs.Tasks, "dir.md")
			if err := os.MkdirAll(p, 0o755); err != nil {
				t.Fatal(err)
			}
			return p
		}},
		{name: "malformed milestone", op: watcher.OpWrite, wantErr: "failed to parse milestone", setup: func(t *testing.T) string {
			p := filepath.Join(dirs.Milestones, "bad.md")
			_ = os.WriteFile(p, []byte(malformed), 0o644)
			return p
		}},
		{name: "malformed strategy", op: watcher.OpWrite, wantErr: "failed to parse strategy", setup: func(t *testing.T) string {
			p := filepath.Join(dirs.Strategies, "bad.md")
			_ = os.WriteFile(p, []byte(malformed), 0o644)
			return p
		}},
		{name: "malformed glossary", op: watcher.OpWrite, wantErr: "failed to parse glossary term", setup: func(t *testing.T) string {
			p := filepath.Join(dirs.Glossary, "bad.md")
			_ = os.WriteFile(p, []byte(malformed), 0o644)
			return p
		}},
		{name: "delete milestone", op: watcher.OpDelete, wantCB: true, setup: func(t *testing.T) string {
			return filepath.Join(dirs.Milestones, "m.md")
		}},
		{name: "delete strategy", op: watcher.OpDelete, wantCB: true, setup: func(t *testing.T) string {
			return filepath.Join(dirs.Strategies, "s.md")
		}},
		{name: "delete glossary", op: watcher.OpDelete, wantCB: true, setup: func(t *testing.T) string {
			return filepath.Join(dirs.Glossary, "g.md")
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := watcher.NewPipeline(openStore(t), nil, dirs)
			called := false
			p.SetOnEntityChange(func(watcher.IngestEvent) { called = true })
			err := p.HandleEvent(t.Context(), watcher.FileEvent{Path: tt.setup(t), Op: tt.op})
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
				}
			} else if err != nil {
				t.Fatalf("unexpected error %v", err)
			}
			if called != tt.wantCB {
				t.Fatalf("callback called = %v, want %v", called, tt.wantCB)
			}
		})
	}
}

func TestHandleEvent_SuppressedDelete(t *testing.T) {
	root := t.TempDir()
	dirs := testDirs(root)
	cache := writer.NewSuppressionCache(time.Minute)
	path := filepath.Join(dirs.Tasks, "t.md")
	cache.RecordDelete(path)

	p := watcher.NewPipeline(openStore(t), cache, dirs)
	called := false
	p.SetOnEntityChange(func(watcher.IngestEvent) { called = true })
	if err := p.HandleEvent(t.Context(), watcher.FileEvent{Path: path, Op: watcher.OpDelete}); err != nil {
		t.Fatal(err)
	}
	if called {
		t.Fatal("suppressed delete must not trigger the callback")
	}
}

func TestHandleEvent_StoreErrors(t *testing.T) {
	root := t.TempDir()
	dirs := testDirs(root)
	files := map[string]string{
		filepath.Join(dirs.Tasks, "t.md"):      taskMarkdown("T"),
		filepath.Join(dirs.Milestones, "m.md"): "+++\ntitle = \"M\"\nstatus = \"open\"\nsummary = \"s\"\n+++\n",
		filepath.Join(dirs.Strategies, "s.md"): "+++\ntitle = \"S\"\ntier = 1\nsummary = \"s\"\n+++\n",
		filepath.Join(dirs.Glossary, "g.md"):   "+++\ntitle = \"G\"\nsummary = \"s\"\n+++\n",
	}
	for path, data := range files {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	st := openStore(t)
	_ = st.Close()
	p := watcher.NewPipeline(st, nil, dirs)

	tests := []struct {
		path    string
		op      watcher.FileOp
		wantErr string
	}{
		{filepath.Join(dirs.Tasks, "t.md"), watcher.OpWrite, "failed to upsert task"},
		{filepath.Join(dirs.Milestones, "m.md"), watcher.OpWrite, "failed to upsert milestone"},
		{filepath.Join(dirs.Strategies, "s.md"), watcher.OpWrite, "failed to upsert strategy"},
		{filepath.Join(dirs.Glossary, "g.md"), watcher.OpWrite, "failed to upsert glossary term"},
		{filepath.Join(dirs.Tasks, "t.md"), watcher.OpDelete, "failed to delete task"},
	}
	for _, tt := range tests {
		t.Run(tt.wantErr, func(t *testing.T) {
			err := p.HandleEvent(t.Context(), watcher.FileEvent{Path: tt.path, Op: tt.op})
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestProcessAll_SkipsAndReportsDirErrors(t *testing.T) {
	root := t.TempDir()
	dirs := testDirs(root)
	if err := os.MkdirAll(filepath.Join(dirs.Tasks, "subdir.md"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, data := range map[string]string{".hidden.md": "junk", "notes.txt": "junk", "ok.md": taskMarkdown("OK")} {
		if err := os.WriteFile(filepath.Join(dirs.Tasks, name), []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// Strategies "directory" is a file: reading it fails with a non-NotExist error.
	if err := os.WriteFile(dirs.Strategies, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	// Milestones and glossary dirs do not exist: silently skipped.

	st := openStore(t)
	err := watcher.NewPipeline(st, nil, dirs).ProcessAll(t.Context())
	if err == nil || !strings.Contains(err.Error(), "failed to read directory") {
		t.Fatalf("expected directory read error, got %v", err)
	}
	if _, err := st.GetTask(t.Context(), "ok"); err != nil {
		t.Fatalf("ok.md not loaded: %v", err)
	}
	if _, err := st.GetTask(t.Context(), ".hidden"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("hidden file must be skipped, got %v", err)
	}
}
