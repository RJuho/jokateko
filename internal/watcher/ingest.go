package watcher

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/RJuho/jokateko/internal/parser"
	"github.com/RJuho/jokateko/internal/store"
	"github.com/RJuho/jokateko/internal/writer"
)

// IngestConfig defines directory paths monitored by the ingestion pipeline.
type IngestConfig struct {
	TasksDir      string
	MilestonesDir string
	StrategiesDir string
	GlossaryDir   string
	ConfigFile    string
}

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
	cfg            IngestConfig
	onConfigChange func()
	onEntityChange func(IngestEvent)
}

// NewPipeline creates an ingestion pipeline with the given store, suppression cache, and directory layout.
func NewPipeline(st *store.Store, sc *writer.SuppressionCache, cfg IngestConfig) *Pipeline {
	return &Pipeline{
		store:         st,
		suppressCache: sc,
		cfg: IngestConfig{
			TasksDir:      filepath.Clean(cfg.TasksDir),
			MilestonesDir: filepath.Clean(cfg.MilestonesDir),
			StrategiesDir: filepath.Clean(cfg.StrategiesDir),
			GlossaryDir:   filepath.Clean(cfg.GlossaryDir),
			ConfigFile:    filepath.Clean(cfg.ConfigFile),
		},
	}
}

// SetOnConfigChange registers a callback invoked when the configuration file changes.
func (p *Pipeline) SetOnConfigChange(fn func()) {
	p.onConfigChange = fn
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
	return !strings.HasPrefix(rel, "..") && !filepath.IsAbs(rel)
}

// HandleEvent processes a single debounced filesystem event.
func (p *Pipeline) HandleEvent(ctx context.Context, ev FileEvent) error {
	cleanPath := filepath.Clean(ev.Path)

	// 1. Config file handling
	if p.cfg.ConfigFile != "" && cleanPath == p.cfg.ConfigFile {
		if p.onConfigChange != nil {
			p.onConfigChange()
		}
		return nil
	}

	// Only process markdown files for entities
	if filepath.Ext(cleanPath) != ".md" {
		return nil
	}

	// 2. Identify entity type
	var entityType string
	switch {
	case p.isInDir(p.cfg.TasksDir, cleanPath):
		entityType = "task"
	case p.isInDir(p.cfg.MilestonesDir, cleanPath):
		entityType = "milestone"
	case p.isInDir(p.cfg.StrategiesDir, cleanPath):
		entityType = "strategy"
	case p.isInDir(p.cfg.GlossaryDir, cleanPath):
		entityType = "glossary"
	default:
		return nil // Not in a monitored entity folder
	}

	entityID := strings.TrimSuffix(filepath.Base(cleanPath), ".md")

	// 3. Handle delete operation
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

	// 4. Handle write operation (create or update)
	data, err := os.ReadFile(cleanPath)
	if err != nil {
		if os.IsNotExist(err) {
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

// ProcessAll walks all configured entity directories and loads existing markdown files into the store.
func (p *Pipeline) ProcessAll(ctx context.Context) error {
	dirs := []string{
		p.cfg.TasksDir,
		p.cfg.MilestonesDir,
		p.cfg.StrategiesDir,
		p.cfg.GlossaryDir,
	}

	for _, d := range dirs {
		if d == "" {
			continue
		}
		entries, err := os.ReadDir(d)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("failed to read directory %q: %w", d, err)
		}

		for _, entry := range entries {
			if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" || strings.HasPrefix(entry.Name(), ".") {
				continue
			}
			path := filepath.Join(d, entry.Name())
			if err := p.HandleEvent(ctx, FileEvent{Path: path, Op: OpWrite}); err != nil {
				return err
			}
		}
	}

	return nil
}
