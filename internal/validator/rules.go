package validator

import (
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/RJuho/jokateko/internal/config"
	"github.com/RJuho/jokateko/internal/model"
	"github.com/RJuho/jokateko/internal/parser"
)

var (
	dateFilenameRegex = regexp.MustCompile(`^[0-9]{6}-[a-z0-9-]+\.md$`)
)

// TaskContext provides surrounding project context for validating a task.
type TaskContext struct {
	Config         *config.Config
	ValidStatuses  map[string]bool
	AllTaskSlugs   map[string]bool
	MilestoneSlugs map[string]bool
	AllowedTags    map[string]bool
}

// ValidateTask checks all TSK-xxx rules for a given task file.
func ValidateTask(path string, content []byte, ctx TaskContext) (*model.Task, []Diagnostic) {
	var diags []Diagnostic
	baseName := filepath.Base(path)
	slug := strings.TrimSuffix(baseName, ".md")

	// TSK-001: filename-format (Warning)
	if !dateFilenameRegex.MatchString(baseName) {
		diags = append(diags, Diagnostic{
			RuleID:   "TSK-001",
			Severity: SeverityWarning,
			File:     path,
			Message:  "Filename does not follow standard date convention.",
			Fix:      fmt.Sprintf("Rename file to 'YYMMDD-%s' (e.g. '%s-%s').", baseName, time.Now().Format("060102"), baseName),
		})
	}

	// TSK-002: toml-frontmatter (Error)
	fmBytes, bodyBytes, delim, bodyLine, splitErr := parser.Split(content)
	if splitErr != nil {
		diags = append(diags, Diagnostic{
			RuleID:   "TSK-002",
			Severity: SeverityError,
			File:     path,
			Message:  fmt.Sprintf("Invalid frontmatter delimiter: %v", splitErr),
			Fix:      "Enclose frontmatter in '+++' delimiters.",
		})
		return nil, diags
	}

	if delim != "+++" {
		diags = append(diags, Diagnostic{
			RuleID:   "TSK-002",
			Severity: SeverityError,
			File:     path,
			Line:     1,
			Message:  "Frontmatter must use '+++' delimiters.",
			Fix:      "Replace frontmatter delimiters with '+++'.",
		})
		return nil, diags
	}

	task, parseErr := parser.ParseTaskWithCriteria(content, slug)
	if parseErr != nil {
		diags = append(diags, Diagnostic{
			RuleID:   "TSK-002",
			Severity: SeverityError,
			File:     path,
			Line:     bodyLine,
			Message:  fmt.Sprintf("Failed to parse TOML frontmatter: %v", parseErr),
			Fix:      "Ensure frontmatter conforms to valid TOML.",
		})
		return nil, diags
	}

	// TSK-003: required-fields (Error)
	if strings.TrimSpace(task.Title) == "" {
		diags = append(diags, Diagnostic{
			RuleID:   "TSK-003",
			Severity: SeverityError,
			File:     path,
			Message:  "Task frontmatter missing required field: title",
			Fix:      "Add 'title = \"...\"' to frontmatter.",
		})
	}
	if strings.TrimSpace(task.Summary) == "" {
		diags = append(diags, Diagnostic{
			RuleID:   "TSK-003",
			Severity: SeverityError,
			File:     path,
			Message:  "Task frontmatter missing required field: summary",
			Fix:      "Add 'summary = \"...\"' to frontmatter.",
		})
	}

	// TSK-004: valid-status (Error)
	if task.Status == "" || (ctx.ValidStatuses != nil && !ctx.ValidStatuses[task.Status]) {
		var allowed []string
		for s := range ctx.ValidStatuses {
			allowed = append(allowed, s)
		}
		slices.Sort(allowed)

		diags = append(diags, Diagnostic{
			RuleID:   "TSK-004",
			Severity: SeverityError,
			File:     path,
			Message:  fmt.Sprintf("Invalid status: %q", task.Status),
			Context: []string{
				fmt.Sprintf("Allowed column statuses: [%s]", strings.Join(allowed, ", ")),
			},
			Fix: fmt.Sprintf("Update 'status = %q' to one of the defined board columns.", task.Status),
		})
	}

	// TSK-005: valid-priority (Warning)
	if task.Priority != "" {
		validPriority := false
		if ctx.Config != nil && len(ctx.Config.Priorities) > 0 {
			validPriority = ctx.Config.HasPriority(string(task.Priority))
		} else {
			validPriority = task.Priority.IsValid()
		}
		if !validPriority {
			diags = append(diags, Diagnostic{
				RuleID:   "TSK-005",
				Severity: SeverityWarning,
				File:     path,
				Message:  fmt.Sprintf("Invalid priority: %q", task.Priority),
				Fix:      "Priority must match one of the configured priorities.",
			})
		}
	}

	// TSK-006: milestone-exists (Error)
	if task.Milestone != "" && ctx.MilestoneSlugs != nil && !ctx.MilestoneSlugs[task.Milestone] {
		diags = append(diags, Diagnostic{
			RuleID:   "TSK-006",
			Severity: SeverityError,
			File:     path,
			Message:  fmt.Sprintf("Referenced milestone does not exist: %q", task.Milestone),
			Fix:      "Create the milestone file or update the 'milestone' field to an existing milestone slug.",
		})
	}

	// TSK-007 & TSK-008: dependencies (Error)
	for _, dep := range task.Dependencies {
		if dep == slug {
			diags = append(diags, Diagnostic{
				RuleID:   "TSK-008",
				Severity: SeverityError,
				File:     path,
				Message:  fmt.Sprintf("Self-dependency detected: task %q lists itself in dependencies", slug),
				Fix:      "Remove the self-referencing slug from 'dependencies'.",
			})
		} else if ctx.AllTaskSlugs != nil && !ctx.AllTaskSlugs[dep] {
			diags = append(diags, Diagnostic{
				RuleID:   "TSK-007",
				Severity: SeverityError,
				File:     path,
				Message:  fmt.Sprintf("Referenced dependency does not exist: %q", dep),
				Fix:      "Ensure the referenced dependency task file exists in .jokateko/tasks/.",
			})
		}
	}

	// TSK-009: allowed-tags (Error)
	if ctx.Config != nil && ctx.Config.Tags.EnforceAllowed {
		for _, tag := range task.Tags {
			if !ctx.AllowedTags[tag] {
				diags = append(diags, Diagnostic{
					RuleID:   "TSK-009",
					Severity: SeverityError,
					File:     path,
					Message:  fmt.Sprintf("Tag %q is not in the allowed tags list", tag),
					Fix:      fmt.Sprintf("Add %q to [tags.allowed] in config.toml or remove it from the task.", tag),
				})
			}
		}
	}

	_ = fmBytes
	_ = bodyBytes
	return task, diags
}

