package watcher_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/RJuho/jokateko/internal/config"
	"github.com/RJuho/jokateko/internal/store"
	"github.com/RJuho/jokateko/internal/watcher"
	"github.com/RJuho/jokateko/internal/writer"
)

func TestWatcherDebounceAndIgnore(t *testing.T) {
	tempDir := t.TempDir()
	watchDir := filepath.Join(tempDir, "tasks")

	w, err := watcher.New([]string{watchDir}, 40*time.Millisecond)
	if err != nil {
		t.Fatalf("failed to create watcher: %v", err)
	}
	defer func() {
		_ = w.Close()
	}()

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	w.Start(ctx)

	// Write an ignored file (e.g. .swp or .tmp)
	ignoredPath := filepath.Join(watchDir, ".task.md.tmp")
	_ = os.WriteFile(ignoredPath, []byte("ignored"), 0644)

	// Write a valid markdown file multiple times in rapid succession
	validPath := filepath.Join(watchDir, "260901-test.md")
	for range 3 {
		_ = os.WriteFile(validPath, []byte("content"), 0644)
		time.Sleep(5 * time.Millisecond)
	}

	// Read event from channel
	select {
	case ev := <-w.Events():
		if ev.Path != validPath || ev.Op != watcher.OpWrite {
			t.Errorf("unexpected event: %+v", ev)
		}
	case <-time.After(250 * time.Millisecond):
		t.Fatal("timed out waiting for debounced event")
	}

	// Verify no other events arrive (the ignored file produced no event, rapid writes were coalesced)
	select {
	case ev := <-w.Events():
		t.Errorf("unexpected additional event received: %+v", ev)
	case <-time.After(100 * time.Millisecond):
		// Success: no extra events
	}
}

