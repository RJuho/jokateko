package validator_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RJuho/jokateko/internal/validator"
)

func createValidWorkspace(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	tasksDir := filepath.Join(dir, ".jokateko", "tasks")
	msDir := filepath.Join(dir, ".jokateko", "milestones")
	stratDir := filepath.Join(dir, ".jokateko", "strategies")
	glossDir := filepath.Join(dir, ".jokateko", "glossary")

	for _, d := range []string{tasksDir, msDir, stratDir, glossDir} {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatalf("failed to create directory: %v", err)
		}
	}

	configTOML := `version = "0"

[server]
host = "127.0.0.1"
port = 8080

[board]
[[board.columns]]
id = "ready"
name = "Ready"
color = "#3b82f6"

[[board.columns]]
id = "done"
name = "Done"
color = "#10b981"

[tags]
enforce_allowed = true
allowed = ["backend", "database", "mvp", "concept", "core"]
`
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(configTOML), 0644); err != nil {
		t.Fatalf("failed to write config.toml: %v", err)
	}

	// Valid milestone
	msContent := `+++
title = "MVP Release"
status = "open"
target_date = "2026-09-15"
summary = "MVP delivery"
tags = ["mvp"]
+++
`
	_ = os.WriteFile(filepath.Join(msDir, "260915-mvp.md"), []byte(msContent), 0644)

	// Valid task 1
	task1Content := `+++
title = "Task One"
status = "ready"
priority = "high"
summary = "Summary of task one"
milestone = "260915-mvp"
tags = ["backend"]
dependencies = []
+++
`
	_ = os.WriteFile(filepath.Join(tasksDir, "260901-task-one.md"), []byte(task1Content), 0644)

	// Valid task 2 depending on task 1
	task2Content := `+++
title = "Task Two"
status = "done"
priority = "medium"
summary = "Summary of task two"
milestone = "260915-mvp"
tags = ["database"]
dependencies = ["260901-task-one"]
+++
`
	_ = os.WriteFile(filepath.Join(tasksDir, "260902-task-two.md"), []byte(task2Content), 0644)

	// Valid strategy
	stratContent := `+++
title = "Zero CGO"
tier = 1
summary = "Pure Go rules"
tags = ["core"]
+++
`
	_ = os.WriteFile(filepath.Join(stratDir, "zero-cgo.md"), []byte(stratContent), 0644)

	// Valid glossary
	glossContent := `+++
title = "Tasks-as-Code"
summary = "Markdown files as tasks"
tags = ["concept"]
+++
`
	_ = os.WriteFile(filepath.Join(glossDir, "tac.md"), []byte(glossContent), 0644)

	return dir
}

func TestValidWorkspace(t *testing.T) {
	wsDir := createValidWorkspace(t)

	res, err := validator.ValidateWorkspace(wsDir)
	if err != nil {
		t.Fatalf("ValidateWorkspace failed: %v", err)
	}

	if res.HasErrors() {
		t.Errorf("expected no errors, got %d:\n%s", res.ErrorCount(), res.FormatReport())
	}

	if res.TaskCount != 2 || res.MilestoneCount != 1 || res.StrategyCount != 1 || res.GlossaryCount != 1 {
		t.Errorf("unexpected counts: %+v", res)
	}

	report := res.FormatReport()
	if !strings.Contains(report, "[OK] Validation successful") {
		t.Errorf("expected [OK] in report, got:\n%s", report)
	}
}

func TestCycleDetection(t *testing.T) {
	t.Run("2-node cycle A -> B -> A", func(t *testing.T) {
		deps := map[string][]string{
			"task-a": {"task-b"},
			"task-b": {"task-a"},
		}
		taskFiles := map[string]string{
			"task-a": "task-a.md",
			"task-b": "task-b.md",
		}

		diags := validator.DetectCycles(deps, taskFiles)
		if len(diags) != 1 {
			t.Fatalf("expected 1 cycle diagnostic, got %d: %+v", len(diags), diags)
		}
		d := diags[0]
		if d.RuleID != "DAG-001" || d.Severity != validator.SeverityError {
			t.Errorf("unexpected diagnostic: %+v", d)
		}
		if !strings.Contains(d.Context[0], "task-a -> task-b -> task-a") {
			t.Errorf("expected cycle path in context, got %q", d.Context[0])
		}
	})

	t.Run("multi-node cycle A -> B -> C -> A", func(t *testing.T) {
		deps := map[string][]string{
			"task-a": {"task-b"},
			"task-b": {"task-c"},
			"task-c": {"task-a"},
		}
		taskFiles := map[string]string{
			"task-a": "task-a.md",
			"task-b": "task-b.md",
			"task-c": "task-c.md",
		}

		diags := validator.DetectCycles(deps, taskFiles)
		if len(diags) != 1 {
			t.Fatalf("expected 1 cycle diagnostic, got %d: %+v", len(diags), diags)
		}
		if !strings.Contains(diags[0].Context[0], "task-a -> task-b -> task-c -> task-a") {
			t.Errorf("expected cycle path in context, got %q", diags[0].Context[0])
		}
	})
}

