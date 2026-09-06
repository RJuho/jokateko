package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/RJuho/jokateko/internal/config"
	"github.com/RJuho/jokateko/internal/model"
)

func TestDefaultConfig(t *testing.T) {
	cfg := config.Default("/tmp/my-project")

	if cfg.Version != config.CurrentVersion {
		t.Errorf("expected version %q, got %q", config.CurrentVersion, cfg.Version)
	}

	if cfg.Project.Name != "my-project" {
		t.Errorf("expected project name 'my-project', got %q", cfg.Project.Name)
	}

	if len(cfg.Board.Columns) != 5 {
		t.Errorf("expected 5 default columns, got %d", len(cfg.Board.Columns))
	}

	if len(cfg.Tags.Allowed) != 11 {
		t.Errorf("expected 11 default tags, got %d", len(cfg.Tags.Allowed))
	}

	if !cfg.Tags.EnforceAllowed {
		t.Error("expected tags.enforce_allowed to be true by default")
	}

	if cfg.Server.Port != 8080 {
		t.Errorf("expected port 8080, got %d", cfg.Server.Port)
	}

	if err := config.Validate(cfg, "/tmp/my-project"); err != nil {
		t.Fatalf("default config failed validation: %v", err)
	}

	if !cfg.HasColumn("backlog") {
		t.Error("expected HasColumn('backlog') to be true")
	}

	if cfg.HasColumn("nonexistent") {
		t.Error("expected HasColumn('nonexistent') to be false")
	}

	if len(cfg.Priorities) != 4 {
		t.Errorf("expected 4 default priorities, got %d", len(cfg.Priorities))
	}

	if !cfg.HasPriority("critical") || !cfg.HasPriority("low") {
		t.Error("expected HasPriority('critical') and HasPriority('low') to be true")
	}

	if cfg.HasPriority("nonexistent") {
		t.Error("expected HasPriority('nonexistent') to be false")
	}

	if !cfg.IsAllowedTag("backend") {
		t.Error("expected IsAllowedTag('backend') to be true")
	}

	if cfg.IsAllowedTag("invalid-tag-foo") {
		t.Error("expected IsAllowedTag('invalid-tag-foo') to be false")
	}

	if cfg.MCP.Timeout() != 30*time.Second {
		t.Errorf("expected default timeout 30s, got %v", cfg.MCP.Timeout())
	}

	if cfg.Translations.SortBy != "Sort" {
		t.Errorf("expected default Translations.SortBy 'Sort', got %q", cfg.Translations.SortBy)
	}
	if cfg.Translations.SortDefault != "Default (Workflow)" {
		t.Errorf("expected default Translations.SortDefault 'Default (Workflow)', got %q", cfg.Translations.SortDefault)
	}
	if cfg.Translations.ArialSortTasks != "Sort tasks" {
		t.Errorf("expected default Translations.ArialSortTasks 'Sort tasks', got %q", cfg.Translations.ArialSortTasks)
	}
}

func TestLoadMissingFileFallback(t *testing.T) {
	tempDir := t.TempDir()

	cfg, err := config.Load(tempDir)
	if err != nil {
		t.Fatalf("expected graceful fallback on missing config, got error: %v", err)
	}

	if cfg.Version != config.CurrentVersion {
		t.Errorf("expected version %q, got %q", config.CurrentVersion, cfg.Version)
	}
}

