package model

import (
	"strings"
)

// Column represents a single board workflow column definition.
type Column struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

// ColumnState represents a Kanban board column populated with its matching tasks.
type ColumnState struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
	Tasks []Task `json:"tasks"`
	Count int    `json:"count"`
}

// BoardState represents the aggregated board snapshot served to the Web UI and MCP clients.
type BoardState struct {
	ProjectName    string        `json:"project_name"`
	Columns        []ColumnState `json:"columns"`
	EditableStates []string      `json:"editable_states,omitempty"`
}

// FilterCriteria defines filtering parameters for task listing and board views.
type FilterCriteria struct {
	Status      string   `json:"status,omitempty"`
	Milestone   string   `json:"milestone,omitempty"`
	Tag         string   `json:"tag,omitempty"`
	Priority    Priority `json:"priority,omitempty"`
	SearchQuery string   `json:"search_query,omitempty"`
}

// Matches reports whether a task satisfies all non-empty filter criteria.
func (f FilterCriteria) Matches(task Task) bool {
	if f.Status != "" && task.Status != f.Status {
		return false
	}

	if f.Milestone != "" && task.Milestone != f.Milestone {
		return false
	}

	if f.Tag != "" && !task.HasTag(f.Tag) {
		return false
	}

	if f.Priority != "" && task.Priority != f.Priority {
		return false
	}

	if f.SearchQuery != "" {
		query := strings.ToLower(f.SearchQuery)
		matchTitle := strings.Contains(strings.ToLower(task.Title), query)
		matchSummary := strings.Contains(strings.ToLower(task.Summary), query)
		if !matchTitle && !matchSummary {
			return false
		}
	}

	return true
}
