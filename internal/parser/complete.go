package parser

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

var (
	// checkboxLineRegex matches standard markdown checkboxes: - [ ] or * [x] or - [X]
	checkboxLineRegex = regexp.MustCompile(`(?m)^(\s*[-*]\s+)\[([ xX])\](.*)$`)

	// completionSummaryRegex matches an existing ## Completion Summary header and anything following it
	completionSummaryRegex = regexp.MustCompile(`(?ms)\n*## Completion Summary.*$`)
)

// ErrOpenCheckboxes is returned when attempting to complete a task that still has open checklist items.
type ErrOpenCheckboxes struct {
	OpenCount  int
	TotalCount int
}

func (e *ErrOpenCheckboxes) Error() string {
	return fmt.Sprintf("cannot complete task: %d acceptance criteria items remain uncompleted (out of %d total). Use list_task_items to view remaining items and update_task_item to mark them as completed before completing the task", e.OpenCount, e.TotalCount)
}

// VerifyAllCheckboxesCompleted verifies that no open checkboxes remain in the task body.
// Tasks with 0 checkboxes pass verification immediately.
func VerifyAllCheckboxesCompleted(body string) error {
	total, completed, _ := ExtractAcceptanceCriteria([]byte(body))
	if total > completed {
		return &ErrOpenCheckboxes{
			OpenCount:  total - completed,
			TotalCount: total,
		}
	}
	return nil
}

// UpdateCheckboxByIndex toggles the 1-based targetIndex-th checkbox in the markdown body
// between completed (- [x]) and uncompleted (- [ ]).
func UpdateCheckboxByIndex(body string, targetIndex int, completed bool) (string, error) {
	matches := checkboxLineRegex.FindAllStringSubmatchIndex(body, -1)
	if len(matches) == 0 {
		return "", fmt.Errorf("task has no checklist items")
	}

	if targetIndex < 1 || targetIndex > len(matches) {
		return "", fmt.Errorf("checkbox index %d out of bounds (task has %d checklist items)", targetIndex, len(matches))
	}

	// targetIndex is 1-based, matchIndex is 0-based
	match := matches[targetIndex-1]
	// match[4] and match[5] are the start and end indices of group 2: ([ xX])
	charStart := match[4]
	charEnd := match[5]

	replacementChar := " "
	if completed {
		replacementChar = "x"
	}

	var buf strings.Builder
	buf.WriteString(body[:charStart])
	buf.WriteString(replacementChar)
	buf.WriteString(body[charEnd:])

	return buf.String(), nil
}

// FormatCompletionSummary produces a standardized markdown completion section.
func FormatCompletionSummary(completedAt time.Time, whatWasDone, whyRationale string) string {
	var buf strings.Builder
	buf.WriteString("## Completion Summary\n")
	fmt.Fprintf(&buf, "- **Completed At:** %s\n\n", completedAt.UTC().Format(time.RFC3339))

	buf.WriteString("### What Was Done\n")
	if strings.TrimSpace(whatWasDone) != "" {
		buf.WriteString(strings.TrimSpace(whatWasDone))
	} else {
		buf.WriteString("Task completed successfully.")
	}
	buf.WriteString("\n\n")

	buf.WriteString("### Why / Rationale\n")
	if strings.TrimSpace(whyRationale) != "" {
		buf.WriteString(strings.TrimSpace(whyRationale))
	} else {
		buf.WriteString("Satisfies all task acceptance criteria.")
	}
	buf.WriteString("\n")

	return buf.String()
}

// CompleteTaskBody validates that all acceptance criteria are completed,
// removes any previous completion summary, and appends a fresh, formatted
// ## Completion Summary section. If open checkboxes exist, it strictly rejects completion.
func CompleteTaskBody(body string, completedAt time.Time, whatWasDone, whyRationale string) (string, error) {
	if err := VerifyAllCheckboxesCompleted(body); err != nil {
		return "", err
	}

	// Strip any existing completion summary section
	cleanBody := strings.TrimRight(completionSummaryRegex.ReplaceAllString(body, ""), "\r\n")

	summary := FormatCompletionSummary(completedAt, whatWasDone, whyRationale)

	if cleanBody == "" {
		return summary, nil
	}

	return cleanBody + "\n\n" + summary, nil
}
