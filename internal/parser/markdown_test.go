package parser_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/RJuho/jokateko/internal/parser"
)

func TestExtractAcceptanceCriteria(t *testing.T) {
	md := `
## Description
Implement the database.

## Acceptance Criteria
- [x] Pure Go SQLite (completed)
- [ ] In-memory support (pending)
- [x] sqlc code generation (completed)
- [ ] Zero CGO compilation (pending)
- [ ] Verify with ` + "`go test ./...`" + ` command
`
	total, completed, items := parser.ExtractAcceptanceCriteria([]byte(md))
	if total != 5 {
		t.Errorf("expected 5 total criteria, got %d", total)
	}
	if completed != 2 {
		t.Errorf("expected 2 completed criteria, got %d", completed)
	}
	if len(items) != 5 {
		t.Fatalf("expected 5 criteria items, got %d", len(items))
	}

	if items[0].Index != 1 || items[0].Text != "Pure Go SQLite (completed)" || !items[0].Completed {
		t.Errorf("unexpected first criterion: %+v", items[0])
	}
	if items[1].Index != 2 || items[1].Text != "In-memory support (pending)" || items[1].Completed {
		t.Errorf("unexpected second criterion: %+v", items[1])
	}
	if items[4].Index != 5 || items[4].Text != "Verify with `go test ./...` command" && items[4].Text != "Verify with go test ./... command" {
		t.Logf("criterion 4 text: %q", items[4].Text)
	}

	if !parser.HasOpenCheckboxes([]byte(md)) {
		t.Error("expected HasOpenCheckboxes to be true")
	}
}

func TestExtractHeadings(t *testing.T) {
	md := `
# Project Title
## Overview
### Technical Details
`
	headings := parser.ExtractHeadings([]byte(md))
	if len(headings) != 3 {
		t.Fatalf("expected 3 headings, got %d", len(headings))
	}

	if headings[0].Level != 1 || headings[0].Text != "Project Title" {
		t.Errorf("unexpected heading 0: %+v", headings[0])
	}
	if headings[1].Level != 2 || headings[1].Text != "Overview" {
		t.Errorf("unexpected heading 1: %+v", headings[1])
	}
	if headings[2].Level != 3 || headings[2].Text != "Technical Details" {
		t.Errorf("unexpected heading 2: %+v", headings[2])
	}
}

func TestRenderHTML(t *testing.T) {
	md := `## Section
This is **bold** text and ` + "`code`" + `.`

	html, err := parser.RenderHTML([]byte(md))
	if err != nil {
		t.Fatalf("expected successful HTML render, got: %v", err)
	}

	if !strings.Contains(html, "<h2>Section</h2>") {
		t.Errorf("missing h2 tag in rendered HTML: %s", html)
	}
	if !strings.Contains(html, "<strong>bold</strong>") {
		t.Errorf("missing strong tag in rendered HTML: %s", html)
	}
	if !strings.Contains(html, "<code>code</code>") {
		t.Errorf("missing code tag in rendered HTML: %s", html)
	}
}

func TestRenderHTMLTaskCheckboxes(t *testing.T) {
	md := "- [ ] first\n- [x] second <b>&</b>\n"

	html, err := parser.RenderHTML([]byte(md))
	if err != nil {
		t.Fatalf("expected successful HTML render, got: %v", err)
	}

	for _, want := range []string{
		`<input type="checkbox" name="criterion-1" aria-label="first" `,
		`data-checkbox-index="1"`,
		`<input type="checkbox" checked name="criterion-2" aria-label="second &amp;" `,
		`data-checkbox-index="2"`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("missing %q in rendered HTML: %s", want, html)
		}
	}
}

func TestParseTaskWithCriteria(t *testing.T) {
	doc := `+++
title = "Implement Database"
status = "in_progress"
summary = "Setup SQLite with sqlc"
+++

## Acceptance Criteria
- [x] Pure Go modernc.org/sqlite
- [ ] sqlc queries configured
- [ ] in-memory test runner
`
	task, err := parser.ParseTaskWithCriteria([]byte(doc), "260901-db")
	if err != nil {
		t.Fatalf("expected successful task parse, got: %v", err)
	}

	if task.TotalCriteria != 3 {
		t.Errorf("expected 3 total criteria, got %d", task.TotalCriteria)
	}
	if task.CompletedCriteria != 1 {
		t.Errorf("expected 1 completed criterion, got %d", task.CompletedCriteria)
	}
}

