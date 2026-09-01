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
`
	total, completed, items := parser.ExtractAcceptanceCriteria([]byte(md))
	if total != 4 {
		t.Errorf("expected 4 total criteria, got %d", total)
	}
	if completed != 2 {
		t.Errorf("expected 2 completed criteria, got %d", completed)
	}
	if len(items) != 4 {
		t.Fatalf("expected 4 criteria items, got %d", len(items))
	}

	if items[0].Index != 1 || items[0].Text != "Pure Go SQLite (completed)" || !items[0].Completed {
		t.Errorf("unexpected first criterion: %+v", items[0])
	}
	if items[1].Index != 2 || items[1].Text != "In-memory support (pending)" || items[1].Completed {
		t.Errorf("unexpected second criterion: %+v", items[1])
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