func TestIngestionPipeline(t *testing.T) {
	tempDir := t.TempDir()
	tasksDir := filepath.Join(tempDir, "tasks")
	msDir := filepath.Join(tempDir, "milestones")
	stratDir := filepath.Join(tempDir, "strategies")
	glossDir := filepath.Join(tempDir, "glossary")

	for _, d := range []string{tasksDir, msDir, stratDir, glossDir} {
		_ = os.MkdirAll(d, 0755)
	}

	st, err := store.OpenMemory()
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer func() {
		_ = st.Close()
	}()

	cache := writer.NewSuppressionCache(500 * time.Millisecond)
	wr := writer.New(cache)

	cfg := config.Dirs{
		Tasks:      tasksDir,
		Milestones: msDir,
		Strategies: stratDir,
		Glossary:   glossDir,
	}

	pipeline := watcher.NewPipeline(st, cache, cfg)

	var entityChangeCount atomic.Int32
	pipeline.SetOnEntityChange(func(e watcher.IngestEvent) {
		entityChangeCount.Add(1)
	})

	ctx := context.Background()

	// 1. Task ingestion (external write without suppression)
	taskFile := filepath.Join(tasksDir, "260901-test-task.md")
	taskContent := `+++
title = "Ingested Task"
status = "ready"
priority = "high"
summary = "Summary of task"
tags = ["test", "ci"]
dependencies = []
+++

## Acceptance Criteria
- [ ] Task item 1
- [x] Task item 2
`
	if err := os.WriteFile(taskFile, []byte(taskContent), 0644); err != nil {
		t.Fatalf("failed to write task file: %v", err)
	}

	if err := pipeline.HandleEvent(ctx, watcher.FileEvent{Path: taskFile, Op: watcher.OpWrite}); err != nil {
		t.Fatalf("HandleEvent task write failed: %v", err)
	}

	// Verify task in store
	gotTask, err := st.GetTask(ctx, "260901-test-task")
	if err != nil {
		t.Fatalf("failed to get task from store: %v", err)
	}
	if gotTask.Title != "Ingested Task" || gotTask.TotalCriteria != 2 || gotTask.CompletedCriteria != 1 {
		t.Errorf("unexpected task in store: %+v", gotTask)
	}

	// 2. Suppression test (writer writes file)
	entityChangeCount.Store(0)
	updatedTaskContent := `+++
title = "Suppressed Task"
status = "in_progress"
priority = "high"
summary = "Updated summary"
tags = ["test"]
dependencies = []
+++

Body text
`
	if err := wr.WriteFile(taskFile, []byte(updatedTaskContent), 0644); err != nil {
		t.Fatalf("writer WriteFile failed: %v", err)
	}

	// Handle event for this write; suppression cache should swallow it
	if err := pipeline.HandleEvent(ctx, watcher.FileEvent{Path: taskFile, Op: watcher.OpWrite}); err != nil {
		t.Fatalf("HandleEvent should succeed: %v", err)
	}
	if count := entityChangeCount.Load(); count != 0 {
		t.Errorf("expected event to be suppressed, but callback was called %d times", count)
	}

	// 3. Task deletion
	if err := os.Remove(taskFile); err != nil {
		t.Fatalf("failed to delete task file: %v", err)
	}
	if err := pipeline.HandleEvent(ctx, watcher.FileEvent{Path: taskFile, Op: watcher.OpDelete}); err != nil {
		t.Fatalf("HandleEvent task delete failed: %v", err)
	}
	if _, err := st.GetTask(ctx, "260901-test-task"); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("expected ErrNotFound for deleted task, got %v", err)
	}

	// 4. Milestone, Strategy, Glossary, Config tests
	msFile := filepath.Join(msDir, "260915-mvp.md")
	msContent := `+++
title = "MVP Milestone"
status = "open"
target_date = "2026-09-15"
summary = "MVP Summary"
tags = ["mvp"]
+++
`
	_ = os.WriteFile(msFile, []byte(msContent), 0644)
	if err := pipeline.HandleEvent(ctx, watcher.FileEvent{Path: msFile, Op: watcher.OpWrite}); err != nil {
		t.Fatalf("HandleEvent milestone failed: %v", err)
	}
	if _, err := st.GetMilestone(ctx, "260915-mvp"); err != nil {
		t.Errorf("milestone not found in store: %v", err)
	}

	stratFile := filepath.Join(stratDir, "zero-cgo.md")
	stratContent := `+++
title = "Zero CGO"
tier = 1
summary = "Pure Go rules"
tags = ["arch"]
+++
`
	_ = os.WriteFile(stratFile, []byte(stratContent), 0644)
	if err := pipeline.HandleEvent(ctx, watcher.FileEvent{Path: stratFile, Op: watcher.OpWrite}); err != nil {
		t.Fatalf("HandleEvent strategy failed: %v", err)
	}
	if _, err := st.GetStrategy(ctx, "zero-cgo"); err != nil {
		t.Errorf("strategy not found in store: %v", err)
	}

	glossFile := filepath.Join(glossDir, "tac.md")
	glossContent := `+++
title = "Tasks-as-Code"
summary = "Concept definition"
tags = ["concept"]
+++
`
	_ = os.WriteFile(glossFile, []byte(glossContent), 0644)
	if err := pipeline.HandleEvent(ctx, watcher.FileEvent{Path: glossFile, Op: watcher.OpWrite}); err != nil {
		t.Fatalf("HandleEvent glossary failed: %v", err)
	}
	if _, err := st.GetGlossaryTerm(ctx, "tac"); err != nil {
		t.Errorf("glossary term not found in store: %v", err)
	}

	// 5. Test ProcessAll
	if err := pipeline.ProcessAll(ctx); err != nil {
		t.Fatalf("ProcessAll failed: %v", err)
	}
}

func TestProcessAll_ContinuesPastMalformedFiles(t *testing.T) {
	dir := t.TempDir()
	tasksDir := filepath.Join(dir, "tasks")
	_ = os.MkdirAll(tasksDir, 0o755)

	// "a-broken" sorts first so a fail-fast scan would never reach the good file.
	_ = os.WriteFile(filepath.Join(tasksDir, "a-broken.md"), []byte("+++\ntitle = \n+++\n"), 0o644)
	_ = os.WriteFile(filepath.Join(tasksDir, "b-good.md"), []byte("+++\ntitle = \"Good\"\nstatus = \"backlog\"\nsummary = \"ok\"\n+++\n"), 0o644)

	st, err := store.OpenMemory()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = st.Close() }()

	p := watcher.NewPipeline(st, nil, config.Dirs{Tasks: tasksDir})
	err = p.ProcessAll(t.Context())
	if err == nil || !strings.Contains(err.Error(), "a-broken.md") {
		t.Fatalf("expected an error naming the malformed file, got %v", err)
	}
	if _, err := st.GetTask(t.Context(), "b-good"); err != nil {
		t.Fatalf("valid file after a malformed one was not loaded: %v", err)
	}
}