func TestUpdateCheckboxByIndex(t *testing.T) {
	body := `
## Acceptance Criteria
- [ ] Item 1 (first)
- [x] Item 2 (second)
* [ ] Item 3 (third)
`
	// Mark Item 1 as completed (index 1)
	updated, err := parser.UpdateCheckboxByIndex(body, 1, true)
	if err != nil {
		t.Fatalf("unexpected error updating item 1: %v", err)
	}
	if !strings.Contains(updated, "- [x] Item 1 (first)") {
		t.Errorf("item 1 not checked: %s", updated)
	}

	// Mark Item 2 as uncompleted (index 2)
	updated2, err := parser.UpdateCheckboxByIndex(body, 2, false)
	if err != nil {
		t.Fatalf("unexpected error unchecking item 2: %v", err)
	}
	if !strings.Contains(updated2, "- [ ] Item 2 (second)") {
		t.Errorf("item 2 not unchecked: %s", updated2)
	}

	// Out of bounds checks
	_, err = parser.UpdateCheckboxByIndex(body, 0, true)
	if err == nil || !strings.Contains(err.Error(), "out of bounds") {
		t.Errorf("expected out of bounds error for index 0, got: %v", err)
	}

	_, err = parser.UpdateCheckboxByIndex(body, 99, true)
	if err == nil || !strings.Contains(err.Error(), "out of bounds") {
		t.Errorf("expected out of bounds error for index 99, got: %v", err)
	}
}

func TestCompleteTaskBody(t *testing.T) {
	ts := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)

	t.Run("rejects when open checkboxes remain", func(t *testing.T) {
		body := `
## Acceptance Criteria
- [x] Item 1
- [ ] Item 2
- [ ] Item 3
`
		_, err := parser.CompleteTaskBody(body, ts, "Finished work", "All done")
		if err == nil {
			t.Fatal("expected error completing task with open checkboxes, got nil")
		}

		var openErr *parser.ErrOpenCheckboxes
		if !errors.As(err, &openErr) {
			t.Fatalf("expected ErrOpenCheckboxes type, got: %T (%v)", err, err)
		}
		if openErr.OpenCount != 2 || openErr.TotalCount != 3 {
			t.Errorf("expected 2 open out of 3, got: open=%d, total=%d", openErr.OpenCount, openErr.TotalCount)
		}
		if !strings.Contains(err.Error(), "list_task_items") || !strings.Contains(err.Error(), "update_task_item") {
			t.Errorf("expected guidance to use list_task_items and update_task_item in error: %v", err)
		}
	})

	t.Run("allows completion when all checkboxes are completed", func(t *testing.T) {
		body := `
## Acceptance Criteria
- [x] Item 1
- [x] Item 2
`
		result, err := parser.CompleteTaskBody(body, ts, "Implemented both items.", "Verified with unit tests.")
		if err != nil {
			t.Fatalf("expected successful completion, got: %v", err)
		}

		if !strings.Contains(result, "## Completion Summary") {
			t.Errorf("missing ## Completion Summary: %s", result)
		}
		if !strings.Contains(result, "2026-09-01T12:00:00Z") {
			t.Errorf("missing timestamp: %s", result)
		}
		if !strings.Contains(result, "Implemented both items.") {
			t.Errorf("missing what was done text: %s", result)
		}
		if !strings.Contains(result, "Verified with unit tests.") {
			t.Errorf("missing why rationale text: %s", result)
		}

		// Re-completing replaces the existing summary
		recompleted, err := parser.CompleteTaskBody(result, ts, "Updated work.", "Updated rationale.")
		if err != nil {
			t.Fatalf("re-completion failed: %v", err)
		}
		if strings.Count(recompleted, "## Completion Summary") != 1 {
			t.Errorf("expected 1 Completion Summary header, got: %s", recompleted)
		}
		if !strings.Contains(recompleted, "Updated work.") {
			t.Errorf("missing updated work text: %s", recompleted)
		}
	})

	t.Run("allows completion when task has zero checkboxes", func(t *testing.T) {
		body := `
## Description
Simple task without any checkbox criteria. Just prose.
`
		result, err := parser.CompleteTaskBody(body, ts, "Finished prose task", "No criteria needed")
		if err != nil {
			t.Fatalf("expected completion of 0-checkbox task to succeed, got: %v", err)
		}
		if !strings.Contains(result, "## Completion Summary") {
			t.Errorf("missing completion summary: %s", result)
		}
	})
}

