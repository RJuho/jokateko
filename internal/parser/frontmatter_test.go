package parser_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/RJuho/jokateko/internal/model"
	"github.com/RJuho/jokateko/internal/parser"
)

func TestSplit_TOMLDelim(t *testing.T) {
	doc := `+++
title = "My Task"
summary = "Summary text"
+++

# Heading

Markdown body content.
`
	fm, body, delim, bodyLine, err := parser.Split([]byte(doc))
	if err != nil {
		t.Fatalf("expected successful split, got error: %v", err)
	}

	if delim != parser.DelimTOML {
		t.Errorf("expected delimiter '+++', got %q", delim)
	}

	if !strings.Contains(string(fm), `title = "My Task"`) {
		t.Errorf("unexpected frontmatter: %s", fm)
	}

	if !strings.Contains(string(body), "Markdown body content.") {
		t.Errorf("unexpected body: %s", body)
	}

	if bodyLine <= 1 {
		t.Errorf("expected body line > 1, got %d", bodyLine)
	}
}

func TestSplit_RejectYAMLDelim(t *testing.T) {
	doc := `---
title = "YAML Delim Task"
summary = "Summary text"
---

Body content here.
`
	_, _, _, _, err := parser.Split([]byte(doc))
	if !errors.Is(err, parser.ErrNoFrontmatter) {
		t.Errorf("expected ErrNoFrontmatter when using ---, got: %v", err)
	}
}

func TestSplit_Errors(t *testing.T) {
	t.Run("missing opening delimiter", func(t *testing.T) {
		doc := "Just regular markdown\nwithout delimiters."
		_, _, _, _, err := parser.Split([]byte(doc))
		if !errors.Is(err, parser.ErrNoFrontmatter) {
			t.Errorf("expected ErrNoFrontmatter, got: %v", err)
		}
	})

	t.Run("unclosed delimiter", func(t *testing.T) {
		doc := "+++\ntitle = 'unclosed'\nbody without closing"
		_, _, _, _, err := parser.Split([]byte(doc))
		if !errors.Is(err, parser.ErrUnclosedFrontmatter) {
			t.Errorf("expected ErrUnclosedFrontmatter, got: %v", err)
		}
	})
}

func TestParseTask(t *testing.T) {
	doc := `+++
title = "Setup SQLite Database"
status = "in_progress"
priority = "high"
milestone = "260915-mvp"
tags = ["backend", "database"]
summary = "Implement in-memory SQLite store."
dependencies = ["260901-init"]
+++

## Description
We need pure Go SQLite.

## Acceptance Criteria
- [ ] Uses modernc.org/sqlite
- [ ] No CGO
`
	task, err := parser.ParseTask([]byte(doc), "260902-setup-db")
	if err != nil {
		t.Fatalf("expected successful task parse, got: %v", err)
	}

	if task.ID != "260902-setup-db" {
		t.Errorf("expected ID '260902-setup-db', got %q", task.ID)
	}

	if task.Title != "Setup SQLite Database" {
		t.Errorf("expected title 'Setup SQLite Database', got %q", task.Title)
	}

	if task.Priority != model.PriorityHigh {
		t.Errorf("expected priority 'high', got %q", task.Priority)
	}

	if task.Milestone != "260915-mvp" {
		t.Errorf("expected milestone '260915-mvp', got %q", task.Milestone)
	}

	if len(task.Tags) != 2 || task.Tags[0] != "backend" {
		t.Errorf("unexpected tags: %v", task.Tags)
	}

	if len(task.Dependencies) != 1 || task.Dependencies[0] != "260901-init" {
		t.Errorf("unexpected dependencies: %v", task.Dependencies)
	}

	if !strings.Contains(task.Body, "## Description") {
		t.Errorf("body missing expected description: %s", task.Body)
	}

	t.Run("missing required fields", func(t *testing.T) {
		badDoc := "+++\nsummary = 'no title'\n+++"
		_, err := parser.ParseTask([]byte(badDoc), "t1")
		if err == nil || !strings.Contains(err.Error(), "missing required field: title") {
			t.Errorf("expected missing title error, got: %v", err)
		}

		badDoc2 := "+++\ntitle = 'no summary'\n+++"
		_, err = parser.ParseTask([]byte(badDoc2), "t2")
		if err == nil || !strings.Contains(err.Error(), "missing required field: summary") {
			t.Errorf("expected missing summary error, got: %v", err)
		}
	})
}