// ValidateMilestone checks MLS-xxx rules for a given milestone file.
func ValidateMilestone(path string, content []byte, cfg *config.Config, allowedTags map[string]bool) (*model.Milestone, []Diagnostic) {
	var diags []Diagnostic
	baseName := filepath.Base(path)
	slug := strings.TrimSuffix(baseName, ".md")

	// MLS-001: filename-format (Warning)
	if !dateFilenameRegex.MatchString(baseName) {
		diags = append(diags, Diagnostic{
			RuleID:   "MLS-001",
			Severity: SeverityWarning,
			File:     path,
			Message:  "Filename does not follow standard date convention.",
			Fix:      fmt.Sprintf("Rename file to 'YYMMDD-%s'.", baseName),
		})
	}

	// MLS-002: toml-frontmatter (Error)
	ms, err := parser.ParseMilestone(content, slug)
	if err != nil {
		diags = append(diags, Diagnostic{
			RuleID:   "MLS-002",
			Severity: SeverityError,
			File:     path,
			Message:  fmt.Sprintf("Invalid milestone frontmatter: %v", err),
			Fix:      "Ensure milestone uses '+++' delimiters and valid TOML.",
		})
		return nil, diags
	}

	// MLS-003: required-fields (Error)
	if strings.TrimSpace(ms.Title) == "" {
		diags = append(diags, Diagnostic{
			RuleID:   "MLS-003",
			Severity: SeverityError,
			File:     path,
			Message:  "Milestone frontmatter missing required field: title",
			Fix:      "Add 'title = \"...\"' to frontmatter.",
		})
	}
	if strings.TrimSpace(ms.Summary) == "" {
		diags = append(diags, Diagnostic{
			RuleID:   "MLS-003",
			Severity: SeverityError,
			File:     path,
			Message:  "Milestone frontmatter missing required field: summary",
			Fix:      "Add 'summary = \"...\"' to frontmatter.",
		})
	}

	// MLS-004: date-format (Warning)
	if ms.TargetDate != "" {
		if _, err := time.Parse("2006-01-02", ms.TargetDate); err != nil {
			diags = append(diags, Diagnostic{
				RuleID:   "MLS-004",
				Severity: SeverityWarning,
				File:     path,
				Message:  fmt.Sprintf("Invalid target_date format: %q", ms.TargetDate),
				Fix:      "Format target_date as 'YYYY-MM-DD' (ISO-8601).",
			})
		}
	}

	// MLS-005: allowed-tags (Error)
	if cfg != nil && cfg.Tags.EnforceAllowed {
		for _, tag := range ms.Tags {
			if !allowedTags[tag] {
				diags = append(diags, Diagnostic{
					RuleID:   "MLS-005",
					Severity: SeverityError,
					File:     path,
					Message:  fmt.Sprintf("Tag %q is not in the allowed tags list", tag),
					Fix:      fmt.Sprintf("Add %q to [tags.allowed] in config.toml or remove it from the milestone.", tag),
				})
			}
		}
	}

	return ms, diags
}