func TestAppendTaskNote(t *testing.T) {
	ts := time.Date(2026, 9, 4, 11, 30, 0, 0, time.UTC)

	// 1. Append note when no ## Notes section exists
	body1 := `# Task Title
## Acceptance Criteria
- [x] Item 1
`
	res1, err := parser.AppendTaskNote(body1, ts, "First note content")
	if err != nil {
		t.Fatalf("failed to append note: %v", err)
	}
	if !strings.Contains(res1, "## Notes") {
		t.Errorf("expected ## Notes header, got:\n%s", res1)
	}
	if !strings.Contains(res1, "### [2026-09-04 11:30 UTC]") {
		t.Errorf("expected timestamp header, got:\n%s", res1)
	}
	if !strings.Contains(res1, "First note content") {
		t.Errorf("expected note content, got:\n%s", res1)
	}

	// 2. Append second note when ## Notes already exists
	ts2 := time.Date(2026, 9, 4, 12, 15, 0, 0, time.UTC)
	res2, err := parser.AppendTaskNote(res1, ts2, "Second note content\n- [ ] Note checkbox item")
	if err != nil {
		t.Fatalf("failed to append second note: %v", err)
	}
	if strings.Count(res2, "## Notes") != 1 {
		t.Errorf("expected exactly 1 ## Notes header, got %d:\n%s", strings.Count(res2, "## Notes"), res2)
	}
	if !strings.Contains(res2, "### [2026-09-04 12:15 UTC]") {
		t.Errorf("expected second timestamp header, got:\n%s", res2)
	}

	// 3. Verify checkbox in notes is extracted as required criteria
	total, completed, items := parser.ExtractAcceptanceCriteria([]byte(res2))
	if total != 2 {
		t.Errorf("expected 2 total criteria (1 in body, 1 in note), got %d", total)
	}
	if completed != 1 {
		t.Errorf("expected 1 completed criterion, got %d", completed)
	}
	if items[1].Text != "Note checkbox item" || items[1].Completed {
		t.Errorf("unexpected note criterion: %+v", items[1])
	}
	if !parser.HasOpenCheckboxes([]byte(res2)) {
		t.Error("expected HasOpenCheckboxes to be true due to open note checkbox")
	}

	// Verify complete_task fails while note checkbox is open
	err = parser.VerifyAllCheckboxesCompleted(res2)
	if err == nil {
		t.Fatal("expected VerifyAllCheckboxesCompleted to fail with open note checkbox")
	}

	// 4. Insertion before ## Completion Summary
	bodyWithSummary := `# Task
## Acceptance Criteria
- [x] Done

## Completion Summary
### Completed At
2026-09-04T10:00:00Z
`
	res3, err := parser.AppendTaskNote(bodyWithSummary, ts, "Post-completion observation")
	if err != nil {
		t.Fatalf("failed to append note before completion summary: %v", err)
	}
	summaryIdx := strings.Index(res3, "## Completion Summary")
	notesIdx := strings.Index(res3, "## Notes")
	if notesIdx == -1 || summaryIdx == -1 || notesIdx >= summaryIdx {
		t.Errorf("expected ## Notes to be before ## Completion Summary, got:\n%s", res3)
	}
}

func TestIsOnlyCheckboxToggle(t *testing.T) {
	orig := `# Title
## Criteria
- [ ] Task item 1
- [x] Task item 2
`
	// Toggling item 1 to checked
	toggled := `# Title
## Criteria
- [x] Task item 1
- [x] Task item 2
`
	if !parser.IsOnlyCheckboxToggle(orig, toggled) {
		t.Error("expected IsOnlyCheckboxToggle to be true for checkbox toggle")
	}

	// Editing text in item 1
	editedText := `# Title
## Criteria
- [ ] Task item 1 edited
- [x] Task item 2
`
	if parser.IsOnlyCheckboxToggle(orig, editedText) {
		t.Error("expected IsOnlyCheckboxToggle to be false when text is edited")
	}

	// Adding a new item
	addedItem := `# Title
## Criteria
- [ ] Task item 1
- [x] Task item 2
- [ ] Task item 3
`
	if parser.IsOnlyCheckboxToggle(orig, addedItem) {
		t.Error("expected IsOnlyCheckboxToggle to be false when item is added")
	}
}

