package model

import (
	"cmp"
	"slices"
	"strings"
)

// PriorityRank returns an integer rank for priority comparisons.
// Lower numerical rank corresponds to higher urgency (critical is lowest rank number, thus first).
func PriorityRank(p Priority) int {
	switch p {
	case PriorityCritical:
		return 0
	case PriorityHigh:
		return 1
	case PriorityMedium:
		return 2
	case PriorityLow:
		return 3
	default:
		return 2
	}
}

// CompareTasksForColumn compares two tasks according to the column's configured sort rules
// or default workflow state sorting rules.
// Returns -1 if a < b, 1 if a > b, 0 if equal.
func CompareTasksForColumn(a, b Task, col Column) int {
	sortBy := strings.ToLower(strings.TrimSpace(col.SortBy))
	sortDir := strings.ToLower(strings.TrimSpace(col.SortDirection))

	// For 'done' state, default sorting is Recently changed (changed_at desc),
	// since priority does not matter anymore when task is done.
	if col.ID == "done" || a.Status == "done" {
		if sortBy != "" && sortBy != "default" {
			if res := compareCustomField(a, b, sortBy, sortDir); res != 0 {
				return res
			}
		} else {
			if res := cmp.Compare(b.ChangedAt, a.ChangedAt); res != 0 {
				return res
			}
		}
		return cmp.Compare(a.ID, b.ID)
	}

	// Primary Tier for active and backlog states: Priority (critical > high > medium > low)
	pA := PriorityRank(a.Priority)
	pB := PriorityRank(b.Priority)
	if pA != pB {
		return cmp.Compare(pA, pB)
	}

	// Secondary Tier: Custom column sort or state-specific default workflow rules
	if sortBy != "" && sortBy != "default" {
		if res := compareCustomField(a, b, sortBy, sortDir); res != 0 {
			return res
		}
	} else {
		if res := compareDefaultWorkflowTier(a, b, col.ID); res != 0 {
			return res
		}
	}

	// Tertiary Tier: Deterministic tie-breaking on ID ascending
	return cmp.Compare(a.ID, b.ID)
}

func compareDefaultWorkflowTier(a, b Task, colID string) int {
	switch colID {
	case "ready", "in_progress", "in_review":
		// Target date ascending (soonest deadline first).
		// Tasks with target_at set precede tasks with empty target_at.
		hasTargetA := strings.TrimSpace(a.TargetAt) != ""
		hasTargetB := strings.TrimSpace(b.TargetAt) != ""

		if hasTargetA && hasTargetB {
			if res := cmp.Compare(a.TargetAt, b.TargetAt); res != 0 {
				return res
			}
			// Same target_at -> fallback to changed_at descending
			return cmp.Compare(b.ChangedAt, a.ChangedAt)
		}
		if hasTargetA && !hasTargetB {
			return -1 // A has target date, so A comes first
		}
		if !hasTargetA && hasTargetB {
			return 1 // B has target date, so B comes first
		}
		// Neither has target_at -> fallback to changed_at descending
		return cmp.Compare(b.ChangedAt, a.ChangedAt)

	case "backlog", "done":
		fallthrough
	default:
		// changed_at descending (newest first)
		return cmp.Compare(b.ChangedAt, a.ChangedAt)
	}
}

func compareCustomField(a, b Task, field, direction string) int {
	isDesc := direction == "desc"

	switch field {
	case "target_at":
		hasTargetA := strings.TrimSpace(a.TargetAt) != ""
		hasTargetB := strings.TrimSpace(b.TargetAt) != ""

		if hasTargetA && hasTargetB {
			res := cmp.Compare(a.TargetAt, b.TargetAt)
			if isDesc {
				res = -res
			}
			if res != 0 {
				return res
			}
			return cmp.Compare(b.ChangedAt, a.ChangedAt)
		}
		if hasTargetA && !hasTargetB {
			return -1
		}
		if !hasTargetA && hasTargetB {
			return 1
		}
		return cmp.Compare(b.ChangedAt, a.ChangedAt)

	case "changed_at":
		if isDesc {
			return cmp.Compare(b.ChangedAt, a.ChangedAt)
		}
		return cmp.Compare(a.ChangedAt, b.ChangedAt)

	case "created_at":
		if isDesc {
			return cmp.Compare(b.CreatedAt, a.CreatedAt)
		}
		return cmp.Compare(a.CreatedAt, b.CreatedAt)

	case "title":
		res := strings.Compare(strings.ToLower(a.Title), strings.ToLower(b.Title))
		if isDesc {
			return -res
		}
		return res

	case "id":
		if isDesc {
			return cmp.Compare(b.ID, a.ID)
		}
		return cmp.Compare(a.ID, b.ID)

	case "priority":
		// Already compared in primary tier, tie-break with changed_at desc
		return cmp.Compare(b.ChangedAt, a.ChangedAt)

	default:
		return 0
	}
}

// SortTasksForColumn sorts a slice of tasks in-place for a specific column.
func SortTasksForColumn(tasks []Task, col Column) {
	slices.SortFunc(tasks, func(a, b Task) int {
		return CompareTasksForColumn(a, b, col)
	})
}

// SortTasksByColumnOrder sorts a slice of tasks across multiple columns:
// first by the column order defined in columns, and within each column by that column's sort rules.
// Any tasks belonging to an unknown column are placed at the end, sorted by default workflow rules.
func SortTasksByColumnOrder(tasks []Task, columns []Column) []Task {
	colIndex := make(map[string]int, len(columns))
	colMap := make(map[string]Column, len(columns))
	for i, c := range columns {
		colIndex[c.ID] = i
		colMap[c.ID] = c
	}

	result := slices.Clone(tasks)
	slices.SortFunc(result, func(a, b Task) int {
		idxA, okA := colIndex[a.Status]
		idxB, okB := colIndex[b.Status]

		if okA && okB {
			if idxA != idxB {
				return cmp.Compare(idxA, idxB)
			}
			return CompareTasksForColumn(a, b, colMap[a.Status])
		}
		if okA && !okB {
			return -1
		}
		if !okA && okB {
			return 1
		}

		// Both unknown columns
		if a.Status != b.Status {
			return cmp.Compare(a.Status, b.Status)
		}
		return CompareTasksForColumn(a, b, Column{ID: a.Status})
	})

	return result
}
