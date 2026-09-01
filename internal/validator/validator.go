package validator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/RJuho/jokateko/internal/config"
)

// ValidationResult summarizes the findings of the validation engine.
type ValidationResult struct {
	Diagnostics    []Diagnostic
	TaskCount      int
	MilestoneCount int
	StrategyCount  int
	GlossaryCount  int
	ColumnCount    int
	ConfigFile     string
}

// HasErrors returns true if any diagnostic has SeverityError.
func (r *ValidationResult) HasErrors() bool {
	for _, d := range r.Diagnostics {
		if d.Severity == SeverityError {
			return true
		}
	}
	return false
}

// ErrorCount returns the number of diagnostics with SeverityError.
func (r *ValidationResult) ErrorCount() int {
	var count int
	for _, d := range r.Diagnostics {
		if d.Severity == SeverityError {
			count++
		}
	}
	return count
}

// WarningCount returns the number of diagnostics with SeverityWarning.
func (r *ValidationResult) WarningCount() int {
	var count int
	for _, d := range r.Diagnostics {
		if d.Severity == SeverityWarning {
			count++
		}
	}
	return count
}

// FormatReport creates the formatted compiler-style output string.
func (r *ValidationResult) FormatReport() string {
	var sb strings.Builder

	if r.HasErrors() || r.WarningCount() > 0 {
		errCount := r.ErrorCount()
		warnCount := r.WarningCount()

		errLabel := "error"
		if errCount != 1 {
			errLabel = "errors"
		}
		warnLabel := "warning"
		if warnCount != 1 {
			warnLabel = "warnings"
		}

		if r.HasErrors() {
			fmt.Fprintf(&sb, "[FAIL] Found %d %s and %d %s during validation:\n\n",
				errCount, errLabel, warnCount, warnLabel)
		} else {
			fmt.Fprintf(&sb, "[WARNING] Found %d %s during validation:\n\n",
				warnCount, warnLabel)
		}

		for i, d := range r.Diagnostics {
			if i > 0 {
				sb.WriteString("\n\n")
			}
			sb.WriteString(d.String())
		}

		if r.HasErrors() {
			fmt.Fprintf(&sb, "\n\nValidation failed with %d %s. Exiting with code 1.", errCount, errLabel)
		} else {
			sb.WriteString("\n\nProject passed with warnings. Exiting with code 0.")
		}
		return sb.String()
	}

	sb.WriteString("[OK] Validation successful:\n")
	fmt.Fprintf(&sb, "  ✓ Configuration: %s (%d columns defined)\n", r.ConfigFile, r.ColumnCount)
	fmt.Fprintf(&sb, "  ✓ Tasks: %d files parsed (0 errors, 0 cycle conflicts)\n", r.TaskCount)
	fmt.Fprintf(&sb, "  ✓ Milestones: %d files parsed\n", r.MilestoneCount)
	fmt.Fprintf(&sb, "  ✓ Strategies: %d guidelines validated\n", r.StrategyCount)
	fmt.Fprintf(&sb, "  ✓ Glossary: %d terms validated (0 errors)\n\n", r.GlossaryCount)
	sb.WriteString("Project is healthy. Exiting with code 0.")

	return sb.String()
}

