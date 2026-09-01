package model

import (
	"math"
	"slices"
	"time"
)

// MilestoneStatus represents the active lifecycle of a milestone.
type MilestoneStatus string

const (
	MilestoneStatusOpen   MilestoneStatus = "open"
	MilestoneStatusClosed MilestoneStatus = "closed"
)

// MilestoneFrontmatter represents TOML frontmatter for a milestone file.
type MilestoneFrontmatter struct {
	Title      string          `json:"title" toml:"title" yaml:"title"`
	Status     MilestoneStatus `json:"status" toml:"status" yaml:"status"`
	TargetDate string          `json:"target_date,omitempty" toml:"target_date,omitempty" yaml:"target_date,omitempty"`
	Tags       []string        `json:"tags" toml:"tags" yaml:"tags"`
	Summary    string          `json:"summary" toml:"summary" yaml:"summary"`
}

// Milestone represents a grouping of related tasks toward a deliverable target.
type Milestone struct {
	ID                 string          `json:"id"`
	Title              string          `json:"title"`
	Status             MilestoneStatus `json:"status"`
	IsArchived         bool            `json:"is_archived"`
	TargetDate         string          `json:"target_date,omitempty"`
	Tags               []string        `json:"tags"`
	Summary            string          `json:"summary"`
	Body               string          `json:"body,omitempty"`
	TotalTasks         int             `json:"total_tasks"`
	CompletedTasks     int             `json:"completed_tasks"`
	ProgressPercentage float64         `json:"progress_percentage"`
	FilePath           string          `json:"file_path,omitempty"`
	ModTime            time.Time       `json:"mod_time,omitzero"`
}

// Frontmatter returns the MilestoneFrontmatter representation.
func (m Milestone) Frontmatter() MilestoneFrontmatter {
	status := m.Status
	if status == "" {
		status = MilestoneStatusOpen
	}

	tags := m.Tags
	if tags == nil {
		tags = []string{}
	}

	return MilestoneFrontmatter{
		Title:      m.Title,
		Status:     status,
		TargetDate: m.TargetDate,
		Tags:       tags,
		Summary:    m.Summary,
	}
}

// RecalculateProgress updates TotalTasks, CompletedTasks, ProgressPercentage,
// and auto-archive status based on the assigned tasks.
func (m *Milestone) RecalculateProgress(tasks []Task) {
	total := 0
	completed := 0

	for _, t := range tasks {
		if t.Milestone == m.ID {
			total++
			if t.IsDone() {
				completed++
			}
		}
	}

	m.TotalTasks = total
	m.CompletedTasks = completed

	if total > 0 {
		pct := (float64(completed) / float64(total)) * 100.0
		// Round to 1 decimal place
		m.ProgressPercentage = math.Round(pct*10) / 10
		// If 100% completed, auto-archive per specification
		if completed == total {
			m.IsArchived = true
		} else {
			m.IsArchived = false
		}
	} else {
		m.ProgressPercentage = 0.0
		m.IsArchived = false
	}
}

// HasTag returns true if the milestone has the given tag.
func (m Milestone) HasTag(tag string) bool {
	return slices.Contains(m.Tags, tag)
}