func TestParseMilestone(t *testing.T) {
	doc := `+++
title = "MVP Release"
status = "open"
target_date = "2026-09-15"
tags = ["release", "mvp"]
summary = "Core local daemon and live Kanban."
+++

## Goals
Deliver working MVP.
`
	ms, err := parser.ParseMilestone([]byte(doc), "260915-mvp")
	if err != nil {
		t.Fatalf("expected successful milestone parse, got: %v", err)
	}

	if ms.ID != "260915-mvp" {
		t.Errorf("expected ID '260915-mvp', got %q", ms.ID)
	}

	if ms.TargetDate != "2026-09-15" {
		t.Errorf("expected target_date '2026-09-15', got %q", ms.TargetDate)
	}

	if ms.Status != model.MilestoneStatusOpen {
		t.Errorf("expected status 'open', got %q", ms.Status)
	}

	if len(ms.Tags) != 2 {
		t.Errorf("expected 2 tags, got: %v", ms.Tags)
	}
}

func TestParseStrategy(t *testing.T) {
	doc := `+++
title = "Zero CGO Architecture"
tier = 1
tags = ["architecture", "go"]
summary = "All Go code must compile cleanly without CGO."
+++

# Rules
No CGO allowed.
`
	strat, err := parser.ParseStrategy([]byte(doc), "architecture")
	if err != nil {
		t.Fatalf("expected successful strategy parse, got: %v", err)
	}

	if strat.Tier != model.TierCore {
		t.Errorf("expected TierCore (1), got %d", strat.Tier)
	}

	if strat.Title != "Zero CGO Architecture" {
		t.Errorf("expected title 'Zero CGO Architecture', got %q", strat.Title)
	}

	t.Run("invalid tier", func(t *testing.T) {
		badDoc := "+++\ntitle = 'Strat'\ntier = 99\nsummary = 'sum'\n+++"
		_, err := parser.ParseStrategy([]byte(badDoc), "s1")
		if err == nil || !strings.Contains(err.Error(), "invalid tier") {
			t.Errorf("expected invalid tier error, got: %v", err)
		}
	})
}

func TestParseGlossaryTerm(t *testing.T) {
	doc := `+++
title = "Tasks-as-Code"
tags = ["methodology", "core"]
summary = "Actionable tasks are stored as version-controlled Markdown."
+++

# Tasks-as-Code
The concept treats markdown files as absolute source of truth.
`
	term, err := parser.ParseGlossaryTerm([]byte(doc), "tasks-as-code")
	if err != nil {
		t.Fatalf("expected successful glossary parse, got: %v", err)
	}

	if term.Title != "Tasks-as-Code" {
		t.Errorf("expected title 'Tasks-as-Code', got %q", term.Title)
	}

	if len(term.Tags) != 2 {
		t.Errorf("expected 2 tags, got: %v", term.Tags)
	}
}

