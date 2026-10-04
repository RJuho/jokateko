package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/RJuho/jokateko/internal/config"
)

func cmdInit(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dirFlag := fs.String("dir", ".", "Directory in which to initialize Jokateko")
	nameFlag := fs.String("name", "", "Project name (default: directory name)")
	replaceFlag := fs.Bool("replace", false, "Overwrite existing config.toml with fresh default template")

	if err := fs.Parse(args); err != nil {
		return 1
	}

	targetDir, err := filepath.Abs(*dirFlag)
	if err != nil {
		fmt.Fprintf(stderr, "failed to resolve directory %q: %v\n", *dirFlag, err)
		return 1
	}

	projectName := *nameFlag
	if projectName == "" {
		projectName = filepath.Base(targetDir)
	}

	jokatekoDir := filepath.Join(targetDir, ".jokateko")
	tasksDir := filepath.Join(jokatekoDir, "tasks")
	msDir := filepath.Join(jokatekoDir, "milestones")
	stratDir := filepath.Join(jokatekoDir, "strategies")
	glossDir := filepath.Join(jokatekoDir, "glossary")
	tmplDir := filepath.Join(jokatekoDir, "templates")

	// 1. Create directory structure
	for _, d := range []string{tasksDir, msDir, stratDir, glossDir, tmplDir} {
		if err := os.MkdirAll(d, 0755); err != nil {
			fmt.Fprintf(stderr, "failed to create directory %q: %v\n", d, err)
			return 1
		}
	}

	// 2. Default config.toml
	configFile := filepath.Join(jokatekoDir, "config.toml")
	if _, err := os.Stat(configFile); errors.Is(err, os.ErrNotExist) || *replaceFlag {
		commentedConfig := config.GenerateCommentedConfig(projectName, "Local, Markdown-driven Kanban and task management")
		if err := os.WriteFile(configFile, []byte(commentedConfig), 0644); err != nil {
			fmt.Fprintf(stderr, "failed to write config.toml: %v\n", err)
			return 1
		}
	}

	// 3. Write initial starter files if folders are empty
	today := time.Now().Format("060102")

	// Sample Strategy
	stratFile := filepath.Join(stratDir, "architecture.md")
	if _, err := os.Stat(stratFile); errors.Is(err, os.ErrNotExist) {
		stratContent := `+++
title = "Tasks-as-Code Architecture"
tier = 1
summary = "Core architectural principles: repository markdown files as the source of truth"
tags = ["docs"]
+++

# Tasks-as-Code Architecture

All tasks, milestones, and architectural strategies are stored as versioned Markdown files directly in the repository.
`
		_ = os.WriteFile(stratFile, []byte(stratContent), 0644)
	}

	// Sample Glossary
	glossFile := filepath.Join(glossDir, "tasks-as-code.md")
	if _, err := os.Stat(glossFile); errors.Is(err, os.ErrNotExist) {
		glossContent := `+++
title = "Tasks-as-Code"
summary = "A methodology where tasks and project specs are version-controlled alongside source code"
tags = ["docs"]
+++

# Tasks-as-Code

Tasks-as-Code treats work items and specifications as code artifacts with pull request reviews, automated linting, and offline access.
`
		_ = os.WriteFile(glossFile, []byte(glossContent), 0644)
	}

	// Sample Milestone
	msSlug := fmt.Sprintf("%s-mvp", today)
	msFile := filepath.Join(msDir, fmt.Sprintf("%s.md", msSlug))
	if _, err := os.Stat(msFile); errors.Is(err, os.ErrNotExist) {
		targetDate := time.Now().AddDate(0, 1, 0).Format("2006-01-02")
		msContent := fmt.Sprintf(`+++
title = "MVP Release"
status = "open"
target_date = %q
summary = "Initial usable release of the project"
tags = ["release"]
+++

# MVP Release

Deliver the core foundational milestones for the initial usable release.
`, targetDate)
		_ = os.WriteFile(msFile, []byte(msContent), 0644)
	}

	// Sample Task
	taskFile := filepath.Join(tasksDir, fmt.Sprintf("%s-initial-setup.md", today))
	if _, err := os.Stat(taskFile); errors.Is(err, os.ErrNotExist) {
		taskContent := fmt.Sprintf(`+++
title = "Initial Project Setup"
status = "ready"
priority = "high"
summary = "Verify Jokateko project configuration and task workflow"
milestone = %q
tags = ["release"]
dependencies = []
+++

# Initial Project Setup

Verify that all project directories and configurations are ready.

## Acceptance Criteria
- [ ] Review .jokateko/config.toml board columns
- [ ] Run 'jokateko parse' to validate project state
`, msSlug)
		_ = os.WriteFile(taskFile, []byte(taskContent), 0644)
	}

	fmt.Fprintf(stdout, `[OK] Initialized Jokateko project in %s
  ✓ Created directory structure (.jokateko/{tasks,milestones,strategies,glossary,templates})
  ✓ Created default configuration (.jokateko/config.toml)
  ✓ Added starter milestone and task (%s-initial-setup.md)
Run 'jokateko parse' to validate your project state.
`, jokatekoDir, today)

	return 0
}
