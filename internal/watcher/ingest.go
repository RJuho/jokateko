package watcher

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/RJuho/jokateko/internal/config"
	"github.com/RJuho/jokateko/internal/parser"
	"github.com/RJuho/jokateko/internal/store"
	"github.com/RJuho/jokateko/internal/writer"
)

// IngestEvent represents a notification that an entity changed in the store.
type IngestEvent struct {
	EntityType string
	EntityID   string
	Op         FileOp
}

// Pipeline coordinates filesystem events with parser, in-memory SQLite store,
// and suppression cache to prevent circular write-watch loops.
type Pipeline struct {
	store          *store.Store
	suppressCache  *writer.SuppressionCache
	dirs           config.Dirs
	onEntityChange func(IngestEvent)
}

// NewPipeline creates an ingestion pipeline for the given store, suppression cache, and entity directories.
func NewPipeline(st *store.Store, sc *writer.SuppressionCache, dirs config.Dirs) *Pipeline {
	return &Pipeline{
		store:         st,
		suppressCache: sc,
		dirs: config.Dirs{
			Tasks:      filepath.Clean(dirs.Tasks),
			Milestones: filepath.Clean(dirs.Milestones),
			Strategies: filepath.Clean(dirs.Strategies),
			Glossary:   filepath.Clean(dirs.Glossary),
		},
	}
}

// SetOnEntityChange registers a callback invoked when any entity is created, updated, or removed in the store.
func (p *Pipeline) SetOnEntityChange(fn func(IngestEvent)) {
	p.onEntityChange = fn
}

func (p *Pipeline) isInDir(dir, path string) bool {
	if dir == "" || dir == "." {
		return false
	}
	rel, err := filepath.Rel(dir, path)
	if err != nil {
		return false
	}
	// Match ".." as a whole path element only: "..foo.md" is inside dir.
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}

// HandleEvent processes a single debounced filesystem event.
func (p *Pipeline) HandleEvent(ctx context.Context, ev FileEvent) error {
	cleanPath := filepath.Clean(ev.Path)

	// Only process markdown files for entities
	if filepath.Ext(cleanPath) != ".md" {
		return nil
	}

	// 1. Identify entity type
	var entityType string
	switch {
	case p.isInDir(p.dirs.Tasks, cleanPath):
		entityType = "task"
	case p.isInDir(p.dirs.Milestones, cleanPath):
		entityType = "milestone"
	case p.isInDir(p.dirs.Strategies, cleanPath):
		entityType = "strategy"
	case p.isInDir(p.dirs.Glossary, cleanPath):
		entityType = "glossary"
	default:
		return nil // Not in a monitored entity folder
	}

	entityID := strings.TrimSuffix(filepath.Base(cleanPath), ".md")

	// 2. Handle delete operation
	if ev.Op == OpDelete {
		if p.suppressCache != nil && p.suppressCache.ShouldSuppressDelete(cleanPath) {
			return nil
		}

		var err error
		switch entityType {
		case "task":
			err = p.store.DeleteTask(ctx, entityID)
		case "milestone":
			err = p.store.DeleteMilestone(ctx, entityID)
		case "strategy":
			err = p.store.DeleteStrategy(ctx, entityID)
		case "glossary":
			err = p.store.DeleteGlossaryTerm(ctx, entityID)
		}
		if err != nil {
			return fmt.Errorf("failed to delete %s %q from store: %w", entityType, entityID, err)
		}

		if p.onEntityChange != nil {
			p.onEntityChange(IngestEvent{EntityType: entityType, EntityID: entityID, Op: OpDelete})
		}
		return nil
	}

	// 3. Handle write operation (create or update)
	data, err := os.ReadFile(cleanPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("failed to read file %q: %w", cleanPath, err)
	}

	if p.suppressCache != nil && p.suppressCache.ShouldSuppressWrite(cleanPath, data) {
		return nil
	}

	var modTime time.Time
	if fi, err := os.Stat(cleanPath); err == nil {
		modTime = fi.ModTime()
	}

	switch entityType {
	case "task":
		task, err := parser.ParseTaskWithCriteria(data, entityID)
		if err != nil {
			return fmt.Errorf("failed to parse task %q: %w", cleanPath, err)
		}
		task.FilePath = cleanPath
		task.ModTime = modTime
		if err := p.store.UpsertTask(ctx, *task); err != nil {
			return fmt.Errorf("failed to upsert task %q: %w", entityID, err)
		}

	case "milestone":
		ms, err := parser.ParseMilestone(data, entityID)
		if err != nil {
			return fmt.Errorf("failed to parse milestone %q: %w", cleanPath, err)
		}
		ms.FilePath = cleanPath
		ms.ModTime = modTime
		if err := p.store.UpsertMilestone(ctx, *ms); err != nil {
			return fmt.Errorf("failed to upsert milestone %q: %w", entityID, err)
		}

	case "strategy":
		strat, err := parser.ParseStrategy(data, entityID)
		if err != nil {
			return fmt.Errorf("failed to parse strategy %q: %w", cleanPath, err)
		}
		strat.FilePath = cleanPath
		strat.ModTime = modTime
		if err := p.store.UpsertStrategy(ctx, *strat); err != nil {
			return fmt.Errorf("failed to upsert strategy %q: %w", entityID, err)
		}

	case "glossary":
		term, err := parser.ParseGlossaryTerm(data, entityID)
		if err != nil {
			return fmt.Errorf("failed to parse glossary term %q: %w", cleanPath, err)
		}
		term.FilePath = cleanPath
		term.ModTime = modTime
		if err := p.store.UpsertGlossaryTerm(ctx, *term); err != nil {
			return fmt.Errorf("failed to upsert glossary term %q: %w", entityID, err)
		}
	}

	if p.onEntityChange != nil {
		p.onEntityChange(IngestEvent{EntityType: entityType, EntityID: entityID, Op: OpWrite})
	}

	return nil
}

