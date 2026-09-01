// Package model defines core domain entities, YAML frontmatter schemas,
// workflow enums, and filtering types for Jokateko.
package model

import (
	"slices"
	"time"
)

// Priority represents the urgency level of a task.
type Priority string

const (
	PriorityLow      Priority = "low"
	PriorityMedium   Priority = "medium"
	PriorityHigh     Priority = "high"
	PriorityCritical Priority = "critical"
)

// ValidPriorities is the list of all supported task priority levels.
var ValidPriorities = []Priority{
	PriorityLow,
	PriorityMedium,
	PriorityHigh,
	PriorityCritical,
}

// IsValid reports whether the priority is one of the allowed values.
func (p Priority) IsValid() bool {
	return slices.Contains(ValidPriorities, p)
}

// TaskFrontmatter represents the TOML frontmatter stored at the top of a task markdown file.
type TaskFrontmatter struct {
	Title        string   `json:"title" toml:"title" yaml:"title"`
	Status       string   `json:"status" toml:"status" yaml:"status"`
	Priority     Priority `json:"priority,omitempty" toml:"priority,omitempty" yaml:"priority,omitempty"`
	Milestone    string   `json:"milestone,omitempty" toml:"milestone,omitempty" yaml:"milestone,omitempty"`
	Tags         []string `json:"tags" toml:"tags" yaml:"tags"`
	Summary      string   `json:"summary" toml:"summary" yaml:"summary"`
	Dependencies []string `json:"dependencies,omitempty" toml:"dependencies,omitempty" yaml:"dependencies,omitempty"`
}

// Task represents a fully resolved actionable work item in Jokateko.
type Task struct {
	ID                string    `json:"id"`
	Title             string    `json:"title"`
	Status            string    `json:"status"`
	Priority          Priority  `json:"priority"`
	Milestone         string    `json:"milestone,omitempty"`
	Tags              []string  `json:"tags"`
	Summary           string    `json:"summary"`
	Dependencies      []string  `json:"dependencies"`
	Body              string    `json:"body,omitempty"`
	TotalCriteria     int       `json:"total_criteria,omitzero"`
	CompletedCriteria int       `json:"completed_criteria,omitzero"`
	FilePath          string    `json:"file_path,omitempty"`
	ModTime           time.Time `json:"mod_time,omitzero"`
}

// Frontmatter returns the TaskFrontmatter representation of the task.
func (t Task) Frontmatter() TaskFrontmatter {
	p := t.Priority
	if p == "" {
		p = PriorityMedium
	}

	tags := t.Tags
	if tags == nil {
		tags = []string{}
	}

	deps := t.Dependencies
	if deps == nil {
		deps = []string{}
	}

	return TaskFrontmatter{
		Title:        t.Title,
		Status:       t.Status,
		Priority:     p,
		Milestone:    t.Milestone,
		Tags:         tags,
		Summary:      t.Summary,
		Dependencies: deps,
	}
}

// HasDependency returns true if the task lists the given task slug as a dependency.
func (t Task) HasDependency(slug string) bool {
	return slices.Contains(t.Dependencies, slug)
}

// HasTag returns true if the task has the given tag.
func (t Task) HasTag(tag string) bool {
	return slices.Contains(t.Tags, tag)
}

// IsDone returns true if the task status is "done".
func (t Task) IsDone() bool {
	return t.Status == "done"
}