// ValidateWorkspace scans the workspace directory, parses configuration and markdown files,
// checks references, evaluates DAG cycle detection, and aggregates all diagnostics.
func ValidateWorkspace(workspaceDir string) (*ValidationResult, error) {
	res := &ValidationResult{
		ConfigFile: "config.toml",
	}

	// 1. Locate and validate config.toml
	cfgPath := filepath.Join(workspaceDir, "config.toml")
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		cfgPath = filepath.Join(workspaceDir, ".jokateko", "config.toml")
	}

	var cfg *config.Config
	if _, err := os.Stat(cfgPath); err == nil {
		res.ConfigFile = cfgPath
		loadedCfg, loadErr := config.Load(workspaceDir)
		if loadErr != nil {
			res.Diagnostics = append(res.Diagnostics, Diagnostic{
				RuleID:   "CFG-001",
				Severity: SeverityError,
				File:     cfgPath,
				Message:  fmt.Sprintf("Failed to load config: %v", loadErr),
			})
		} else {
			cfg = loadedCfg
		}
	} else {
		// Use default configuration if missing
		cfg = config.Default(workspaceDir)
	}

	if cfg != nil {
		res.ColumnCount = len(cfg.Board.Columns)
		if err := config.Validate(cfg, workspaceDir); err != nil {
			type unwrapMulti interface {
				Unwrap() []error
			}
			if u, ok := err.(unwrapMulti); ok {
				for _, singleErr := range u.Unwrap() {
					msg := singleErr.Error()
					ruleID := "CFG-001"
					if len(msg) >= 7 && msg[7] == ':' {
						ruleID = msg[:7]
					}
					res.Diagnostics = append(res.Diagnostics, Diagnostic{
						RuleID:   ruleID,
						Severity: SeverityError,
						File:     cfgPath,
						Message:  msg,
					})
				}
			} else {
				res.Diagnostics = append(res.Diagnostics, Diagnostic{
					RuleID:   "CFG-001",
					Severity: SeverityError,
					File:     cfgPath,
					Message:  err.Error(),
				})
			}
		}
	}

	// Prepare lookups
	validStatuses := make(map[string]bool)
	if cfg != nil {
		for _, col := range cfg.Board.Columns {
			validStatuses[col.ID] = true
		}
	}

	allowedTags := make(map[string]bool)
	if cfg != nil {
		for _, t := range cfg.Tags.Allowed {
			allowedTags[t] = true
		}
	}

	tasksDir := filepath.Join(workspaceDir, ".jokateko", "tasks")
	msDir := filepath.Join(workspaceDir, ".jokateko", "milestones")
	stratDir := filepath.Join(workspaceDir, ".jokateko", "strategies")
	glossDir := filepath.Join(workspaceDir, ".jokateko", "glossary")

	if cfg != nil {
		if cfg.Paths.Tasks != "" {
			tasksDir = filepath.Join(workspaceDir, cfg.Paths.Tasks)
		}
		if cfg.Paths.Milestones != "" {
			msDir = filepath.Join(workspaceDir, cfg.Paths.Milestones)
		}
		if cfg.Paths.Strategies != "" {
			stratDir = filepath.Join(workspaceDir, cfg.Paths.Strategies)
		}
		if cfg.Paths.Glossary != "" {
			glossDir = filepath.Join(workspaceDir, cfg.Paths.Glossary)
		}
	}

	// 2. Pre-scan entity slugs
	allTaskSlugs := make(map[string]bool)
	taskFiles := make(map[string]string)
	if entries, err := os.ReadDir(tasksDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() && filepath.Ext(e.Name()) == ".md" && !strings.HasPrefix(e.Name(), ".") {
				slug := strings.TrimSuffix(e.Name(), ".md")
				allTaskSlugs[slug] = true
				taskFiles[slug] = filepath.Join(tasksDir, e.Name())
			}
		}
	}

	milestoneSlugs := make(map[string]bool)
	if entries, err := os.ReadDir(msDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() && filepath.Ext(e.Name()) == ".md" && !strings.HasPrefix(e.Name(), ".") {
				slug := strings.TrimSuffix(e.Name(), ".md")
				milestoneSlugs[slug] = true
			}
		}
	}

	// 3. Validate Tasks & Build Dependency Graph
	depsGraph := make(map[string][]string)
	taskCtx := TaskContext{
		Config:         cfg,
		ValidStatuses:  validStatuses,
		AllTaskSlugs:   allTaskSlugs,
		MilestoneSlugs: milestoneSlugs,
		AllowedTags:    allowedTags,
	}

	if entries, err := os.ReadDir(tasksDir); err == nil {
		for _, e := range entries {
			if e.IsDir() || filepath.Ext(e.Name()) != ".md" || strings.HasPrefix(e.Name(), ".") {
				continue
			}
			res.TaskCount++
			path := filepath.Join(tasksDir, e.Name())
			data, err := os.ReadFile(path)
			if err != nil {
				res.Diagnostics = append(res.Diagnostics, Diagnostic{
					RuleID:   "TSK-002",
					Severity: SeverityError,
					File:     path,
					Message:  fmt.Sprintf("Failed to read file: %v", err),
				})
				continue
			}

			task, diags := ValidateTask(path, data, taskCtx)
			res.Diagnostics = append(res.Diagnostics, diags...)
			if task != nil {
				depsGraph[task.ID] = task.Dependencies
			}
		}
	}

	// 4. Check DAG Cycles (DAG-001)
	cycleDiags := DetectCycles(depsGraph, taskFiles)
	res.Diagnostics = append(res.Diagnostics, cycleDiags...)

	// 5. Validate Milestones
	if entries, err := os.ReadDir(msDir); err == nil {
		for _, e := range entries {
			if e.IsDir() || filepath.Ext(e.Name()) != ".md" || strings.HasPrefix(e.Name(), ".") {
				continue
			}
			res.MilestoneCount++
			path := filepath.Join(msDir, e.Name())
			data, err := os.ReadFile(path)
			if err != nil {
				res.Diagnostics = append(res.Diagnostics, Diagnostic{
					RuleID:   "MLS-002",
					Severity: SeverityError,
					File:     path,
					Message:  fmt.Sprintf("Failed to read file: %v", err),
				})
				continue
			}

			_, diags := ValidateMilestone(path, data, cfg, allowedTags)
			res.Diagnostics = append(res.Diagnostics, diags...)
		}
	}

	// 6. Validate Strategies
	if entries, err := os.ReadDir(stratDir); err == nil {
		for _, e := range entries {
			if e.IsDir() || filepath.Ext(e.Name()) != ".md" || strings.HasPrefix(e.Name(), ".") {
				continue
			}
			res.StrategyCount++
			path := filepath.Join(stratDir, e.Name())
			data, err := os.ReadFile(path)
			if err != nil {
				res.Diagnostics = append(res.Diagnostics, Diagnostic{
					RuleID:   "STR-001",
					Severity: SeverityError,
					File:     path,
					Message:  fmt.Sprintf("Failed to read file: %v", err),
				})
				continue
			}

			_, diags := ValidateStrategy(path, data, cfg, allowedTags)
			res.Diagnostics = append(res.Diagnostics, diags...)
		}
	}

	// 7. Validate Glossary Terms (and check GLS-003 uniqueness)
	glossaryTitles := make(map[string]string) // title -> file path
	if entries, err := os.ReadDir(glossDir); err == nil {
		for _, e := range entries {
			if e.IsDir() || filepath.Ext(e.Name()) != ".md" || strings.HasPrefix(e.Name(), ".") {
				continue
			}
			res.GlossaryCount++
			path := filepath.Join(glossDir, e.Name())
			data, err := os.ReadFile(path)
			if err != nil {
				res.Diagnostics = append(res.Diagnostics, Diagnostic{
					RuleID:   "GLS-001",
					Severity: SeverityError,
					File:     path,
					Message:  fmt.Sprintf("Failed to read file: %v", err),
				})
				continue
			}

			term, diags := ValidateGlossaryTerm(path, data, cfg, allowedTags)
			res.Diagnostics = append(res.Diagnostics, diags...)

			if term != nil && term.Title != "" {
				lowerTitle := strings.ToLower(strings.TrimSpace(term.Title))
				if existingFile, ok := glossaryTitles[lowerTitle]; ok {
					res.Diagnostics = append(res.Diagnostics, Diagnostic{
						RuleID:   "GLS-003",
						Severity: SeverityError,
						File:     path,
						Message:  fmt.Sprintf("Duplicate glossary term title %q (already defined in %s)", term.Title, existingFile),
						Fix:      "Rename the term title so every glossary definition is unique.",
					})
				} else {
					glossaryTitles[lowerTitle] = path
				}
			}
		}
	}

	return res, nil
}