func TestValidationRules(t *testing.T) {
	wsDir := createValidWorkspace(t)
	tasksDir := filepath.Join(wsDir, ".jokateko", "tasks")

	// TSK-001: filename format warning
	badNameTask := filepath.Join(tasksDir, "invalid-name.md")
	_ = os.WriteFile(badNameTask, []byte(`+++
title = "Bad Name"
status = "ready"
summary = "Summary"
+++
`), 0644)

	// TSK-004: invalid status
	badStatusTask := filepath.Join(tasksDir, "260903-bad-status.md")
	_ = os.WriteFile(badStatusTask, []byte(`+++
title = "Bad Status"
status = "in_review"
summary = "Summary"
+++
`), 0644)

	// TSK-006: dangling milestone
	badMSTask := filepath.Join(tasksDir, "260904-bad-ms.md")
	_ = os.WriteFile(badMSTask, []byte(`+++
title = "Bad Milestone"
status = "ready"
milestone = "non-existent-ms"
summary = "Summary"
+++
`), 0644)

	// TSK-007: dangling dependency
	badDepTask := filepath.Join(tasksDir, "260905-bad-dep.md")
	_ = os.WriteFile(badDepTask, []byte(`+++
title = "Bad Dependency"
status = "ready"
dependencies = ["missing-dep"]
summary = "Summary"
+++
`), 0644)

	// TSK-008: self-dependency
	selfDepTask := filepath.Join(tasksDir, "260906-self-dep.md")
	_ = os.WriteFile(selfDepTask, []byte(`+++
title = "Self Dependency"
status = "ready"
dependencies = ["260906-self-dep"]
summary = "Summary"
+++
`), 0644)

	// TSK-009: unauthorized tag
	unauthorizedTagTask := filepath.Join(tasksDir, "260907-unauth-tag.md")
	_ = os.WriteFile(unauthorizedTagTask, []byte(`+++
title = "Unauthorized Tag"
status = "ready"
tags = ["forbidden-tag"]
summary = "Summary"
+++
`), 0644)

	res, err := validator.ValidateWorkspace(wsDir)
	if err != nil {
		t.Fatalf("ValidateWorkspace failed: %v", err)
	}

	ruleHits := make(map[string]bool)
	for _, d := range res.Diagnostics {
		ruleHits[d.RuleID] = true
	}

	for _, expectedRule := range []string{"TSK-001", "TSK-004", "TSK-006", "TSK-007", "TSK-008", "TSK-009"} {
		if !ruleHits[expectedRule] {
			t.Errorf("expected rule %s to trigger, but it was not reported", expectedRule)
		}
	}
}

func TestGlossaryDuplicateTitle(t *testing.T) {
	wsDir := createValidWorkspace(t)
	glossDir := filepath.Join(wsDir, ".jokateko", "glossary")

	// Create second glossary file with identical term title
	term2 := filepath.Join(glossDir, "tac-duplicate.md")
	_ = os.WriteFile(term2, []byte(`+++
title = "Tasks-as-Code"
summary = "Duplicate definition"
+++
`), 0644)

	res, err := validator.ValidateWorkspace(wsDir)
	if err != nil {
		t.Fatalf("ValidateWorkspace failed: %v", err)
	}

	foundGLS003 := false
	for _, d := range res.Diagnostics {
		if d.RuleID == "GLS-003" {
			foundGLS003 = true
			break
		}
	}
	if !foundGLS003 {
		t.Error("expected GLS-003 duplicate glossary title diagnostic")
	}
}

func TestTaskCustomPriorityValidation(t *testing.T) {
	wsDir := createValidWorkspace(t)
	tasksDir := filepath.Join(wsDir, ".jokateko", "tasks")

	// Task with non-existent priority
	invalidTask := filepath.Join(tasksDir, "260903-invalid-priority.md")
	_ = os.WriteFile(invalidTask, []byte(`+++
title = "Invalid Priority Task"
status = "backlog"
priority = "ultra-urgent"
summary = "Invalid priority testing"
tags = ["database"]
+++
`), 0644)

	res, err := validator.ValidateWorkspace(wsDir)
	if err != nil {
		t.Fatalf("ValidateWorkspace failed: %v", err)
	}

	foundTSK005 := false
	for _, d := range res.Diagnostics {
		if d.RuleID == "TSK-005" {
			foundTSK005 = true
			break
		}
	}
	if !foundTSK005 {
		t.Error("expected TSK-005 invalid priority warning")
	}
}

func TestStrategyCustomTierValidation(t *testing.T) {
	wsDir := t.TempDir()
	jokDir := filepath.Join(wsDir, ".jokateko")
	_ = os.MkdirAll(jokDir, 0755)

	tasksDir := filepath.Join(jokDir, "tasks")
	_ = os.MkdirAll(tasksDir, 0755)
	msDir := filepath.Join(jokDir, "milestones")
	_ = os.MkdirAll(msDir, 0755)
	stratDir := filepath.Join(jokDir, "strategies")
	_ = os.MkdirAll(stratDir, 0755)

	// Valid strategy with tier 1
	validStrat := filepath.Join(stratDir, "core.md")
	_ = os.WriteFile(validStrat, []byte(`+++
title = "Core Guidelines"
tier = 1
summary = "Core rules"
tags = []
+++
`), 0644)

	// Invalid strategy with tier 99
	invalidStrat := filepath.Join(stratDir, "invalid.md")
	_ = os.WriteFile(invalidStrat, []byte(`+++
title = "Invalid Tier Guidelines"
tier = 99
summary = "Invalid rules"
tags = []
+++
`), 0644)

	res, err := validator.ValidateWorkspace(wsDir)
	if err != nil {
		t.Fatalf("ValidateWorkspace failed: %v", err)
	}

	foundSTR003 := false
	for _, d := range res.Diagnostics {
		if d.RuleID == "STR-003" {
			foundSTR003 = true
			break
		}
	}
	if !foundSTR003 {
		t.Error("expected STR-003 invalid tier diagnostic")
	}
}