// ProcessAll walks all entity directories and loads existing markdown files into the store.
// A file that fails to parse does not stop the scan; all failures are returned joined.
func (p *Pipeline) ProcessAll(ctx context.Context) error {
	var errs []error
	for _, d := range p.dirs.All() {
		if d == "" || d == "." {
			continue
		}
		entries, err := os.ReadDir(d)
		if err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				errs = append(errs, fmt.Errorf("failed to read directory %q: %w", d, err))
			}
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" || strings.HasPrefix(entry.Name(), ".") {
				continue
			}
			path := filepath.Join(d, entry.Name())
			if err := p.HandleEvent(ctx, FileEvent{Path: path, Op: OpWrite}); err != nil {
				errs = append(errs, err)
			}
		}
	}
	return errors.Join(errs...)
}

// StartPipeline loads every entity file into st and then keeps st in sync with
// filesystem changes until ctx is cancelled or stop is called. onChange (optional)
// is invoked for each ingested change. Initial scan and event errors are reported
// through logf. If the filesystem watcher cannot be started, the store is still
// loaded and err describes the watcher failure; stop is always safe to call.
func StartPipeline(ctx context.Context, st *store.Store, sc *writer.SuppressionCache, dirs config.Dirs, onChange func(IngestEvent), logf func(format string, args ...any)) (stop func(), err error) {
	pipeline := NewPipeline(st, sc, dirs)
	pipeline.SetOnEntityChange(onChange)

	if err := pipeline.ProcessAll(ctx); err != nil {
		logf("warning: errors during initial directory scan:\n%v", err)
	}

	fsw, err := New(dirs.All(), 50*time.Millisecond)
	if err != nil {
		return func() {}, fmt.Errorf("failed to start filesystem watcher: %w", err)
	}

	ctx, cancel := context.WithCancel(ctx)
	fsw.Start(ctx)

	go func() {
		for ev := range fsw.Events() {
			if err := pipeline.HandleEvent(ctx, ev); err != nil {
				logf("[ERROR] failed to handle file event for %s: %v", ev.Path, err)
			}
		}
	}()
	go func() {
		for err := range fsw.Errors() {
			logf("[ERROR] filesystem watcher: %v", err)
		}
	}()

	return func() {
		cancel()
		_ = fsw.Close()
	}, nil
}