func TestFormatRoundTrip(t *testing.T) {
	originalTask := model.Task{
		ID:           "260901-test",
		Title:        "Round Trip Task",
		Status:       "ready",
		Priority:     model.PriorityCritical,
		Milestone:    "mvp",
		Tags:         []string{"backend"},
		Summary:      "Round trip testing",
		Dependencies: []string{"dep-1"},
		Body:         "## Details\nTesting format round-trip.",
	}

	formatted, err := parser.Format(originalTask.Frontmatter(), originalTask.Body)
	if err != nil {
		t.Fatalf("failed to format task: %v", err)
	}

	parsedTask, err := parser.ParseTask(formatted, "260901-test")
	if err != nil {
		t.Fatalf("failed to re-parse formatted task: %v", err)
	}

	if parsedTask.Title != originalTask.Title {
		t.Errorf("expected title %q, got %q", originalTask.Title, parsedTask.Title)
	}
	if parsedTask.Priority != originalTask.Priority {
		t.Errorf("expected priority %q, got %q", originalTask.Priority, parsedTask.Priority)
	}
	if parsedTask.Summary != originalTask.Summary {
		t.Errorf("expected summary %q, got %q", originalTask.Summary, parsedTask.Summary)
	}
	if parsedTask.Body != originalTask.Body {
		t.Errorf("expected body %q, got %q", originalTask.Body, parsedTask.Body)
	}
}

func TestNormalizeTimestamp(t *testing.T) {
	cases := []struct {
		input    string
		expected string
		hasErr   bool
	}{
		{"", "", false},
		{"   ", "", false},
		{"2026-09-08", "2026-09-08T00:00:00Z", false},
		{"2026-09-08T15:04", "2026-09-08T15:04:00Z", false},
		{"2026-09-08T15:04:05", "2026-09-08T15:04:05Z", false},
		{"2026-09-08T15:04:05Z", "2026-09-08T15:04:05Z", false},
		{"2026-09-08T15:04:05+02:00", "2026-09-08T13:04:05Z", false},
		{"invalid-date", "", true},
	}

	for _, c := range cases {
		out, err := parser.NormalizeTimestamp(c.input)
		if c.hasErr && err == nil {
			t.Errorf("expected error for input %q, got nil", c.input)
		}
		if !c.hasErr && err != nil {
			t.Errorf("unexpected error for input %q: %v", c.input, err)
		}
		if out != c.expected {
			t.Errorf("for input %q: expected %q, got %q", c.input, c.expected, out)
		}
	}
}

func TestTaskTimestampsParsingAndFallback(t *testing.T) {
	// Case 1: Task with explicit timestamps
	docWithTimestamps := `+++
title = "Task with Timestamps"
summary = "Has created_at, changed_at, target_at"
created_at = "2026-09-04T12:00:00Z"
changed_at = "2026-09-05T14:30:00Z"
target_at = "2026-09-10T00:00:00Z"
+++
# Body
`
	task1, err := parser.ParseTask([]byte(docWithTimestamps), "260904-explicit")
	if err != nil {
		t.Fatalf("failed to parse task with timestamps: %v", err)
	}
	if task1.CreatedAt != "2026-09-04T12:00:00Z" {
		t.Errorf("expected CreatedAt '2026-09-04T12:00:00Z', got %q", task1.CreatedAt)
	}
	if task1.ChangedAt != "2026-09-05T14:30:00Z" {
		t.Errorf("expected ChangedAt '2026-09-05T14:30:00Z', got %q", task1.ChangedAt)
	}
	if task1.TargetAt != "2026-09-10T00:00:00Z" {
		t.Errorf("expected TargetAt '2026-09-10T00:00:00Z', got %q", task1.TargetAt)
	}

	// Case 2: Legacy task without timestamps - fallback to YYMMDD
	docLegacy := `+++
title = "Legacy Task"
summary = "No timestamps in frontmatter"
+++
# Body
`
	task2, err := parser.ParseTask([]byte(docLegacy), "260901-legacy")
	if err != nil {
		t.Fatalf("failed to parse legacy task: %v", err)
	}
	if task2.CreatedAt != "2026-09-01T00:00:00Z" {
		t.Errorf("expected fallback CreatedAt '2026-09-01T00:00:00Z', got %q", task2.CreatedAt)
	}
	if task2.ChangedAt != "2026-09-01T00:00:00Z" {
		t.Errorf("expected fallback ChangedAt '2026-09-01T00:00:00Z', got %q", task2.ChangedAt)
	}
}