// ValidateStrategy checks STR-xxx rules for a given strategy file.
func ValidateStrategy(path string, content []byte, cfg *config.Config, allowedTags map[string]bool) (*model.Strategy, []Diagnostic) {
	var diags []Diagnostic
	baseName := filepath.Base(path)
	slug := strings.TrimSuffix(baseName, ".md")

	strat, err := parser.ParseStrategy(content, slug)
	if err != nil {
		diags = append(diags, Diagnostic{
			RuleID:   "STR-001",
			Severity: SeverityError,
			File:     path,
			Message:  fmt.Sprintf("Invalid strategy frontmatter: %v", err),
			Fix:      "Ensure strategy uses '+++' delimiters and valid TOML.",
		})
		return nil, diags
	}

	// STR-002: required-fields (Error)
	if strings.TrimSpace(strat.Title) == "" {
		diags = append(diags, Diagnostic{
			RuleID:   "STR-002",
			Severity: SeverityError,
			File:     path,
			Message:  "Strategy frontmatter missing required field: title",
			Fix:      "Add 'title = \"...\"' to frontmatter.",
		})
	}
	if strings.TrimSpace(strat.Summary) == "" {
		diags = append(diags, Diagnostic{
			RuleID:   "STR-002",
			Severity: SeverityError,
			File:     path,
			Message:  "Strategy frontmatter missing required field: summary",
			Fix:      "Add 'summary = \"...\"' to frontmatter.",
		})
	}

	// STR-003: valid-tier (Error)
	if !strat.Tier.IsValid() {
		diags = append(diags, Diagnostic{
			RuleID:   "STR-003",
			Severity: SeverityError,
			File:     path,
			Message:  fmt.Sprintf("Invalid strategy tier: %d", strat.Tier),
			Fix:      "Set 'tier' to 1 (Core), 2 (Domain), or 3 (Implementation).",
		})
	}

	// STR-004: allowed-tags (Error)
	if cfg != nil && cfg.Tags.EnforceAllowed {
		for _, tag := range strat.Tags {
			if !allowedTags[tag] {
				diags = append(diags, Diagnostic{
					RuleID:   "STR-004",
					Severity: SeverityError,
					File:     path,
					Message:  fmt.Sprintf("Tag %q is not in the allowed tags list", tag),
					Fix:      fmt.Sprintf("Add %q to [tags.allowed] in config.toml or remove it from the strategy.", tag),
				})
			}
		}
	}

	return strat, diags
}

// ValidateGlossaryTerm checks GLS-xxx rules for a given glossary term file.
func ValidateGlossaryTerm(path string, content []byte, cfg *config.Config, allowedTags map[string]bool) (*model.GlossaryTerm, []Diagnostic) {
	var diags []Diagnostic
	baseName := filepath.Base(path)
	slug := strings.TrimSuffix(baseName, ".md")

	term, err := parser.ParseGlossaryTerm(content, slug)
	if err != nil {
		diags = append(diags, Diagnostic{
			RuleID:   "GLS-001",
			Severity: SeverityError,
			File:     path,
			Message:  fmt.Sprintf("Invalid glossary term frontmatter: %v", err),
			Fix:      "Ensure glossary term uses '+++' delimiters and valid TOML.",
		})
		return nil, diags
	}

	// GLS-002: required-fields (Error)
	if strings.TrimSpace(term.Title) == "" {
		diags = append(diags, Diagnostic{
			RuleID:   "GLS-002",
			Severity: SeverityError,
			File:     path,
			Message:  "Glossary frontmatter missing required field: title",
			Fix:      "Add 'title = \"...\"' to frontmatter.",
		})
	}
	if strings.TrimSpace(term.Summary) == "" {
		diags = append(diags, Diagnostic{
			RuleID:   "GLS-002",
			Severity: SeverityError,
			File:     path,
			Message:  "Glossary frontmatter missing required field: summary",
			Fix:      "Add 'summary = \"...\"' to frontmatter.",
		})
	}

	// GLS-004: allowed-tags (Error)
	if cfg != nil && cfg.Tags.EnforceAllowed {
		for _, tag := range term.Tags {
			if !allowedTags[tag] {
				diags = append(diags, Diagnostic{
					RuleID:   "GLS-004",
					Severity: SeverityError,
					File:     path,
					Message:  fmt.Sprintf("Tag %q is not in the allowed tags list", tag),
					Fix:      fmt.Sprintf("Add %q to [tags.allowed] in config.toml or remove it from the glossary term.", tag),
				})
			}
		}
	}

	return term, diags
}
