// Package exporter provides functionality to serialize in-memory project data
// into self-contained, offline-first HTML snapshots.
package exporter

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/RJuho/jokateko/internal/config"
	"github.com/RJuho/jokateko/internal/model"
	"github.com/RJuho/jokateko/internal/store"
	"github.com/RJuho/jokateko/internal/version"
)

// BuildSnapshot extracts all entities from the store and configuration to form an offline snapshot.
func BuildSnapshot(ctx context.Context, cfg *config.Config, st *store.Store) (*model.Snapshot, error) {
	if cfg == nil {
		cfg = config.Default("")
	}

	cols := make([]model.Column, 0, len(cfg.Board.Columns))
	for _, c := range cfg.Board.Columns {
		cols = append(cols, model.Column{
			ID:    c.ID,
			Name:  c.Name,
			Color: c.Color,
		})
	}

	tasks, err := st.ListTasks(ctx, model.FilterCriteria{})
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks for snapshot: %w", err)
	}
	if tasks == nil {
		tasks = []model.Task{}
	}

	milestones, err := st.ListMilestones(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list milestones for snapshot: %w", err)
	}
	if milestones == nil {
		milestones = []model.Milestone{}
	}

	strategies, err := st.ListStrategies(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list strategies for snapshot: %w", err)
	}
	if strategies == nil {
		strategies = []model.Strategy{}
	}

	glossary, err := st.ListGlossaryTerms(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list glossary terms for snapshot: %w", err)
	}
	if glossary == nil {
		glossary = []model.GlossaryTerm{}
	}

	vInfo := version.Get()
	branch, commit := resolveGitInfo(".")
	if commit == "" && vInfo.Commit != "none" {
		commit = vInfo.Commit
	}

	snap := &model.Snapshot{
		Config: model.SnapshotConfig{
			Project: model.ProjectConfig{
				Name:        cfg.Project.Name,
				Description: cfg.Project.Description,
			},
			Board: model.BoardConfig{
				Columns: cols,
			},
			Tags: model.TagsConfig{
				Allowed:        cfg.Tags.Allowed,
				EnforceAllowed: cfg.Tags.EnforceAllowed,
			},
			Build: model.BuildConfig{
				Time:    time.Now().UTC().Format(time.RFC3339),
				Branch:  branch,
				Commit:  commit,
				Version: vInfo.Version,
			},
		},
		Tasks:      tasks,
		Milestones: milestones,
		Strategies: strategies,
		Glossary:   glossary,
	}

	return snap, nil
}

func resolveGitInfo(dir string) (branch, commit string) {
	headPath := filepath.Join(dir, ".git", "HEAD")
	data, err := os.ReadFile(headPath)
	if err != nil {
		return "", ""
	}
	s := strings.TrimSpace(string(data))
	if strings.HasPrefix(s, "ref: refs/heads/") {
		branch = strings.TrimPrefix(s, "ref: refs/heads/")
		refPath := filepath.Join(dir, ".git", "refs", "heads", branch)
		if refData, err := os.ReadFile(refPath); err == nil {
			commit = strings.TrimSpace(string(refData))
		}
	} else if len(s) == 40 {
		commit = s
	}
	return branch, commit
}

// SerializeSnapshot serializes a Snapshot into minified, HTML-safe JSON bytes.
// Go's standard json.Marshal automatically escapes '<', '>', and '&' to prevent
// script breakout vulnerabilities when embedded inside HTML <script> tags.
func SerializeSnapshot(snap *model.Snapshot) ([]byte, error) {
	if snap == nil {
		return nil, fmt.Errorf("snapshot cannot be nil")
	}

	data, err := json.Marshal(snap)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize snapshot to JSON: %w", err)
	}

	return data, nil
}