func TestLoadValidTOMLFile(t *testing.T) {
	tempDir := t.TempDir()
	jokatekoDir := filepath.Join(tempDir, ".jokateko")
	if err := os.MkdirAll(jokatekoDir, 0755); err != nil {
		t.Fatal(err)
	}

	tomlContent := `
version = "0"

[project]
name = "Custom App"
description = "Custom description"

[server]
host = "0.0.0.0"
port = 9090

[[board.columns]]
id = "todo"
name = "To Do"
color = "#3b82f6"

[[board.columns]]
id = "done"
name = "Done"
color = "#10b981"

[tags]
allowed = ["backend", "frontend"]
enforce_allowed = true
`
	if err := os.WriteFile(filepath.Join(jokatekoDir, "config.toml"), []byte(tomlContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(tempDir)
	if err != nil {
		t.Fatalf("failed to load valid config: %v", err)
	}

	if cfg.Project.Name != "Custom App" {
		t.Errorf("expected 'Custom App', got %q", cfg.Project.Name)
	}

	if cfg.Server.Port != 9090 {
		t.Errorf("expected port 9090, got %d", cfg.Server.Port)
	}

	if len(cfg.Board.Columns) != 2 {
		t.Fatalf("expected 2 columns, got %d", len(cfg.Board.Columns))
	}
}

func TestLoadPartialProjectNameOnly(t *testing.T) {
	tempDir := t.TempDir()
	jokatekoDir := filepath.Join(tempDir, ".jokateko")
	if err := os.MkdirAll(jokatekoDir, 0755); err != nil {
		t.Fatal(err)
	}

	tomlContent := `
[project]
name = "Partial Project Name Only"
`
	if err := os.WriteFile(filepath.Join(jokatekoDir, "config.toml"), []byte(tomlContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(tempDir)
	if err != nil {
		t.Fatalf("expected successful load of partial config, got: %v", err)
	}

	if cfg.Project.Name != "Partial Project Name Only" {
		t.Errorf("expected project name 'Partial Project Name Only', got %q", cfg.Project.Name)
	}

	// Verify all omitted sections and fields retain standard defaults
	if cfg.Project.Description != "Markdown-driven task management and Kanban" {
		t.Errorf("expected default description, got %q", cfg.Project.Description)
	}

	if cfg.Version != config.CurrentVersion {
		t.Errorf("expected default version %q, got %q", config.CurrentVersion, cfg.Version)
	}

	if cfg.Server.Port != 8080 {
		t.Errorf("expected default port 8080, got %d", cfg.Server.Port)
	}

	if len(cfg.Board.Columns) != 5 {
		t.Errorf("expected default 5 columns, got %d", len(cfg.Board.Columns))
	}

	if len(cfg.Tags.Allowed) != 11 {
		t.Errorf("expected default 11 tags, got %d", len(cfg.Tags.Allowed))
	}

	if cfg.Paths.Tasks != filepath.Join(".jokateko", "tasks") {
		t.Errorf("expected default tasks path, got %q", cfg.Paths.Tasks)
	}
}

func TestLoadPartialProjectDescriptionOnly(t *testing.T) {
	tempDir := t.TempDir()
	jokatekoDir := filepath.Join(tempDir, ".jokateko")
	if err := os.MkdirAll(jokatekoDir, 0755); err != nil {
		t.Fatal(err)
	}

	tomlContent := `
[project]
description = "My Custom Description Only"
`
	if err := os.WriteFile(filepath.Join(jokatekoDir, "config.toml"), []byte(tomlContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(tempDir)
	if err != nil {
		t.Fatalf("expected successful load, got: %v", err)
	}

	if cfg.Project.Description != "My Custom Description Only" {
		t.Errorf("expected custom description, got %q", cfg.Project.Description)
	}

	// Project name defaults to directory basename
	expectedName := filepath.Base(tempDir)
	if cfg.Project.Name != expectedName {
		t.Errorf("expected default project name %q, got %q", expectedName, cfg.Project.Name)
	}

	if len(cfg.Board.Columns) != 5 {
		t.Errorf("expected default 5 columns, got %d", len(cfg.Board.Columns))
	}
}

func TestLoadPartialTopLevelNameAlias(t *testing.T) {
	tempDir := t.TempDir()
	jokatekoDir := filepath.Join(tempDir, ".jokateko")
	if err := os.MkdirAll(jokatekoDir, 0755); err != nil {
		t.Fatal(err)
	}

	tomlContent := `
name = "Top Level Name App"
`
	if err := os.WriteFile(filepath.Join(jokatekoDir, "config.toml"), []byte(tomlContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(tempDir)
	if err != nil {
		t.Fatalf("expected successful load, got: %v", err)
	}

	if cfg.Project.Name != "Top Level Name App" {
		t.Errorf("expected 'Top Level Name App', got %q", cfg.Project.Name)
	}

	if cfg.Project.Description != "Markdown-driven task management and Kanban" {
		t.Errorf("expected default description, got %q", cfg.Project.Description)
	}
}

func TestLoadPartialServerPortOnly(t *testing.T) {
	tempDir := t.TempDir()
	jokatekoDir := filepath.Join(tempDir, ".jokateko")
	if err := os.MkdirAll(jokatekoDir, 0755); err != nil {
		t.Fatal(err)
	}

	tomlContent := `
[server]
port = 9999
`
	if err := os.WriteFile(filepath.Join(jokatekoDir, "config.toml"), []byte(tomlContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(tempDir)
	if err != nil {
		t.Fatalf("expected successful load, got: %v", err)
	}

	if cfg.Server.Port != 9999 {
		t.Errorf("expected custom port 9999, got %d", cfg.Server.Port)
	}

	if cfg.Server.Host != "127.0.0.1" {
		t.Errorf("expected default host '127.0.0.1', got %q", cfg.Server.Host)
	}

	if len(cfg.Board.Columns) != 5 {
		t.Errorf("expected default 5 columns, got %d", len(cfg.Board.Columns))
	}
}

func TestLoadEmptyTOMLFile(t *testing.T) {
	tempDir := t.TempDir()
	jokatekoDir := filepath.Join(tempDir, ".jokateko")
	if err := os.MkdirAll(jokatekoDir, 0755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(jokatekoDir, "config.toml"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(tempDir)
	if err != nil {
		t.Fatalf("expected empty TOML to succeed with defaults, got: %v", err)
	}

	if cfg.Version != config.CurrentVersion {
		t.Errorf("expected default version %q, got %q", config.CurrentVersion, cfg.Version)
	}

	if cfg.Server.Port != 8080 {
		t.Errorf("expected default port 8080, got %d", cfg.Server.Port)
	}
}


func TestEnvironmentOverrides(t *testing.T) {
	tempDir := t.TempDir()

	t.Setenv(config.EnvHost, "0.0.0.0")
	t.Setenv(config.EnvPort, "8888")

	cfg, err := config.Load(tempDir)
	if err != nil {
		t.Fatalf("failed to load config with env overrides: %v", err)
	}

	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("expected host '0.0.0.0', got %q", cfg.Server.Host)
	}

	if cfg.Server.Port != 8888 {
		t.Errorf("expected port 8888, got %d", cfg.Server.Port)
	}
}

func TestValidationRules(t *testing.T) {
	root := "/workspace"

	t.Run("CFG-000: Invalid Version", func(t *testing.T) {
		cfg := config.Default(root)
		cfg.Version = "999"
		err := config.Validate(cfg, root)
		if err == nil || !strings.Contains(err.Error(), "CFG-000") {
			t.Errorf("expected CFG-000 error, got: %v", err)
		}
	})

	t.Run("CFG-002: Less than 2 columns", func(t *testing.T) {
		cfg := config.Default(root)
		cfg.Board.Columns = []config.ColumnConfig{
			{ID: "only-one", Name: "Only One"},
		}
		err := config.Validate(cfg, root)
		if err == nil || !strings.Contains(err.Error(), "CFG-002") {
			t.Errorf("expected CFG-002 error, got: %v", err)
		}
	})

	t.Run("CFG-003: Duplicate column ID", func(t *testing.T) {
		cfg := config.Default(root)
		cfg.Board.Columns = []config.ColumnConfig{
			{ID: "col1", Name: "Column 1"},
			{ID: "col1", Name: "Column 1 Duplicate"},
		}
		err := config.Validate(cfg, root)
		if err == nil || !strings.Contains(err.Error(), "CFG-003") {
			t.Errorf("expected CFG-003 error, got: %v", err)
		}
	})

	t.Run("Column sort_by and sort_direction validation", func(t *testing.T) {
		cfg := config.Default(root)
		cfg.Board.Columns[0].SortBy = "non_existent_field"
		err := config.Validate(cfg, root)
		if err == nil || !strings.Contains(err.Error(), "invalid sort_by") {
			t.Errorf("expected invalid sort_by error, got: %v", err)
		}

		cfg = config.Default(root)
		cfg.Board.Columns[0].SortDirection = "diagonal"
		err = config.Validate(cfg, root)
		if err == nil || !strings.Contains(err.Error(), "invalid sort_direction") {
			t.Errorf("expected invalid sort_direction error, got: %v", err)
		}

		cfg = config.Default(root)
		cfg.Board.Columns[0].SortBy = "target_at"
		cfg.Board.Columns[0].SortDirection = "asc"
		if err := config.Validate(cfg, root); err != nil {
			t.Errorf("expected valid sort_by and sort_direction to pass, got: %v", err)
		}
	})

	t.Run("CFG-004: Invalid port range", func(t *testing.T) {
		cfg := config.Default(root)
		cfg.Server.Port = 80 // Privileged port < 1024
		err := config.Validate(cfg, root)
		if err == nil || !strings.Contains(err.Error(), "CFG-004") {
			t.Errorf("expected CFG-004 error for port 80, got: %v", err)
		}

		cfg.Server.Port = 70000 // > 65535
		err = config.Validate(cfg, root)
		if err == nil || !strings.Contains(err.Error(), "CFG-004") {
			t.Errorf("expected CFG-004 error for port 70000, got: %v", err)
		}
	})

	t.Run("CFG-005: Path escaping root", func(t *testing.T) {
		cfg := config.Default(root)
		cfg.Paths.Tasks = "../../etc/passwd"
		err := config.Validate(cfg, root)
		if err == nil || !strings.Contains(err.Error(), "CFG-005") {
			t.Errorf("expected CFG-005 error for escaping path, got: %v", err)
		}
	})

	t.Run("TAG-001: Invalid tag format", func(t *testing.T) {
		cfg := config.Default(root)
		cfg.Tags.Allowed = []string{"valid-tag", "INVALID_TAG", "has space"}
		err := config.Validate(cfg, root)
		if err == nil || !strings.Contains(err.Error(), "TAG-001") {
			t.Errorf("expected TAG-001 error for uppercase/spaced tags, got: %v", err)
		}
	})

	t.Run("TAG-002: Duplicate tag", func(t *testing.T) {
		cfg := config.Default(root)
		cfg.Tags.Allowed = []string{"backend", "frontend", "backend"}
		err := config.Validate(cfg, root)
		if err == nil || !strings.Contains(err.Error(), "TAG-002") {
			t.Errorf("expected TAG-002 error for duplicate tag, got: %v", err)
		}
	})

	t.Run("Invalid column color format", func(t *testing.T) {
		cfg := config.Default(root)
		cfg.Board.Columns[0].Color = "blue" // Not a hex color
		err := config.Validate(cfg, root)
		if err == nil || !strings.Contains(err.Error(), "invalid color hex") {
			t.Errorf("expected error for non-hex color, got: %v", err)
		}
	})
}

func TestLoadTranslationsTOML(t *testing.T) {
	tempDir := t.TempDir()
	jokatekoDir := filepath.Join(tempDir, ".jokateko")
	if err := os.MkdirAll(jokatekoDir, 0755); err != nil {
		t.Fatal(err)
	}

	tomlContent := `
version = "0"

[project]
name = "Localized App"

[translations]
board = "Taulu"
strategies = "Strategiat"
glossary = "Sanasto"
search = "Hae"
tasks = "tehtävää"
architectural_strategies = "Arkkitehtuuristrategiat"
strategies_subtitle = "Porrastetut ohjeet ja järjestelmäsuunnittelun säännöt"
tiers = "Tasot"
all_tiers = "Kaikki tasot"
project_glossary = "Projektin sanasto"
glossary_subtitle = "Standardoidut termimääritelmät ja sanakirja"
footer_text = "Rakennettu ❤️ 🇪🇺 kera 🤖"
sort_by = "Järjestä"
sort_default = "Oletus (Työnkulku)"
arial_main_nav = "Päänavigointi"
arial_search = "Hae sivustolta"
arial_search_input = "Hae tehtäviä, virstanpylväitä, strategioita ja sanastoa"
arial_sort_tasks = "Järjestä tehtävät"
`

	if err := os.WriteFile(filepath.Join(jokatekoDir, "config.toml"), []byte(tomlContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(tempDir)
	if err != nil {
		t.Fatalf("failed to load config with translations: %v", err)
	}

	if cfg.Translations.Board != "Taulu" {
		t.Errorf("expected Translations.Board to be 'Taulu', got %q", cfg.Translations.Board)
	}
	if cfg.Translations.Strategies != "Strategiat" {
		t.Errorf("expected Translations.Strategies to be 'Strategiat', got %q", cfg.Translations.Strategies)
	}
	if cfg.Translations.Glossary != "Sanasto" {
		t.Errorf("expected Translations.Glossary to be 'Sanasto', got %q", cfg.Translations.Glossary)
	}
	if cfg.Translations.FooterText != "Rakennettu ❤️ 🇪🇺 kera 🤖" {
		t.Errorf("expected Translations.FooterText to be 'Rakennettu ❤️ 🇪🇺 kera 🤖', got %q", cfg.Translations.FooterText)
	}
	if cfg.Translations.SortBy != "Järjestä" {
		t.Errorf("expected Translations.SortBy to be 'Järjestä', got %q", cfg.Translations.SortBy)
	}
	if cfg.Translations.SortDefault != "Oletus (Työnkulku)" {
		t.Errorf("expected Translations.SortDefault to be 'Oletus (Työnkulku)', got %q", cfg.Translations.SortDefault)
	}
	if cfg.Translations.ArialMainNav != "Päänavigointi" {
		t.Errorf("expected Translations.ArialMainNav to be 'Päänavigointi', got %q", cfg.Translations.ArialMainNav)
	}
	if cfg.Translations.ArialSortTasks != "Järjestä tehtävät" {
		t.Errorf("expected Translations.ArialSortTasks to be 'Järjestä tehtävät', got %q", cfg.Translations.ArialSortTasks)
	}
}

func TestCustomPrioritiesTOML(t *testing.T) {
	tempDir := t.TempDir()
	jokatekoDir := filepath.Join(tempDir, ".jokateko")
	if err := os.MkdirAll(jokatekoDir, 0755); err != nil {
		t.Fatal(err)
	}

	tomlContent := `
version = "0"

[project]
name = "Custom Priorities App"

[[priorities]]
id = "p0"
name = "Blocker"
color = "#ef4444"

[[priorities]]
id = "p1"
name = "Critical"
color = "#f97316"

[[priorities]]
id = "p2"
name = "Normal"
color = "#3b82f6"
`

	if err := os.WriteFile(filepath.Join(jokatekoDir, "config.toml"), []byte(tomlContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(tempDir)
	if err != nil {
		t.Fatalf("failed to load config with custom priorities: %v", err)
	}

	if len(cfg.Priorities) != 3 {
		t.Fatalf("expected 3 custom priorities, got %d", len(cfg.Priorities))
	}
	if cfg.Priorities[0].ID != "p0" || cfg.Priorities[0].Name != "Blocker" {
		t.Errorf("unexpected priority 0: %+v", cfg.Priorities[0])
	}
	if !cfg.HasPriority("p0") || !cfg.HasPriority("p1") || !cfg.HasPriority("p2") {
		t.Error("expected custom priorities p0, p1, p2 to exist")
	}
	if cfg.HasPriority("critical") {
		t.Error("expected default 'critical' priority to be replaced")
	}
}

func TestValidatePriorities(t *testing.T) {
	root := t.TempDir()

	t.Run("Empty priorities list error", func(t *testing.T) {
		cfg := config.Default(root)
		cfg.Priorities = nil
		err := config.Validate(cfg, root)
		if err == nil || !strings.Contains(err.Error(), "CFG-006") {
			t.Errorf("expected CFG-006 error for empty priorities, got: %v", err)
		}
	})

	t.Run("Duplicate priority ID error", func(t *testing.T) {
		cfg := config.Default(root)
		cfg.Priorities = append(cfg.Priorities, model.PriorityConfig{ID: "low", Name: "Duplicate Low"})
		err := config.Validate(cfg, root)
		if err == nil || !strings.Contains(err.Error(), "CFG-007") {
			t.Errorf("expected CFG-007 error for duplicate priority ID, got: %v", err)
		}
	})

	t.Run("Invalid priority hex color error", func(t *testing.T) {
		cfg := config.Default(root)
		cfg.Priorities[0].Color = "not-a-hex"
		err := config.Validate(cfg, root)
		if err == nil || !strings.Contains(err.Error(), "invalid color hex") {
			t.Errorf("expected invalid color hex error, got: %v", err)
		}
	})
}

func TestLoadCustomStrategyTiers(t *testing.T) {
	tempDir := t.TempDir()
	jokatekoDir := filepath.Join(tempDir, ".jokateko")
	if err := os.MkdirAll(jokatekoDir, 0755); err != nil {
		t.Fatal(err)
	}

	tomlContent := `
[[strategies.tiers]]
id = "core"
name = "Core"
title = "Critical Invariants"
summary = "Zero CGO and markdown tasks-as-code"
color = "#3b82f6"

[[strategies.tiers]]
id = "patterns"
name = "Patterns"
title = "Architecture Patterns"
summary = "Progressive disclosure domain patterns"
color = "#a855f7"
`

	if err := os.WriteFile(filepath.Join(jokatekoDir, "config.toml"), []byte(tomlContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(tempDir)
	if err != nil {
		t.Fatalf("failed to load config with custom strategy tiers: %v", err)
	}

	if len(cfg.Strategies.Tiers) != 2 {
		t.Fatalf("expected 2 custom tiers, got %d", len(cfg.Strategies.Tiers))
	}
	if cfg.Strategies.Tiers[0].ID != "core" || cfg.Strategies.Tiers[0].Title != "Critical Invariants" {
		t.Errorf("unexpected tier 0: %+v", cfg.Strategies.Tiers[0])
	}
	if !cfg.HasTier("core") || !cfg.HasTier("patterns") {
		t.Error("expected custom tiers 'core' and 'patterns' to exist")
	}
	if cfg.HasTier("1") {
		t.Error("expected default tier '1' to be replaced")
	}
	tr, ok := cfg.GetTier("core")
	if !ok || tr.Name != "Core" {
		t.Errorf("GetTier('core') failed: %+v", tr)
	}
}

func TestLoadTopLevelTiersAlias(t *testing.T) {
	tempDir := t.TempDir()
	jokatekoDir := filepath.Join(tempDir, ".jokateko")
	if err := os.MkdirAll(jokatekoDir, 0755); err != nil {
		t.Fatal(err)
	}

	tomlContent := `
[[tiers]]
id = "t1"
name = "Tier 1"
title = "Base"
summary = "Base rules"
color = "#10b981"
`

	if err := os.WriteFile(filepath.Join(jokatekoDir, "config.toml"), []byte(tomlContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(tempDir)
	if err != nil {
		t.Fatalf("failed to load config with [[tiers]] alias: %v", err)
	}

	if len(cfg.Strategies.Tiers) != 1 || cfg.Strategies.Tiers[0].ID != "t1" {
		t.Fatalf("expected 1 tier with ID 't1', got %+v", cfg.Strategies.Tiers)
	}
}

func TestValidateStrategyTiers(t *testing.T) {
	root := t.TempDir()

	t.Run("Empty strategy tiers list error", func(t *testing.T) {
		cfg := config.Default(root)
		cfg.Strategies.Tiers = nil
		err := config.Validate(cfg, root)
		if err == nil || !strings.Contains(err.Error(), "CFG-008") {
			t.Errorf("expected CFG-008 error for empty strategy tiers, got: %v", err)
		}
	})

	t.Run("Duplicate tier ID error", func(t *testing.T) {
		cfg := config.Default(root)
		cfg.Strategies.Tiers = append(cfg.Strategies.Tiers, model.TierConfig{ID: "1", Name: "Duplicate Tier 1"})
		err := config.Validate(cfg, root)
		if err == nil || !strings.Contains(err.Error(), "CFG-009") {
			t.Errorf("expected CFG-009 error for duplicate tier ID, got: %v", err)
		}
	})

	t.Run("Invalid tier hex color error", func(t *testing.T) {
		cfg := config.Default(root)
		cfg.Strategies.Tiers[0].Color = "invalid-hex"
		err := config.Validate(cfg, root)
		if err == nil || !strings.Contains(err.Error(), "invalid color hex") {
			t.Errorf("expected invalid color hex error, got: %v", err)
		}
	})
}

func TestGenerateCommentedConfig(t *testing.T) {
	projectName := "Custom Project Alpha"
	projectDesc := "A custom test description for testing"

	commented := config.GenerateCommentedConfig(projectName, projectDesc)

	// Verify header contains version = "0"
	if !strings.Contains(commented, `version = "0"`) {
		t.Error("expected commented config to contain active version = \"0\"")
	}

	// Verify project section is active
	if !strings.Contains(commented, `name = "Custom Project Alpha"`) {
		t.Errorf("expected project name to be active and set, got:\n%s", commented)
	}
	if !strings.Contains(commented, `description = "A custom test description for testing"`) {
		t.Errorf("expected project description to be active and set, got:\n%s", commented)
	}

	// Verify other sections are commented out
	expectedCommented := []string{
		"# [paths]",
		"# tasks = ",
		"# [server]",
		"# host = ",
		"# port = ",
		"# [[board.columns]]",
		"# [tags]",
		"# [mcp]",
		"# enabled = ",
	}
	for _, exp := range expectedCommented {
		if !strings.Contains(commented, exp) {
			t.Errorf("expected commented config to contain %q, but was missing or uncommented", exp)
		}
	}

	// Verify that loading this generated config via config.Load produces a valid *Config
	tempDir := t.TempDir()
	jokatekoDir := filepath.Join(tempDir, ".jokateko")
	if err := os.MkdirAll(jokatekoDir, 0755); err != nil {
		t.Fatal(err)
	}
	configFile := filepath.Join(jokatekoDir, "config.toml")
	if err := os.WriteFile(configFile, []byte(commented), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(tempDir)
	if err != nil {
		t.Fatalf("config.Load failed on generated commented config: %v", err)
	}

	if cfg.Project.Name != projectName {
		t.Errorf("expected project name %q, got %q", projectName, cfg.Project.Name)
	}
	if cfg.Project.Description != projectDesc {
		t.Errorf("expected project description %q, got %q", projectDesc, cfg.Project.Description)
	}

	// Inherits standard defaults for omitted/commented sections
	if cfg.Server.Port != 8080 {
		t.Errorf("expected default port 8080, got %d", cfg.Server.Port)
	}
	if len(cfg.Board.Columns) != 5 {
		t.Errorf("expected 5 default columns, got %d", len(cfg.Board.Columns))
	}
	if len(cfg.Tags.Allowed) != 11 {
		t.Errorf("expected 11 default tags, got %d", len(cfg.Tags.Allowed))
	}
	if !cfg.MCP.Enabled {
		t.Error("expected default MCP.Enabled to be true")
	}
	if !strings.Contains(cfg.MCP.Instructions, "Jokateko manages tasks") {
		t.Errorf("expected default MCP instructions to be inherited, got: %q", cfg.MCP.Instructions)
	}
}

func TestMCPInstructionsConfig(t *testing.T) {
	tempDir := t.TempDir()
	jokatekoDir := filepath.Join(tempDir, ".jokateko")
	if err := os.MkdirAll(jokatekoDir, 0755); err != nil {
		t.Fatal(err)
	}

	customTOML := `version = "0"

[project]
name = "MCP Instructions Test"
description = "Testing MCP custom instructions"

[mcp]
instructions = """
Always inspect Tier-1 strategies.
Never edit .jokateko files directly.
"""
`
	if err := os.WriteFile(filepath.Join(jokatekoDir, "config.toml"), []byte(customTOML), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(tempDir)
	if err != nil {
		t.Fatalf("config.Load failed: %v", err)
	}

	expectedGuidance := "Always inspect Tier-1 strategies.\nNever edit .jokateko files directly.\n"
	if cfg.MCP.Instructions != expectedGuidance {
		t.Errorf("expected custom instructions %q, got %q", expectedGuidance, cfg.MCP.Instructions)
	}
}

func TestBoardEditableStates(t *testing.T) {
	// 1. Default config should have editable_states = ["backlog"]
	defCfg := config.Default("")
	if len(defCfg.Board.EditableStates) != 1 || defCfg.Board.EditableStates[0] != "backlog" {
		t.Errorf("expected default editable_states to be ['backlog'], got %v", defCfg.Board.EditableStates)
	}
	if !defCfg.IsTaskEditable("backlog") {
		t.Error("expected backlog to be editable in default config")
	}
	if defCfg.IsTaskEditable("ready") {
		t.Error("expected ready to not be editable in default config")
	}
	if defCfg.IsTaskEditable("in_progress") {
		t.Error("expected in_progress to not be editable in default config")
	}

	// 2. Custom config with multiple editable states
	tempDir := t.TempDir()
	jokatekoDir := filepath.Join(tempDir, ".jokateko")
	_ = os.MkdirAll(jokatekoDir, 0755)

	customTOML := `
[board]
editable_states = ["backlog", "ready", "custom_draft"]
`
	_ = os.WriteFile(filepath.Join(jokatekoDir, "config.toml"), []byte(customTOML), 0644)

	cfg, err := config.Load(tempDir)
	if err != nil {
		t.Fatalf("failed to load custom config: %v", err)
	}
	if !cfg.IsTaskEditable("backlog") {
		t.Error("expected backlog to be editable")
	}
	if !cfg.IsTaskEditable("ready") {
		t.Error("expected ready to be editable")
	}
	if !cfg.IsTaskEditable("custom_draft") {
		t.Error("expected custom_draft to be editable")
	}
	if cfg.IsTaskEditable("in_progress") {
		t.Error("expected in_progress to not be editable")
	}
}

func TestBoardCreatableAndDefaultCreateState(t *testing.T) {
	// 1. Default config
	defCfg := config.Default("")
	if len(defCfg.Board.CreatableStates) != 1 || defCfg.Board.CreatableStates[0] != "backlog" {
		t.Errorf("expected default creatable_states to be ['backlog'], got %v", defCfg.Board.CreatableStates)
	}
	if defCfg.Board.DefaultCreateState != "backlog" {
		t.Errorf("expected default default_create_state to be 'backlog', got %q", defCfg.Board.DefaultCreateState)
	}
	if !defCfg.IsTaskCreatable("backlog") {
		t.Error("expected backlog to be creatable in default config")
	}
	if defCfg.IsTaskCreatable("ready") {
		t.Error("expected ready to not be creatable in default config")
	}
	if defCfg.DefaultCreateState() != "backlog" {
		t.Errorf("expected DefaultCreateState() to be 'backlog', got %q", defCfg.DefaultCreateState())
	}

	// 2. Custom config
	tempDir := t.TempDir()
	jokatekoDir := filepath.Join(tempDir, ".jokateko")
	_ = os.MkdirAll(jokatekoDir, 0755)

	customTOML := `
[board]
creatable_states = ["backlog", "ready"]
default_create_state = "ready"
`
	_ = os.WriteFile(filepath.Join(jokatekoDir, "config.toml"), []byte(customTOML), 0644)

	cfg, err := config.Load(tempDir)
	if err != nil {
		t.Fatalf("failed to load custom config: %v", err)
	}
	if !cfg.IsTaskCreatable("backlog") {
		t.Error("expected backlog to be creatable")
	}
	if !cfg.IsTaskCreatable("ready") {
		t.Error("expected ready to be creatable")
	}
	if cfg.IsTaskCreatable("in_progress") {
		t.Error("expected in_progress to not be creatable")
	}
	if cfg.DefaultCreateState() != "ready" {
		t.Errorf("expected DefaultCreateState() to be 'ready', got %q", cfg.DefaultCreateState())
	}
}

func TestColumnWorkflowRoleAndInstructions(t *testing.T) {
	tempDir := t.TempDir()
	jokatekoDir := filepath.Join(tempDir, ".jokateko")
	_ = os.MkdirAll(jokatekoDir, 0755)

	customTOML := `
version = "0"

[project]
name = "Workflow Test"

[[board.columns]]
id = "backlog"
name = "Backlog"
color = "#94a3b8"
handled_by = "human"
instructions = "Triage and spec definition"

[[board.columns]]
id = "done"
name = "Done"
color = "#10b981"
handled_by = "agent:qa"
instructions = "Verify acceptance criteria"
`
	_ = os.WriteFile(filepath.Join(jokatekoDir, "config.toml"), []byte(customTOML), 0644)

	cfg, err := config.Load(tempDir)
	if err != nil {
		t.Fatalf("failed to load custom workflow config: %v", err)
	}

	if len(cfg.Board.Columns) != 2 {
		t.Fatalf("expected 2 columns, got %d", len(cfg.Board.Columns))
	}
	if cfg.Board.Columns[0].HandledBy != "human" {
		t.Errorf("expected backlog handled_by to be 'human', got %q", cfg.Board.Columns[0].HandledBy)
	}
	if cfg.Board.Columns[0].Instructions != "Triage and spec definition" {
		t.Errorf("expected backlog instructions, got %q", cfg.Board.Columns[0].Instructions)
	}
	if cfg.Board.Columns[1].HandledBy != "agent:qa" {
		t.Errorf("expected done handled_by to be 'agent:qa', got %q", cfg.Board.Columns[1].HandledBy)
	}
}

func TestValidateBoardCreatablePolicies(t *testing.T) {
	root := t.TempDir()

	t.Run("Invalid creatable_states column", func(t *testing.T) {
		cfg := config.Default(root)
		cfg.Board.CreatableStates = []string{"nonexistent_column"}
		err := config.Validate(cfg, root)
		if err == nil || !strings.Contains(err.Error(), "board.creatable_states contains unknown column") {
			t.Errorf("expected error for unknown creatable column, got: %v", err)
		}
	})

	t.Run("Invalid default_create_state column", func(t *testing.T) {
		cfg := config.Default(root)
		cfg.Board.DefaultCreateState = "unknown_column"
		err := config.Validate(cfg, root)
		if err == nil || !strings.Contains(err.Error(), "board.default_create_state \"unknown_column\" is not a defined column") {
			t.Errorf("expected error for unknown default_create_state, got: %v", err)
		}
	})
}






